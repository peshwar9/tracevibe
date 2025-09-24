package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/peshwar9/tracevibe/internal/database"
	"github.com/spf13/cobra"
)

//go:embed web/templates/*.html
var templatesFS embed.FS

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web-based admin UI server",
	Long: `Start the TraceVibe web server to view RTM data through a browser interface.

The server provides:
- Dashboard overview of all projects
- Project details with component breakdown
- Hierarchical requirements view (Scope -> User Stories -> Tech Specs)
- Implementation and test case traceability
- Interactive navigation through the requirements tree

Access the UI at http://localhost:8080 (default port).`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		dbPath, _ := cmd.Flags().GetString("db-path")
		projectBasePath, _ := cmd.Flags().GetString("project-base-path")

		if err := startServer(port, dbPath, projectBasePath); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	serveCmd.Flags().StringP("db-path", "d", getDefaultDBPath(), "SQLite database path")
	serveCmd.Flags().String("project-base-path", "", "Base path for resolving test file paths (e.g., /path/to/project/)")
}

func startServer(port int, dbPath string, projectBasePath string) error {
	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Parse all templates
	tmpl, err := template.ParseFS(templatesFS, "web/templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}

	// Get project base path from environment if not provided via flag
	if projectBasePath == "" {
		projectBasePath = os.Getenv("TRACEVIBE_PROJECT_BASE_PATH")
	}

	server := &Server{
		db:              db,
		templates:       tmpl,
		projectBasePath: projectBasePath,
	}

	// Routes
	http.HandleFunc("/", server.dashboardHandler)
	http.HandleFunc("/projects/", server.projectHandler)
	http.HandleFunc("/api/test/run", server.testRunHandler)
	http.HandleFunc("/api/project/", server.projectAPIHandler)
	http.HandleFunc("/api/", server.apiHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🚀 TraceVibe server starting on http://localhost%s\n", addr)
	fmt.Printf("📊 Database: %s\n", dbPath)
	fmt.Printf("🔍 Open your browser and navigate to the URL above\n")

	return http.ListenAndServe(addr, nil)
}

type Server struct {
	db              *database.DB
	templates       *template.Template
	projectBasePath string
}

// Dashboard handler
func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Title             string
		Projects          []ProjectSummary
		TotalComponents   int
		TotalRequirements int
		TotalTestCases    int
		Error             string
	}{
		Title: "Dashboard",
	}

	// Get all projects with summary data
	projects, err := s.getProjectsSummary()
	if err != nil {
		data.Error = fmt.Sprintf("Error loading projects: %v", err)
	} else {
		data.Projects = projects
		for _, p := range projects {
			data.TotalComponents += p.ComponentCount
			data.TotalRequirements += p.RequirementCount
			data.TotalTestCases += p.TestCaseCount
		}
	}

	s.renderTemplate(w, "dashboard-page.html", data)
}

// Project handler
func (s *Server) projectHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/projects/"):]

	// Parse path: /projects/{projectKey} or /projects/{projectKey}/components/{componentKey}
	pathParts := splitPath(path)
	if len(pathParts) == 0 {
		http.NotFound(w, r)
		return
	}

	projectKey := pathParts[0]

	if len(pathParts) == 1 {
		// Project overview
		s.projectOverviewHandler(w, r, projectKey)
	} else if len(pathParts) == 3 && pathParts[1] == "components" {
		// Component details
		componentKey := pathParts[2]
		s.componentDetailsHandler(w, r, projectKey, componentKey)
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) projectOverviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	data := struct {
		Title          string
		Project        *database.Project
		Components     []ComponentSummary
		Requirements   []RequirementTree
		ScopeCount     int
		UserStoryCount int
		TechSpecCount  int
		TotalTestCount int
		Error          string
	}{
		Title: "Project Overview",
	}

	// Get project
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil {
		data.Error = fmt.Sprintf("Error loading project: %v", err)
		s.renderTemplate(w, "project-page.html", data)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}
	data.Project = project

	// Get components summary
	components, err := s.getComponentsSummary(projectKey)
	if err != nil {
		data.Error = fmt.Sprintf("Error loading components: %v", err)
	} else {
		data.Components = components
		// Calculate total test count from components
		for _, comp := range components {
			data.TotalTestCount += comp.TestCaseCount
		}
	}

	// Get requirements tree
	requirements, err := s.getRequirementsTree(projectKey, "")
	if err != nil {
		data.Error = fmt.Sprintf("Error loading requirements: %v", err)
	} else {
		data.Requirements = requirements
		// Count by type
		for _, req := range requirements {
			countRequirementsByType(req, &data.ScopeCount, &data.UserStoryCount, &data.TechSpecCount)
		}
	}

	s.renderTemplate(w, "project-page.html", data)
}

func (s *Server) componentDetailsHandler(w http.ResponseWriter, r *http.Request, projectKey, componentKey string) {
	data := struct {
		Title          string
		Project        *database.Project
		Component      *ComponentSummary
		Requirements   []RequirementTree
		ScopeCount     int
		UserStoryCount int
		TechSpecCount  int
		TestCaseCount  int
		Error          string
	}{
		Title: "Component Details",
	}

	// Get project
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil {
		data.Error = fmt.Sprintf("Error loading project: %v", err)
		s.renderTemplate(w, "component-page.html", data)
		return
	}
	if project == nil {
		http.NotFound(w, r)
		return
	}
	data.Project = project

	// Get component
	component, err := s.getComponentByKey(projectKey, componentKey)
	if err != nil {
		data.Error = fmt.Sprintf("Error loading component: %v", err)
		s.renderTemplate(w, "component-page.html", data)
		return
	}
	if component == nil {
		http.NotFound(w, r)
		return
	}
	data.Component = component

	// Get requirements tree for this component
	requirements, err := s.getRequirementsTree(projectKey, componentKey)
	if err != nil {
		data.Error = fmt.Sprintf("Error loading requirements: %v", err)
	} else {
		data.Requirements = requirements
		// Count by type and test cases
		for _, req := range requirements {
			countRequirementsByType(req, &data.ScopeCount, &data.UserStoryCount, &data.TechSpecCount)
			data.TestCaseCount += countTestCases(req)
		}
	}

	s.renderTemplate(w, "component-page.html", data)
}

// API handler for AJAX requests
func (s *Server) apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Simple API endpoints can be added here for dynamic updates
	// For now, return basic server info
	response := map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	}

	json.NewEncoder(w).Encode(response)
}

// Test runner handler
func (s *Server) testRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Project   string `json:"project"`
		Component string `json:"component"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := s.runTestsForComponent(req.Project, req.Component)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

// Project API handler for delete operations
func (s *Server) projectAPIHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/project/"):]
	parts := splitPath(path)

	if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodDelete {
		projectKey := parts[0]
		s.deleteProjectHandler(w, r, projectKey)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func (s *Server) deleteProjectHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	// Get project ID first
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error finding project: %v", err), http.StatusInternalServerError)
		return
	}
	if project == nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Delete project and all related data (cascading delete)
	err = s.deleteProject(project.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting project: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": fmt.Sprintf("Project %s deleted successfully", projectKey),
	})
}

func (s *Server) deleteProject(projectID string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete in correct order to respect foreign key constraints
	// 1. Delete requirement_test_coverage
	_, err = tx.Exec(`DELETE FROM requirement_test_coverage WHERE requirement_id IN
		(SELECT id FROM requirements WHERE project_id = ?)`, projectID)
	if err != nil {
		return err
	}

	// 2. Delete test_cases
	_, err = tx.Exec(`DELETE FROM test_cases WHERE test_file_id IN
		(SELECT id FROM test_files WHERE project_id = ?)`, projectID)
	if err != nil {
		return err
	}

	// 3. Delete test_files
	_, err = tx.Exec(`DELETE FROM test_files WHERE project_id = ?`, projectID)
	if err != nil {
		return err
	}

	// 4. Delete implementations
	_, err = tx.Exec(`DELETE FROM implementations WHERE requirement_id IN
		(SELECT id FROM requirements WHERE project_id = ?)`, projectID)
	if err != nil {
		return err
	}

	// 5. Delete requirements
	_, err = tx.Exec(`DELETE FROM requirements WHERE project_id = ?`, projectID)
	if err != nil {
		return err
	}

	// 6. Delete system_components
	_, err = tx.Exec(`DELETE FROM system_components WHERE project_id = ?`, projectID)
	if err != nil {
		return err
	}

	// 7. Delete project
	_, err = tx.Exec(`DELETE FROM projects WHERE id = ?`, projectID)
	if err != nil {
		return err
	}

	// Commit transaction
	return tx.Commit()
}

func (s *Server) runTestsForComponent(projectKey, componentKey string) (*TestResult, error) {
	// Get test files for this component
	testFiles, err := s.getTestFilesForComponent(projectKey, componentKey)
	if err != nil {
		// Return a valid result with error message instead of error
		return &TestResult{
			Passed:   0,
			Failed:   0,
			Duration: "0s",
			Output:   fmt.Sprintf("Error accessing test files: %v", err),
		}, nil
	}

	if len(testFiles) == 0 {
		return &TestResult{
			Passed:   0,
			Failed:   0,
			Duration: "0s",
			Output:   fmt.Sprintf("No test files found for component '%s' in project '%s'.\n\nTo add test files, include them in your RTM JSON with test_cases entries.", componentKey, projectKey),
		}, nil
	}

	// Run tests and collect results
	startTime := time.Now()
	var outputs []string
	passed := 0
	failed := 0
	skipped := 0

	for _, testFile := range testFiles {
		// Resolve test file path using project base path
		fullTestPath := testFile
		if s.projectBasePath != "" {
			fullTestPath = filepath.Join(s.projectBasePath, testFile)
		}

		// Check if test file actually exists
		if _, err := os.Stat(fullTestPath); os.IsNotExist(err) {
			skipped++
			if s.projectBasePath != "" {
				outputs = append(outputs, fmt.Sprintf("Skipping %s: File does not exist at %s", testFile, fullTestPath))
			} else {
				outputs = append(outputs, fmt.Sprintf("Skipping %s: File does not exist (set TRACEVIBE_PROJECT_BASE_PATH or use --project-base-path)", testFile))
			}
			continue
		}

		success, output, err := s.runTestFile(fullTestPath, s.projectBasePath)
		outputs = append(outputs, fmt.Sprintf("Running tests in %s:\n%s", testFile, output))

		if err != nil {
			failed++
			outputs = append(outputs, fmt.Sprintf("ERROR: %v", err))
		} else if success {
			passed++
		} else {
			failed++
		}
	}

	duration := time.Since(startTime)

	// Create summary message
	var summaryParts []string
	if passed > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("✓ %d tests passed", passed))
	}
	if failed > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("✗ %d tests failed", failed))
	}
	if skipped > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("⚠ %d test files skipped (RTM references only)", skipped))
	}

	summary := strings.Join(summaryParts, ", ")
	if summary != "" {
		outputs = append([]string{summary + "\n"}, outputs...)
	}

	return &TestResult{
		Passed:   passed,
		Failed:   failed,
		Duration: duration.Round(time.Millisecond).String(),
		Output:   strings.Join(outputs, "\n\n"),
	}, nil
}

func (s *Server) getTestFilesForComponent(projectKey, componentKey string) ([]string, error) {
	query := `
		SELECT DISTINCT tf.file_path
		FROM test_files tf
		JOIN test_cases tc ON tf.id = tc.test_file_id
		JOIN requirement_test_coverage rtc ON tc.id = rtc.test_case_id
		JOIN requirements r ON rtc.requirement_id = r.id
		JOIN system_components c ON r.component_id = c.id
		JOIN projects p ON tf.project_id = p.id
		WHERE p.project_key = ? AND c.component_key = ?
		ORDER BY tf.file_path`

	rows, err := s.db.Query(query, projectKey, componentKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testFiles []string
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			continue
		}
		testFiles = append(testFiles, filePath)
	}

	return testFiles, nil
}

func (s *Server) runTestFile(testFile, projectBasePath string) (bool, string, error) {
	// Determine test runner based on file extension and run appropriate commands
	var cmd *exec.Cmd
	var workingDir string

	switch {
	case strings.HasSuffix(testFile, "_test.go"):
		// Go tests: run the package directory, not the individual file
		packageDir := filepath.Dir(testFile)

		// Set working directory to project base path if available, otherwise current dir
		if projectBasePath != "" {
			workingDir = projectBasePath
			// Make packageDir relative to project base path
			if relPath, err := filepath.Rel(projectBasePath, packageDir); err == nil {
				packageDir = relPath
			}
		} else {
			workingDir = "."
		}
		cmd = exec.Command("go", "test", "-v", "./"+packageDir)

	case strings.HasSuffix(testFile, ".test.js") || strings.HasSuffix(testFile, ".spec.js") ||
		 strings.HasSuffix(testFile, ".test.ts") || strings.HasSuffix(testFile, ".spec.ts"):
		// JavaScript/TypeScript tests: use npm test or jest directly
		if projectBasePath != "" {
			// Start from project base path and look for package.json
			workingDir = projectBasePath
			packageDir := filepath.Dir(testFile)
			for packageDir != "." && packageDir != "/" && packageDir != projectBasePath {
				if _, err := os.Stat(filepath.Join(packageDir, "package.json")); err == nil {
					workingDir = packageDir
					break
				}
				packageDir = filepath.Dir(packageDir)
			}
		} else {
			// Try to detect if it's a frontend project by checking for package.json
			packageDir := filepath.Dir(testFile)
			for packageDir != "." && packageDir != "/" {
				if _, err := os.Stat(filepath.Join(packageDir, "package.json")); err == nil {
					workingDir = packageDir
					break
				}
				packageDir = filepath.Dir(packageDir)
			}

			if workingDir == "" {
				workingDir = "."
			}
		}

		// Use jest to run specific test file
		relativeTestFile := testFile
		if workingDir != "." {
			if rel, err := filepath.Rel(workingDir, testFile); err == nil {
				relativeTestFile = rel
			}
		}
		cmd = exec.Command("npx", "jest", relativeTestFile, "--verbose")

	case strings.HasSuffix(testFile, ".test.py") || strings.HasSuffix(testFile, "_test.py") || strings.HasSuffix(testFile, "test_*.py"):
		// Python tests: use pytest
		if projectBasePath != "" {
			workingDir = projectBasePath
			// Make test file relative to project base path
			if relPath, err := filepath.Rel(projectBasePath, testFile); err == nil {
				cmd = exec.Command("python", "-m", "pytest", "-v", relPath)
			} else {
				cmd = exec.Command("python", "-m", "pytest", "-v", testFile)
			}
		} else {
			workingDir = "."
			cmd = exec.Command("python", "-m", "pytest", "-v", testFile)
		}

	default:
		return false, "", fmt.Errorf("unsupported test file format: %s (supported: Go _test.go, JS/TS .test/.spec files, Python .test.py/_test.py/test_*.py)", testFile)
	}

	// Set working directory if specified
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Add command info to output for debugging
	cmdInfo := fmt.Sprintf("Command: %s\nWorking Dir: %s\n\n", strings.Join(cmd.Args, " "), workingDir)
	outputStr = cmdInfo + outputStr

	if err != nil {
		// Check if it's a test failure vs execution error
		exitCode := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		// Exit code 1 usually means test failures, not execution errors
		if exitCode == 1 && (strings.Contains(outputStr, "FAIL") || strings.Contains(outputStr, "failed")) {
			return false, outputStr, nil // Test ran but failed
		}
		return false, outputStr, fmt.Errorf("execution failed (exit code %d): %v", exitCode, err)
	}

	return true, outputStr, nil
}

// Helper methods

func (s *Server) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	if err := s.templates.ExecuteTemplate(w, templateName, data); err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}
}

func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}

	var parts []string
	start := 0
	for i, r := range path {
		if r == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}

// Data structures for templates

type ProjectSummary struct {
	ID                    string `json:"id"`
	ProjectKey            string `json:"project_key"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	Status                string `json:"status"`
	ComponentCount        int    `json:"component_count"`
	RequirementCount      int    `json:"requirement_count"`
	ScopeCount            int    `json:"scope_count"`
	UserStoryCount        int    `json:"user_story_count"`
	TechSpecCount         int    `json:"tech_spec_count"`
	TestCaseCount         int    `json:"test_case_count"`
	UnitTestCount         int    `json:"unit_test_count"`
	IntegrationTestCount  int    `json:"integration_test_count"`
	E2ETestCount          int    `json:"e2e_test_count"`
}

type ComponentSummary struct {
	ID               string   `json:"id"`
	ComponentKey     string   `json:"component_key"`
	Name             string   `json:"name"`
	ComponentType    string   `json:"component_type"`
	Technology       string   `json:"technology"`
	Description      string   `json:"description"`
	TotalRequirements int     `json:"total_requirements"`
	ScopeCount       int      `json:"scope_count"`
	UserStoryCount   int      `json:"user_story_count"`
	TechSpecCount    int      `json:"tech_spec_count"`
	ImplementationCount int   `json:"implementation_count"`
	TestCaseCount    int      `json:"test_case_count"`
	ScopeIDs         []string `json:"scope_ids"`
}

type RequirementTree struct {
	ID               string            `json:"id"`
	RequirementKey   string            `json:"requirement_key"`
	RequirementType  string            `json:"requirement_type"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Category         string            `json:"category"`
	Status           string            `json:"status"`
	Priority         string            `json:"priority"`
	Children         []RequirementTree `json:"children"`
	Implementation   []ImplementationInfo `json:"implementation"`
	TestCases        []TestCaseInfo    `json:"test_cases"`
	UserStoryCount   int               `json:"user_story_count"`
	TechSpecCount    int               `json:"tech_spec_count"`
	TestCaseCount    int               `json:"test_case_count"`
}

type ImplementationInfo struct {
	Layer     string   `json:"layer"`
	FilePath  string   `json:"file_path"`
	Functions []string `json:"functions"`
}

type TestCaseInfo struct {
	FilePath string `json:"file_path"`
	TestName string `json:"test_name"`
	TestType string `json:"test_type"`
}

type TestResult struct {
	Passed   int    `json:"passed"`
	Failed   int    `json:"failed"`
	Duration string `json:"duration"`
	Output   string `json:"output"`
}

// Database query methods (these would need to be implemented in the database package)

func (s *Server) getProjectsSummary() ([]ProjectSummary, error) {
	query := `
		SELECT
			p.id, p.project_key, p.name, COALESCE(p.description, '') as description, p.status,
			COALESCE(comp_counts.component_count, 0) as component_count,
			COALESCE(req_counts.requirement_count, 0) as requirement_count,
			COALESCE(scope_counts.scope_count, 0) as scope_count,
			COALESCE(story_counts.story_count, 0) as story_count,
			COALESCE(tech_counts.tech_count, 0) as tech_count,
			COALESCE(test_counts.test_count, 0) as test_count,
			COALESCE(unit_test_counts.unit_count, 0) as unit_count,
			COALESCE(integration_test_counts.integration_count, 0) as integration_count,
			COALESCE(e2e_test_counts.e2e_count, 0) as e2e_count
		FROM projects p
		LEFT JOIN (
			SELECT project_id, COUNT(*) as component_count
			FROM system_components GROUP BY project_id
		) comp_counts ON p.id = comp_counts.project_id
		LEFT JOIN (
			SELECT project_id, COUNT(*) as requirement_count
			FROM requirements GROUP BY project_id
		) req_counts ON p.id = req_counts.project_id
		LEFT JOIN (
			SELECT project_id, COUNT(*) as scope_count
			FROM requirements
			WHERE requirement_type = 'SCOPE'
			GROUP BY project_id
		) scope_counts ON p.id = scope_counts.project_id
		LEFT JOIN (
			SELECT project_id, COUNT(*) as story_count
			FROM requirements
			WHERE requirement_type = 'USER_STORY'
			GROUP BY project_id
		) story_counts ON p.id = story_counts.project_id
		LEFT JOIN (
			SELECT project_id, COUNT(*) as tech_count
			FROM requirements
			WHERE requirement_type = 'TECH_SPEC'
			GROUP BY project_id
		) tech_counts ON p.id = tech_counts.project_id
		LEFT JOIN (
			SELECT p.id as project_id, COUNT(DISTINCT tc.id) as test_count
			FROM projects p
			LEFT JOIN test_files tf ON p.id = tf.project_id
			LEFT JOIN test_cases tc ON tf.id = tc.test_file_id
			GROUP BY p.id
		) test_counts ON p.id = test_counts.project_id
		LEFT JOIN (
			SELECT p.id as project_id, COUNT(DISTINCT tc.id) as unit_count
			FROM projects p
			LEFT JOIN test_files tf ON p.id = tf.project_id
			LEFT JOIN test_cases tc ON tf.id = tc.test_file_id
			WHERE tc.test_type = 'unit'
			GROUP BY p.id
		) unit_test_counts ON p.id = unit_test_counts.project_id
		LEFT JOIN (
			SELECT p.id as project_id, COUNT(DISTINCT tc.id) as integration_count
			FROM projects p
			LEFT JOIN test_files tf ON p.id = tf.project_id
			LEFT JOIN test_cases tc ON tf.id = tc.test_file_id
			WHERE tc.test_type = 'integration'
			GROUP BY p.id
		) integration_test_counts ON p.id = integration_test_counts.project_id
		LEFT JOIN (
			SELECT p.id as project_id, COUNT(DISTINCT tc.id) as e2e_count
			FROM projects p
			LEFT JOIN test_files tf ON p.id = tf.project_id
			LEFT JOIN test_cases tc ON tf.id = tc.test_file_id
			WHERE tc.test_type = 'e2e'
			GROUP BY p.id
		) e2e_test_counts ON p.id = e2e_test_counts.project_id
		ORDER BY p.name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ProjectSummary
	for rows.Next() {
		var p ProjectSummary
		err := rows.Scan(&p.ID, &p.ProjectKey, &p.Name, &p.Description, &p.Status,
			&p.ComponentCount, &p.RequirementCount, &p.ScopeCount, &p.UserStoryCount, &p.TechSpecCount, &p.TestCaseCount,
			&p.UnitTestCount, &p.IntegrationTestCount, &p.E2ETestCount)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, nil
}

func (s *Server) getComponentsSummary(projectKey string) ([]ComponentSummary, error) {
	query := `
		SELECT c.id, c.component_key, c.name, c.component_type,
			   COALESCE(c.technology, '') as technology, COALESCE(c.description, '') as description,
			   COALESCE(cs.total_requirements, 0), COALESCE(cs.scope_count, 0),
			   COALESCE(cs.user_story_count, 0), COALESCE(cs.tech_spec_count, 0),
			   COALESCE(cs.implementation_count, 0), COALESCE(cs.test_case_count, 0)
		FROM system_components c
		JOIN projects p ON c.project_id = p.id
		LEFT JOIN component_summary cs ON c.id = cs.id
		WHERE p.project_key = ?
		ORDER BY c.name`

	rows, err := s.db.Query(query, projectKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []ComponentSummary
	for rows.Next() {
		var c ComponentSummary
		err := rows.Scan(&c.ID, &c.ComponentKey, &c.Name, &c.ComponentType,
			&c.Technology, &c.Description, &c.TotalRequirements, &c.ScopeCount,
			&c.UserStoryCount, &c.TechSpecCount, &c.ImplementationCount, &c.TestCaseCount)
		if err != nil {
			return nil, err
		}

		// Get scope IDs for this component
		scopeQuery := `
			SELECT r.requirement_key
			FROM requirements r
			WHERE r.component_id = ? AND UPPER(r.requirement_type) = 'SCOPE'
			ORDER BY r.requirement_key`

		scopeRows, err := s.db.Query(scopeQuery, c.ID)
		if err == nil {
			defer scopeRows.Close()
			for scopeRows.Next() {
				var scopeID string
				if err := scopeRows.Scan(&scopeID); err == nil {
					c.ScopeIDs = append(c.ScopeIDs, scopeID)
				}
			}
		}

		components = append(components, c)
	}

	return components, nil
}

func (s *Server) getComponentByKey(projectKey, componentKey string) (*ComponentSummary, error) {
	query := `
		SELECT c.id, c.component_key, c.name, c.component_type,
			   COALESCE(c.technology, '') as technology, COALESCE(c.description, '') as description,
			   COALESCE(cs.total_requirements, 0), COALESCE(cs.scope_count, 0),
			   COALESCE(cs.user_story_count, 0), COALESCE(cs.tech_spec_count, 0),
			   COALESCE(cs.implementation_count, 0), COALESCE(cs.test_case_count, 0)
		FROM system_components c
		JOIN projects p ON c.project_id = p.id
		LEFT JOIN component_summary cs ON c.id = cs.id
		WHERE p.project_key = ? AND c.component_key = ?`

	var c ComponentSummary
	err := s.db.QueryRow(query, projectKey, componentKey).Scan(
		&c.ID, &c.ComponentKey, &c.Name, &c.ComponentType,
		&c.Technology, &c.Description, &c.TotalRequirements, &c.ScopeCount,
		&c.UserStoryCount, &c.TechSpecCount, &c.ImplementationCount, &c.TestCaseCount)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (s *Server) getRequirementsTree(projectKey, componentKey string) ([]RequirementTree, error) {
	// This is a simplified version - in practice you'd need recursive queries or multiple queries
	// to build the complete hierarchical tree with implementations and test cases

	whereClause := "WHERE p.project_key = ? AND r.parent_requirement_id IS NULL"
	args := []interface{}{projectKey}

	if componentKey != "" {
		whereClause += " AND c.component_key = ?"
		args = append(args, componentKey)
	}

	query := fmt.Sprintf(`
		SELECT r.id, r.requirement_key, r.requirement_type, r.title,
			   COALESCE(r.description, '') as description, r.category, r.status,
			   COALESCE(r.priority, 'medium') as priority
		FROM requirements r
		JOIN projects p ON r.project_id = p.id
		JOIN system_components c ON r.component_id = c.id
		%s
		ORDER BY r.requirement_key`, whereClause)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requirements []RequirementTree
	for rows.Next() {
		var req RequirementTree
		err := rows.Scan(&req.ID, &req.RequirementKey, &req.RequirementType,
			&req.Title, &req.Description, &req.Category, &req.Status, &req.Priority)
		if err != nil {
			return nil, err
		}

		// Get children recursively (simplified for now)
		children, err := s.getChildRequirements(req.ID)
		if err == nil {
			req.Children = children
			// Calculate counts from children
			for _, child := range children {
				switch strings.ToUpper(child.RequirementType) {
				case "USER_STORY":
					req.UserStoryCount++
				case "TECH_SPEC":
					req.TechSpecCount++
				}
				req.TestCaseCount += len(child.TestCases) + child.TestCaseCount
			}
		}

		// Get implementation and test info
		req.Implementation, _ = s.getImplementationInfo(req.ID)
		req.TestCases, _ = s.getTestCaseInfo(req.ID)
		req.TestCaseCount += len(req.TestCases)

		requirements = append(requirements, req)
	}

	return requirements, nil
}

func (s *Server) getChildRequirements(parentID string) ([]RequirementTree, error) {
	query := `
		SELECT id, requirement_key, requirement_type, title,
			   COALESCE(description, '') as description, category, status,
			   COALESCE(priority, 'medium') as priority
		FROM requirements
		WHERE parent_requirement_id = ?
		ORDER BY requirement_key`

	rows, err := s.db.Query(query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []RequirementTree
	for rows.Next() {
		var child RequirementTree
		err := rows.Scan(&child.ID, &child.RequirementKey, &child.RequirementType,
			&child.Title, &child.Description, &child.Category, &child.Status, &child.Priority)
		if err != nil {
			continue
		}

		// Recursively get children
		grandchildren, err := s.getChildRequirements(child.ID)
		if err == nil {
			child.Children = grandchildren
			// Calculate counts from grandchildren
			for _, grandchild := range grandchildren {
				switch strings.ToUpper(grandchild.RequirementType) {
				case "USER_STORY":
					child.UserStoryCount++
				case "TECH_SPEC":
					child.TechSpecCount++
				}
				child.TestCaseCount += len(grandchild.TestCases) + grandchild.TestCaseCount
			}
		}

		// Get implementation and test info
		child.Implementation, _ = s.getImplementationInfo(child.ID)
		child.TestCases, _ = s.getTestCaseInfo(child.ID)
		child.TestCaseCount += len(child.TestCases)

		children = append(children, child)
	}

	return children, nil
}

func (s *Server) getImplementationInfo(requirementID string) ([]ImplementationInfo, error) {
	query := `SELECT layer, file_path, COALESCE(functions, '[]') as functions
			  FROM implementations WHERE requirement_id = ?`

	rows, err := s.db.Query(query, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var implementations []ImplementationInfo
	for rows.Next() {
		var impl ImplementationInfo
		var functionsJSON string

		err := rows.Scan(&impl.Layer, &impl.FilePath, &functionsJSON)
		if err != nil {
			continue
		}

		// Parse functions JSON
		if functionsJSON != "" && functionsJSON != "[]" {
			json.Unmarshal([]byte(functionsJSON), &impl.Functions)
		}

		implementations = append(implementations, impl)
	}

	return implementations, nil
}

func (s *Server) getTestCaseInfo(requirementID string) ([]TestCaseInfo, error) {
	query := `
		SELECT tf.file_path, tc.test_name, tc.test_type
		FROM requirement_test_coverage rtc
		JOIN test_cases tc ON rtc.test_case_id = tc.id
		JOIN test_files tf ON tc.test_file_id = tf.id
		WHERE rtc.requirement_id = ?`

	rows, err := s.db.Query(query, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []TestCaseInfo
	for rows.Next() {
		var tc TestCaseInfo
		err := rows.Scan(&tc.FilePath, &tc.TestName, &tc.TestType)
		if err != nil {
			continue
		}
		testCases = append(testCases, tc)
	}

	return testCases, nil
}

func countRequirementsByType(req RequirementTree, scopeCount, userStoryCount, techSpecCount *int) {
	switch strings.ToUpper(req.RequirementType) {
	case "SCOPE":
		*scopeCount++
	case "USER_STORY":
		*userStoryCount++
	case "TECH_SPEC":
		*techSpecCount++
	}

	for _, child := range req.Children {
		countRequirementsByType(child, scopeCount, userStoryCount, techSpecCount)
	}
}

func countTestCases(req RequirementTree) int {
	count := len(req.TestCases)
	for _, child := range req.Children {
		count += countTestCases(child)
	}
	return count
}