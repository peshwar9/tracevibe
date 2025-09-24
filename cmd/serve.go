package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

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

		if err := startServer(port, dbPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	serveCmd.Flags().StringP("db-path", "d", getDefaultDBPath(), "SQLite database path")
}

func startServer(port int, dbPath string) error {
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

	server := &Server{db: db, templates: tmpl}

	// Routes
	http.HandleFunc("/", server.dashboardHandler)
	http.HandleFunc("/projects/", server.projectHandler)
	http.HandleFunc("/api/", server.apiHandler)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🚀 TraceVibe server starting on http://localhost%s\n", addr)
	fmt.Printf("📊 Database: %s\n", dbPath)
	fmt.Printf("🔍 Open your browser and navigate to the URL above\n")

	return http.ListenAndServe(addr, nil)
}

type Server struct {
	db        *database.DB
	templates *template.Template
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
	ID               string `json:"id"`
	ProjectKey       string `json:"project_key"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	Status           string `json:"status"`
	ComponentCount   int    `json:"component_count"`
	RequirementCount int    `json:"requirement_count"`
	TestCaseCount    int    `json:"test_case_count"`
}

type ComponentSummary struct {
	ID               string `json:"id"`
	ComponentKey     string `json:"component_key"`
	Name             string `json:"name"`
	ComponentType    string `json:"component_type"`
	Technology       string `json:"technology"`
	Description      string `json:"description"`
	TotalRequirements int    `json:"total_requirements"`
	ScopeCount       int    `json:"scope_count"`
	UserStoryCount   int    `json:"user_story_count"`
	TechSpecCount    int    `json:"tech_spec_count"`
	ImplementationCount int `json:"implementation_count"`
	TestCaseCount    int    `json:"test_case_count"`
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

// Database query methods (these would need to be implemented in the database package)

func (s *Server) getProjectsSummary() ([]ProjectSummary, error) {
	query := `
		SELECT
			p.id, p.project_key, p.name, COALESCE(p.description, '') as description, p.status,
			COALESCE(comp_counts.component_count, 0) as component_count,
			COALESCE(req_counts.requirement_count, 0) as requirement_count,
			COALESCE(test_counts.test_count, 0) as test_count
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
			SELECT p.id as project_id, COUNT(DISTINCT tc.id) as test_count
			FROM projects p
			LEFT JOIN test_files tf ON p.id = tf.project_id
			LEFT JOIN test_cases tc ON tf.id = tc.test_file_id
			GROUP BY p.id
		) test_counts ON p.id = test_counts.project_id
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
			&p.ComponentCount, &p.RequirementCount, &p.TestCaseCount)
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
			   cs.total_requirements, cs.scope_count, cs.user_story_count, cs.tech_spec_count,
			   cs.implementation_count, cs.test_case_count
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
		}

		// Get implementation and test info
		req.Implementation, _ = s.getImplementationInfo(req.ID)
		req.TestCases, _ = s.getTestCaseInfo(req.ID)

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
		}

		// Get implementation and test info
		child.Implementation, _ = s.getImplementationInfo(child.ID)
		child.TestCases, _ = s.getTestCaseInfo(child.ID)

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
	switch req.RequirementType {
	case "scope":
		*scopeCount++
	case "user_story":
		*userStoryCount++
	case "tech_spec":
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