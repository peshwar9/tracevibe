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
	"strconv"
	"strings"
	"time"

	"github.com/peshwar9/tracevibe/internal/database"
	"github.com/peshwar9/tracevibe/internal/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	// Create template with custom functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	})
	tmpl, err = tmpl.ParseFS(templatesFS, "web/templates/*.html")
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
	http.HandleFunc("/export/", server.exportHandler)
	http.HandleFunc("/export-json/", server.exportJSONHandler)
	http.HandleFunc("/export-yaml/", server.exportYAMLHandler)
	http.HandleFunc("/export-markdown/", server.exportMarkdownHandler)
	http.HandleFunc("/api/test/run", server.testRunHandler)
	http.HandleFunc("/api/project/", server.projectAPIHandler)
	http.HandleFunc("/api/projects/create", server.createProjectHandler)
	http.HandleFunc("/api/components", server.componentsAPIHandler)
	http.HandleFunc("/api/requirements/", server.requirementsAPIHandler)
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

type ComponentWithRequirements struct {
	ComponentSummary
	Requirements []RequirementTree `json:"requirements"`
}

func (s *Server) projectOverviewHandler(w http.ResponseWriter, r *http.Request, projectKey string) {
	data := struct {
		Title                string
		Project              *database.Project
		Components           []ComponentSummary
		ComponentsWithReqs   []ComponentWithRequirements
		Requirements         []RequirementTree
		ScopeCount           int
		UserStoryCount       int
		TechSpecCount        int
		TotalTestCount       int
		Error                string
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

		// Get requirements for each component
		var componentsWithReqs []ComponentWithRequirements
		for _, comp := range components {
			requirements, err := s.getRequirementsTree(projectKey, comp.ComponentKey)
			if err != nil {
				// Log error but continue with other components
				continue
			}

			compWithReqs := ComponentWithRequirements{
				ComponentSummary: comp,
				Requirements:     requirements,
			}
			componentsWithReqs = append(componentsWithReqs, compWithReqs)

			// Count by type for totals
			for _, req := range requirements {
				countRequirementsByType(req, &data.ScopeCount, &data.UserStoryCount, &data.TechSpecCount)
			}
		}
		data.ComponentsWithReqs = componentsWithReqs
	}

	// Get requirements tree for backward compatibility
	requirements, err := s.getRequirementsTree(projectKey, "")
	if err != nil {
		data.Error = fmt.Sprintf("Error loading requirements: %v", err)
	} else {
		data.Requirements = requirements
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

// Export handler for generating HTML reports
func (s *Server) exportHandler(w http.ResponseWriter, r *http.Request) {
	// Extract project key from URL path
	path := r.URL.Path[len("/export/"):]
	if path == "" {
		http.Error(w, "Project key required", http.StatusBadRequest)
		return
	}

	// Remove trailing slash and any extra path components
	projectKey := strings.Split(path, "/")[0]
	if projectKey == "" {
		http.Error(w, "Project key required", http.StatusBadRequest)
		return
	}

	// Get project data
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil || project == nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Get components
	components, err := s.getComponentsSummary(projectKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading components: %v", err), http.StatusInternalServerError)
		return
	}

	// Get requirements tree
	requirements, err := s.getRequirementsTree(projectKey, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error loading requirements: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate statistics
	stats := struct {
		TotalComponents   int
		TotalRequirements int
		TotalScopes      int
		TotalUserStories int
		TotalTechSpecs   int
		TotalTestCases   int
	}{}

	stats.TotalComponents = len(components)
	for _, req := range requirements {
		countRequirementsByType(req, &stats.TotalScopes, &stats.TotalUserStories, &stats.TotalTechSpecs)
		stats.TotalTestCases += countTestCases(req)
	}
	stats.TotalRequirements = stats.TotalScopes + stats.TotalUserStories + stats.TotalTechSpecs

	// Prepare template data
	data := struct {
		Project     *database.Project
		Components  []ComponentSummary
		Requirements []RequirementTree
		Stats       interface{}
		ExportDate  string
	}{
		Project:     project,
		Components:  components,
		Requirements: requirements,
		Stats:       stats,
		ExportDate:  time.Now().Format("2006-01-02 15:04:05"),
	}

	// Set content type for HTML download
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-rtm-export.html"`, projectKey))

	// Render export template
	err = s.templates.ExecuteTemplate(w, "export.html", data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating export: %v", err), http.StatusInternalServerError)
		return
	}
}

// JSON export handler for LLM consumption
func (s *Server) exportJSONHandler(w http.ResponseWriter, r *http.Request) {
	projectKey, rtmData, err := s.getExportDataAsRTM(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set content type for JSON download
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-rtm-export.json"`, projectKey))

	// Convert to JSON
	jsonData, err := json.MarshalIndent(rtmData, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(jsonData)
}

// YAML export handler for LLM consumption
func (s *Server) exportYAMLHandler(w http.ResponseWriter, r *http.Request) {
	projectKey, rtmData, err := s.getExportDataAsRTM(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set content type for YAML download
	w.Header().Set("Content-Type", "application/x-yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-rtm-export.yaml"`, projectKey))

	// Convert to YAML
	yamlData, err := yaml.Marshal(rtmData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating YAML: %v", err), http.StatusInternalServerError)
		return
	}

	w.Write(yamlData)
}

// Markdown export handler for human/dev consumption
func (s *Server) exportMarkdownHandler(w http.ResponseWriter, r *http.Request) {
	projectKey, exportData, err := s.getExportData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set content type for markdown download
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-rtm-export.md"`, projectKey))

	// Generate markdown content
	markdown := s.generateMarkdown(exportData)
	w.Write([]byte(markdown))
}

// Helper function to get export data
func (s *Server) getExportData(r *http.Request) (string, map[string]interface{}, error) {
	// Extract project key from URL path
	var path string
	if strings.HasPrefix(r.URL.Path, "/export-json/") {
		path = r.URL.Path[len("/export-json/"):]
	} else if strings.HasPrefix(r.URL.Path, "/export-yaml/") {
		path = r.URL.Path[len("/export-yaml/"):]
	} else if strings.HasPrefix(r.URL.Path, "/export-markdown/") {
		path = r.URL.Path[len("/export-markdown/"):]
	}

	if path == "" {
		return "", nil, fmt.Errorf("project key required")
	}

	// Remove trailing slash and any extra path components
	projectKey := strings.Split(path, "/")[0]
	if projectKey == "" {
		return "", nil, fmt.Errorf("project key required")
	}

	// Get project data
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil || project == nil {
		return "", nil, fmt.Errorf("project not found")
	}

	// Get components
	components, err := s.getComponentsSummary(projectKey)
	if err != nil {
		return "", nil, fmt.Errorf("error loading components: %v", err)
	}

	// Get requirements tree
	requirements, err := s.getRequirementsTree(projectKey, "")
	if err != nil {
		return "", nil, fmt.Errorf("error loading requirements: %v", err)
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"total_components":    len(components),
		"total_requirements":  0,
		"total_scopes":       0,
		"total_user_stories": 0,
		"total_tech_specs":   0,
		"total_test_cases":   0,
	}

	scopeCount, userStoryCount, techSpecCount := 0, 0, 0
	testCaseCount := 0
	for _, req := range requirements {
		countRequirementsByType(req, &scopeCount, &userStoryCount, &techSpecCount)
		testCaseCount += countTestCases(req)
	}

	stats["total_scopes"] = scopeCount
	stats["total_user_stories"] = userStoryCount
	stats["total_tech_specs"] = techSpecCount
	stats["total_requirements"] = scopeCount + userStoryCount + techSpecCount
	stats["total_test_cases"] = testCaseCount

	// Prepare export data
	exportData := map[string]interface{}{
		"project":      project,
		"components":   components,
		"requirements": requirements,
		"statistics":   stats,
		"export_date":  time.Now().Format("2006-01-02 15:04:05"),
		"export_timestamp": time.Now().Unix(),
	}

	return projectKey, exportData, nil
}

// Helper function to get export data in RTMData format (compatible with import)
func (s *Server) getExportDataAsRTM(r *http.Request) (string, *models.RTMData, error) {
	// Extract project key from URL path
	var path string
	if strings.HasPrefix(r.URL.Path, "/export-json/") {
		path = r.URL.Path[len("/export-json/"):]
	} else if strings.HasPrefix(r.URL.Path, "/export-yaml/") {
		path = r.URL.Path[len("/export-yaml/"):]
	} else if strings.HasPrefix(r.URL.Path, "/export-markdown/") {
		path = r.URL.Path[len("/export-markdown/"):]
	}

	if path == "" {
		return "", nil, fmt.Errorf("project key required")
	}

	// Remove trailing slash and any extra path components
	projectKey := strings.Split(path, "/")[0]
	if projectKey == "" {
		return "", nil, fmt.Errorf("project key required")
	}

	// Get project data
	project, err := s.db.GetProjectByKey(projectKey)
	if err != nil || project == nil {
		return "", nil, fmt.Errorf("project not found")
	}

	// Get all requirements for the project
	requirements, err := s.db.GetRequirementsByProject(project.ID)
	if err != nil {
		return "", nil, fmt.Errorf("error loading requirements: %v", err)
	}

	// Get all components using existing method
	componentSummaries, err := s.getComponentsSummary(projectKey)
	if err != nil {
		return "", nil, fmt.Errorf("error loading components: %v", err)
	}

	// Create RTMData structure
	rtmData := &models.RTMData{
		Metadata: models.RTMMetadata{
			GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
			GeneratedBy: "TraceVibe Export",
			Project: models.Project{
				ID:          project.ProjectKey,
				Name:        project.Name,
				Description: s.derefString(project.Description),
				Repository:  s.derefString(project.RepositoryURL),
				Version:     s.derefString(project.Version),
				LastUpdated: project.UpdatedAt,
			},
		},
		Project: models.Project{
			ID:          project.ProjectKey,
			Name:        project.Name,
			Description: s.derefString(project.Description),
			Repository:  s.derefString(project.RepositoryURL),
			Version:     s.derefString(project.Version),
			LastUpdated: project.UpdatedAt,
		},
		SystemComponents: []models.SystemComponent{},
		Scopes:          []models.Scope{},
	}

	// Convert components
	for _, comp := range componentSummaries {
		rtmData.SystemComponents = append(rtmData.SystemComponents, models.SystemComponent{
			ID:            comp.ComponentKey,
			Name:          comp.Name,
			ComponentType: comp.ComponentType,
			Technology:    comp.Technology,
			Description:   comp.Description,
		})
	}

	// Build hierarchical requirements structure (Scopes -> UserStories -> TechSpecs)
	scopeMap := make(map[string]*models.Scope)
	userStoryMap := make(map[string]*models.UserStory)

	// First pass: create all scopes
	for _, req := range requirements {
		if req.RequirementType == "scope" || req.RequirementType == "SCOPE" {
			scope := &models.Scope{
				ID:          req.RequirementKey,
				ComponentID: req.ComponentID,
				Name:        req.Title,
				Description: s.derefString(req.Description),
				Priority:    req.Priority,
				Status:      req.Status,
				UserStories: []models.UserStory{},
			}
			scopeMap[req.ID] = scope
			rtmData.Scopes = append(rtmData.Scopes, *scope)
		}
	}

	// Second pass: create user stories and attach to scopes
	for _, req := range requirements {
		if req.RequirementType == "user_story" || req.RequirementType == "USER_STORY" {
			if req.ParentRequirementID != nil {
				if parentScope, exists := scopeMap[*req.ParentRequirementID]; exists {
					userStory := &models.UserStory{
						ID:          req.RequirementKey,
						Name:        req.Title,
						Description: s.derefString(req.Description),
						Priority:    req.Priority,
						Status:      req.Status,
						TechSpecs:   []models.TechSpec{},
					}
					userStoryMap[req.ID] = userStory
					parentScope.UserStories = append(parentScope.UserStories, *userStory)
				}
			}
		}
	}

	// Third pass: create tech specs and attach to user stories
	for _, req := range requirements {
		if req.RequirementType == "tech_spec" || req.RequirementType == "TECH_SPEC" {
			if req.ParentRequirementID != nil {
				if parentStory, exists := userStoryMap[*req.ParentRequirementID]; exists {
					// Get implementation details
					impl, _ := s.getImplementationForRequirement(req.ID)
					// Get test coverage
					testCov, _ := s.getTestCoverageForRequirement(project.ID, req.ID)

					techSpec := models.TechSpec{
						ID:                 req.RequirementKey,
						Name:               req.Title,
						Description:        s.derefString(req.Description),
						Priority:           req.Priority,
						Status:             req.Status,
						AcceptanceCriteria: req.AcceptanceCriteria,
						Implementation:     impl,
						TestCoverage:       testCov,
					}
					parentStory.TechSpecs = append(parentStory.TechSpecs, techSpec)
				}
			}
		}
	}

	// Update the scopes in rtmData with the populated user stories and tech specs
	for i, scope := range rtmData.Scopes {
		for _, updatedScope := range scopeMap {
			if updatedScope.ID == scope.ID {
				rtmData.Scopes[i] = *updatedScope
				break
			}
		}
	}

	return projectKey, rtmData, nil
}

// Helper function to dereference string pointers
func (s *Server) derefString(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

// Helper function to get implementation for a requirement
func (s *Server) getImplementationForRequirement(requirementID string) (*models.Implementation, error) {
	query := `
		SELECT layer, file_path, functions
		FROM implementations
		WHERE requirement_id = ?`

	rows, err := s.db.Query(query, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	impl := &models.Implementation{}
	hasData := false

	for rows.Next() {
		var layer, filePath, functionsJSON string
		if err := rows.Scan(&layer, &filePath, &functionsJSON); err != nil {
			continue
		}

		hasData = true
		var functions []string
		if functionsJSON != "" && functionsJSON != "[]" {
			if err := json.Unmarshal([]byte(functionsJSON), &functions); err == nil {
				fileImpl := models.FileImpl{
					Path:      filePath,
					Functions: functions,
				}

				switch layer {
				case "backend":
					if impl.Backend == nil {
						impl.Backend = &models.BackendImpl{Files: []models.FileImpl{}}
					}
					impl.Backend.Files = append(impl.Backend.Files, fileImpl)
				case "frontend":
					if impl.Frontend == nil {
						impl.Frontend = &models.FrontendImpl{Files: []models.FileImpl{}}
					}
					impl.Frontend.Files = append(impl.Frontend.Files, fileImpl)
				case "database":
					if impl.Database == nil {
						impl.Database = &models.DatabaseImpl{Files: []models.FileImpl{}}
					}
					impl.Database.Files = append(impl.Database.Files, fileImpl)
				}
			}
		}
	}

	if !hasData {
		return nil, nil
	}
	return impl, nil
}

// Helper function to get test coverage for a requirement
func (s *Server) getTestCoverageForRequirement(projectID, requirementID string) (*models.TestCoverage, error) {
	// Query to get test cases for this requirement
	query := `
		SELECT tf.file_path, tc.test_name, tf.test_type
		FROM requirement_test_coverage rtc
		JOIN test_cases tc ON rtc.test_case_id = tc.id
		JOIN test_files tf ON tc.test_file_id = tf.id
		WHERE rtc.requirement_id = ?`

	rows, err := s.db.Query(query, requirementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	testCov := &models.TestCoverage{
		UnitTests:        []models.TestFile{},
		IntegrationTests: []models.TestFile{},
		E2ETests:         []models.TestFile{},
	}

	// Group test cases by file and type
	testFileMap := make(map[string]map[string][]string) // file -> type -> functions
	hasData := false

	for rows.Next() {
		var filePath, testName, testType string
		if err := rows.Scan(&filePath, &testName, &testType); err != nil {
			continue
		}

		hasData = true
		if _, exists := testFileMap[filePath]; !exists {
			testFileMap[filePath] = make(map[string][]string)
		}

		if testType == "" {
			testType = "unit"
		}

		testFileMap[filePath][testType] = append(testFileMap[filePath][testType], testName)
	}

	// Convert to TestFile structures
	for filePath, typeMap := range testFileMap {
		for testType, functions := range typeMap {
			tf := models.TestFile{
				File:      filePath,
				Functions: functions,
			}

			switch testType {
			case "unit":
				testCov.UnitTests = append(testCov.UnitTests, tf)
			case "integration":
				testCov.IntegrationTests = append(testCov.IntegrationTests, tf)
			case "e2e":
				testCov.E2ETests = append(testCov.E2ETests, tf)
			}
		}
	}

	if !hasData {
		return nil, nil
	}
	return testCov, nil
}

// Helper function to generate markdown content
func (s *Server) generateMarkdown(exportData map[string]interface{}) string {
	project := exportData["project"].(*database.Project)
	components := exportData["components"].([]ComponentSummary)
	requirements := exportData["requirements"].([]RequirementTree)
	stats := exportData["statistics"].(map[string]interface{})
	exportDate := exportData["export_date"].(string)

	var md strings.Builder

	// Header
	md.WriteString(fmt.Sprintf("# %s - Requirements Traceability Matrix\n\n", project.Name))
	md.WriteString(fmt.Sprintf("**Project Key:** %s  \n", project.ProjectKey))
	if project.Description != nil && *project.Description != "" {
		md.WriteString(fmt.Sprintf("**Description:** %s  \n", *project.Description))
	}
	if project.RepositoryURL != nil && *project.RepositoryURL != "" {
		md.WriteString(fmt.Sprintf("**Repository:** %s  \n", *project.RepositoryURL))
	}
	if project.Version != nil && *project.Version != "" {
		md.WriteString(fmt.Sprintf("**Version:** %s  \n", *project.Version))
	}
	md.WriteString(fmt.Sprintf("**Export Date:** %s  \n\n", exportDate))

	// Statistics
	md.WriteString("## 📊 Project Statistics\n\n")
	md.WriteString("| Metric | Count |\n")
	md.WriteString("|--------|-------|\n")
	md.WriteString(fmt.Sprintf("| Components | %v |\n", stats["total_components"]))
	md.WriteString(fmt.Sprintf("| Total Requirements | %v |\n", stats["total_requirements"]))
	md.WriteString(fmt.Sprintf("| Scopes | %v |\n", stats["total_scopes"]))
	md.WriteString(fmt.Sprintf("| User Stories | %v |\n", stats["total_user_stories"]))
	md.WriteString(fmt.Sprintf("| Technical Specifications | %v |\n", stats["total_tech_specs"]))
	md.WriteString(fmt.Sprintf("| Test Cases | %v |\n\n", stats["total_test_cases"]))

	// Components
	md.WriteString("## 🏗️ System Components\n\n")
	for _, comp := range components {
		md.WriteString(fmt.Sprintf("### %s\n", comp.Name))
		md.WriteString(fmt.Sprintf("- **Type:** %s\n", comp.ComponentType))
		if comp.Technology != "" {
			md.WriteString(fmt.Sprintf("- **Technology:** %s\n", comp.Technology))
		}
		if comp.Description != "" {
			md.WriteString(fmt.Sprintf("- **Description:** %s\n", comp.Description))
		}
		md.WriteString(fmt.Sprintf("- **Requirements:** %d Scopes, %d User Stories, %d Tech Specs\n",
			comp.ScopeCount, comp.UserStoryCount, comp.TechSpecCount))
		md.WriteString(fmt.Sprintf("- **Test Cases:** %d\n\n", comp.TestCaseCount))
	}

	// Requirements
	md.WriteString("## 📋 Requirements Hierarchy\n\n")
	for _, req := range requirements {
		s.writeRequirementToMarkdown(&md, req, 3)
	}

	// Footer
	md.WriteString("\n---\n")
	md.WriteString("*Generated by TraceVibe RTM Export*\n")

	return md.String()
}

// Helper function to write requirements recursively to markdown
func (s *Server) writeRequirementToMarkdown(md *strings.Builder, req RequirementTree, level int) {
	// Header with appropriate level
	headerPrefix := strings.Repeat("#", level)

	// Badge for requirement type
	var badge string
	switch strings.ToUpper(req.RequirementType) {
	case "SCOPE":
		badge = "🎯 SCOPE"
	case "USER_STORY":
		badge = "👤 USER STORY"
	case "TECH_SPEC":
		badge = "⚙️ TECH SPEC"
	default:
		badge = req.RequirementType
	}

	md.WriteString(fmt.Sprintf("%s %s: %s\n\n", headerPrefix, badge, req.Title))

	// Metadata
	md.WriteString(fmt.Sprintf("- **ID:** %s\n", req.RequirementKey))
	md.WriteString(fmt.Sprintf("- **Status:** %s\n", req.Status))
	md.WriteString(fmt.Sprintf("- **Priority:** %s\n", req.Priority))
	md.WriteString(fmt.Sprintf("- **Category:** %s\n", req.Category))

	if req.Description != "" {
		md.WriteString(fmt.Sprintf("- **Description:** %s\n", req.Description))
	}

	// Test cases
	if len(req.TestCases) > 0 {
		md.WriteString("- **Test Cases:**\n")
		for _, tc := range req.TestCases {
			md.WriteString(fmt.Sprintf("  - %s: %s\n", tc.TestType, tc.FilePath))
		}
	}

	// Implementation
	if len(req.Implementation) > 0 {
		md.WriteString("- **Implementation:**\n")
		for _, impl := range req.Implementation {
			md.WriteString(fmt.Sprintf("  - **%s:** %s", impl.Layer, impl.FilePath))
			if len(impl.Functions) > 0 {
				md.WriteString(fmt.Sprintf(" (Functions: %s)", strings.Join(impl.Functions, ", ")))
			}
			md.WriteString("\n")
		}
	}

	md.WriteString("\n")

	// Process children recursively
	for _, child := range req.Children {
		s.writeRequirementToMarkdown(md, child, level+1)
	}
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
	// First, check if the project has a Makefile with test targets - use that if available
	if s.projectBasePath != "" {
		makefilePath := filepath.Join(s.projectBasePath, "Makefile")
		if _, err := os.Stat(makefilePath); err == nil {
			// Check if Makefile has full-test target
			if s.hasMakeTarget(makefilePath, "full-test") {
				return s.runMakeTest(projectKey, componentKey, "full-test")
			}
			// Fallback to 'test' target if available
			if s.hasMakeTarget(makefilePath, "test") {
				return s.runMakeTest(projectKey, componentKey, "test")
			}
		}
	}

	// Fallback to individual test file execution
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

	// Inherit environment and ensure GOTOOLCHAIN is set to avoid version mismatches
	cmd.Env = os.Environ()
	// Set GOTOOLCHAIN to use the current Go version and handle auto-downloads
	cmd.Env = append(cmd.Env, "GOTOOLCHAIN=go1.25.1+auto")

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

// hasMakeTarget checks if a Makefile contains a specific target
func (s *Server) hasMakeTarget(makefilePath, target string) bool {
	content, err := os.ReadFile(makefilePath)
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for target: pattern (allowing for dependencies)
		if strings.HasPrefix(line, target+":") {
			return true
		}
	}
	return false
}

// runMakeTest executes a make target for testing
func (s *Server) runMakeTest(projectKey, componentKey, target string) (*TestResult, error) {
	startTime := time.Now()

	// Execute make command in the project directory
	cmd := exec.Command("make", target)
	cmd.Dir = s.projectBasePath

	// Inherit environment and ensure GOTOOLCHAIN is set to avoid version mismatches
	cmd.Env = os.Environ()
	// Set GOTOOLCHAIN to use the current Go version and handle auto-downloads
	cmd.Env = append(cmd.Env, "GOTOOLCHAIN=go1.25.1+auto")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Add command info to output
	cmdInfo := fmt.Sprintf("Command: make %s\nWorking Dir: %s\n\n", target, s.projectBasePath)
	outputStr = cmdInfo + outputStr

	duration := time.Since(startTime)

	// Parse make output to determine pass/fail counts
	passed := 0
	failed := 0

	if err != nil {
		// Make failed - parse output for test results if available
		passed, failed = s.parseMakeTestOutput(outputStr)
		if passed == 0 && failed == 0 {
			failed = 1 // At least one failure if make command failed
		}
	} else {
		// Make succeeded - parse output for test results
		passed, failed = s.parseMakeTestOutput(outputStr)
		if passed == 0 && failed == 0 {
			passed = 1 // At least one success if make succeeded
		}
	}

	return &TestResult{
		Passed:   passed,
		Failed:   failed,
		Duration: duration.Round(time.Millisecond).String(),
		Output:   outputStr,
	}, nil
}

// parseMakeTestOutput extracts test counts from make output
func (s *Server) parseMakeTestOutput(output string) (passed, failed int) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for Go test result patterns
		if strings.Contains(line, "PASS") && (strings.Contains(line, "ok") || strings.Contains(line, "github.com")) {
			passed++
		}
		if strings.Contains(line, "FAIL") && (strings.Contains(line, "github.com") || strings.Contains(line, "FAIL\t")) {
			failed++
		}

		// Look for other test result patterns
		if strings.Contains(line, "✓") || strings.Contains(line, "passed") {
			// Try to extract number if present
			if parts := strings.Fields(line); len(parts) > 0 {
				for _, part := range parts {
					if num, err := strconv.Atoi(part); err == nil && num > 0 && strings.Contains(line, "pass") {
						passed += num
						break
					}
				}
			}
		}

		if strings.Contains(line, "✗") || strings.Contains(line, "failed") {
			// Try to extract number if present
			if parts := strings.Fields(line); len(parts) > 0 {
				for _, part := range parts {
					if num, err := strconv.Atoi(part); err == nil && num > 0 && strings.Contains(line, "fail") {
						failed += num
						break
					}
				}
			}
		}
	}

	return passed, failed
}

// Components API handler
func (s *Server) componentsAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		s.createComponentHandler(w, r)
	default:
		http.Error(w, fmt.Sprintf("Method %s not allowed", r.Method), http.StatusMethodNotAllowed)
	}
}

func (s *Server) createComponentHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ProjectID      string  `json:"project_id"`
		ComponentKey   string  `json:"component_key"`
		Name           string  `json:"name"`
		ComponentType  string  `json:"component_type"`
		Technology     *string `json:"technology"`
		Description    *string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if data.ProjectID == "" || data.ComponentKey == "" || data.Name == "" || data.ComponentType == "" {
		http.Error(w, "Missing required fields: project_id, component_key, name, component_type", http.StatusBadRequest)
		return
	}

	// Create the component in the database
	query := `
		INSERT INTO system_components (project_id, component_key, name, component_type, technology, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query, data.ProjectID, data.ComponentKey, data.Name, data.ComponentType, data.Technology, data.Description)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating component: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the ID of the created component
	componentID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting component ID: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":       true,
		"id":            componentID,
		"component_key": data.ComponentKey,
		"name":          data.Name,
	}

	json.NewEncoder(w).Encode(response)
}

// Requirements API handlers
func (s *Server) requirementsAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Path[len("/api/requirements/"):]
	parts := splitPath(path)

	switch r.Method {
	case http.MethodGet:
		// GET /api/requirements/{id} - Get requirement by ID
		if len(parts) == 1 && parts[0] != "" {
			s.getRequirementHandler(w, r, parts[0])
			return
		}

	case http.MethodPost:
		// POST /api/requirements/create - Create new requirement
		if len(parts) == 1 && parts[0] == "create" {
			s.createRequirementHandler(w, r)
			return
		}
		// POST /api/requirements/generate-key - Generate next requirement key
		if len(parts) == 1 && parts[0] == "generate-key" {
			s.generateRequirementKeyHandler(w, r)
			return
		}

	case http.MethodPut:
		// PUT /api/requirements/{id} - Update requirement
		if len(parts) == 1 && parts[0] != "" {
			s.updateRequirementHandler(w, r, parts[0])
			return
		}
		// PUT /api/requirements/{id}/description - Update description only
		if len(parts) == 2 && parts[1] == "description" {
			s.updateRequirementDescriptionHandler(w, r, parts[0])
			return
		}

	case http.MethodDelete:
		// DELETE /api/requirements/{id} - Delete requirement
		if len(parts) == 1 && parts[0] != "" {
			s.deleteRequirementHandler(w, r, parts[0])
			return
		}
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// Project creation API handler
func (s *Server) createProjectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var projectData struct {
		Name          string `json:"name"`
		ProjectKey    string `json:"project_key"`
		Description   string `json:"description,omitempty"`
		RepositoryURL string `json:"repository_url,omitempty"`
		Version       string `json:"version,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&projectData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if projectData.Name == "" {
		http.Error(w, "Project name is required", http.StatusBadRequest)
		return
	}

	if projectData.ProjectKey == "" {
		http.Error(w, "Project key is required", http.StatusBadRequest)
		return
	}

	// Create project
	project := &database.Project{
		ProjectKey: projectData.ProjectKey,
		Name:       projectData.Name,
		Status:     "active",
	}

	// Set optional fields
	if projectData.Description != "" {
		project.Description = &projectData.Description
	}
	if projectData.RepositoryURL != "" {
		project.RepositoryURL = &projectData.RepositoryURL
	}
	if projectData.Version != "" {
		project.Version = &projectData.Version
	}

	if err := s.db.CreateProject(project); err != nil {
		http.Error(w, fmt.Sprintf("Error creating project: %v", err), http.StatusInternalServerError)
		return
	}

	// TODO: Create a default system component for the project later

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"id":          project.ID,
		"project_key": project.ProjectKey,
	})
}

func (s *Server) getRequirementHandler(w http.ResponseWriter, r *http.Request, requirementID string) {
	requirement, err := s.db.GetRequirementByID(requirementID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting requirement: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(requirement)
}

func (s *Server) createRequirementHandler(w http.ResponseWriter, r *http.Request) {
	var req database.Requirement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Generate requirement key if not provided
	if req.RequirementKey == "" {
		key, err := s.db.GenerateNextRequirementKey(req.ProjectID, req.ComponentID, req.RequirementType, req.ParentRequirementID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error generating requirement key: %v", err), http.StatusInternalServerError)
			return
		}
		req.RequirementKey = key
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Status == "" {
		req.Status = "not_started"
	}

	if err := s.db.CreateRequirement(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error creating requirement: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      req.ID,
		"key":     req.RequirementKey,
	})
}

func (s *Server) updateRequirementHandler(w http.ResponseWriter, r *http.Request, requirementID string) {
	var req database.Requirement
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	req.ID = requirementID
	if err := s.db.UpdateRequirement(&req); err != nil {
		http.Error(w, fmt.Sprintf("Error updating requirement: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      requirementID,
	})
}

func (s *Server) updateRequirementDescriptionHandler(w http.ResponseWriter, r *http.Request, requirementID string) {
	var body struct {
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateRequirementDescription(requirementID, body.Description); err != nil {
		http.Error(w, fmt.Sprintf("Error updating description: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      requirementID,
	})
}

func (s *Server) deleteRequirementHandler(w http.ResponseWriter, r *http.Request, requirementID string) {
	if err := s.db.DeleteRequirement(requirementID); err != nil {
		http.Error(w, fmt.Sprintf("Error deleting requirement: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      requirementID,
	})
}

func (s *Server) generateRequirementKeyHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID           string  `json:"project_id"`
		ComponentID         string  `json:"component_id"`
		RequirementType     string  `json:"requirement_type"`
		ParentRequirementID *string `json:"parent_requirement_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	key, err := s.db.GenerateNextRequirementKey(
		body.ProjectID,
		body.ComponentID,
		body.RequirementType,
		body.ParentRequirementID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating key: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"key":     key,
	})
}