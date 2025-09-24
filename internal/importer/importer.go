package importer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/peshwar9/tracevibe/internal/database"
	"github.com/peshwar9/tracevibe/internal/models"
	"gopkg.in/yaml.v3"
)

type Importer struct {
	db *database.DB
}

func New(db *database.DB) *Importer {
	return &Importer{db: db}
}

func (imp *Importer) ImportRTMFile(filePath, projectKey string, overwrite bool) error {
	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open RTM file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read RTM file: %w", err)
	}

	// Parse based on file extension
	var rtmData models.RTMData
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &rtmData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &rtmData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s (use .json, .yaml, or .yml)", ext)
	}

	// Use project from metadata if available, otherwise from top level
	if rtmData.Metadata.Project.ID != "" {
		rtmData.Project = rtmData.Metadata.Project
	}

	// Override project key if provided via CLI
	if projectKey != "" {
		rtmData.Project.ID = projectKey
	}

	return imp.importRTMData(&rtmData, overwrite)
}

func (imp *Importer) importRTMData(rtmData *models.RTMData, overwrite bool) error {
	// Start transaction
	tx, err := imp.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Import project
	if err := imp.importProject(tx, &rtmData.Project, overwrite); err != nil {
		return fmt.Errorf("failed to import project: %w", err)
	}

	// Get project ID
	var projectID string
	err = tx.QueryRow("SELECT id FROM projects WHERE project_key = ?", rtmData.Project.ID).Scan(&projectID)
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}

	// If overwrite mode, clean up existing project data
	if overwrite {
		if err := imp.cleanupProjectData(tx, projectID); err != nil {
			return fmt.Errorf("failed to cleanup existing project data: %w", err)
		}
	}

	// Import system components
	componentMap := make(map[string]string) // component_key -> component_id
	for _, component := range rtmData.SystemComponents {
		componentID, err := imp.importComponent(tx, projectID, &component)
		if err != nil {
			return fmt.Errorf("failed to import component %s: %w", component.ID, err)
		}
		componentMap[component.ID] = componentID
	}

	// Import requirements hierarchically
	for _, req := range rtmData.Requirements {
		if err := imp.importRequirement(tx, projectID, componentMap[req.ComponentID], &req, "", overwrite); err != nil {
			return fmt.Errorf("failed to import requirement %s: %w", req.ID, err)
		}
	}

	// Import API endpoints
	for _, endpoint := range rtmData.APIEndpoints {
		if err := imp.importAPIEndpoint(tx, projectID, &endpoint); err != nil {
			return fmt.Errorf("failed to import API endpoint %s %s: %w", endpoint.Method, endpoint.Path, err)
		}
	}

	return tx.Commit()
}

// cleanupProjectData removes all data for a project in the correct order (respecting foreign keys)
func (imp *Importer) cleanupProjectData(tx database.Tx, projectID string) error {
	// Delete in reverse dependency order to respect foreign key constraints

	// Delete requirement test coverage
	_, err := tx.Exec("DELETE FROM requirement_test_coverage WHERE requirement_id IN (SELECT id FROM requirements WHERE project_id = ?)", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete requirement test coverage: %w", err)
	}

	// Delete test cases (via test files)
	_, err = tx.Exec("DELETE FROM test_cases WHERE test_file_id IN (SELECT id FROM test_files WHERE project_id = ?)", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete test cases: %w", err)
	}

	// Delete test files
	_, err = tx.Exec("DELETE FROM test_files WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete test files: %w", err)
	}

	// Delete implementations
	_, err = tx.Exec("DELETE FROM implementations WHERE requirement_id IN (SELECT id FROM requirements WHERE project_id = ?)", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete implementations: %w", err)
	}

	// Delete requirements (will cascade to child requirements)
	_, err = tx.Exec("DELETE FROM requirements WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete requirements: %w", err)
	}

	// Delete API endpoints
	_, err = tx.Exec("DELETE FROM api_endpoints WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete API endpoints: %w", err)
	}

	// Delete system components
	_, err = tx.Exec("DELETE FROM system_components WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete system components: %w", err)
	}

	// Delete project tech stacks
	_, err = tx.Exec("DELETE FROM project_tech_stacks WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project tech stacks: %w", err)
	}

	return nil
}

// cleanupRequirementData removes implementation and test data for a specific requirement
func (imp *Importer) cleanupRequirementData(tx database.Tx, requirementID string) error {
	// Delete requirement test coverage
	_, err := tx.Exec("DELETE FROM requirement_test_coverage WHERE requirement_id = ?", requirementID)
	if err != nil {
		return fmt.Errorf("failed to delete requirement test coverage: %w", err)
	}

	// Delete implementations for this requirement
	_, err = tx.Exec("DELETE FROM implementations WHERE requirement_id = ?", requirementID)
	if err != nil {
		return fmt.Errorf("failed to delete implementations: %w", err)
	}

	return nil
}

func (imp *Importer) importProject(tx database.Tx, project *models.Project, overwrite bool) error {
	// Check if project exists
	var count int
	err := tx.QueryRow("SELECT COUNT(*) FROM projects WHERE project_key = ?", project.ID).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Update existing project
		query := `UPDATE projects SET name = ?, description = ?, repository_url = ?, version = ?, updated_at = datetime('now')
				  WHERE project_key = ?`
		_, err = tx.Exec(query, project.Name, project.Description, project.Repository, project.Version, project.ID)
	} else {
		// Insert new project
		query := `INSERT INTO projects (project_key, name, description, repository_url, version, status)
				  VALUES (?, ?, ?, ?, ?, 'active')`
		_, err = tx.Exec(query, project.ID, project.Name, project.Description, project.Repository, project.Version)
	}

	return err
}

func (imp *Importer) importComponent(tx database.Tx, projectID string, component *models.SystemComponent) (string, error) {
	// Check if component exists
	var componentID string
	err := tx.QueryRow("SELECT id FROM system_components WHERE project_id = ? AND component_key = ?",
		projectID, component.ID).Scan(&componentID)

	if err != nil {
		// Insert new component and get its generated ID
		query := `INSERT INTO system_components (project_id, component_key, name, component_type, technology, description)
				  VALUES (?, ?, ?, ?, ?, ?)
				  RETURNING id`
		err := tx.QueryRow(query, projectID, component.ID, component.Name, component.ComponentType,
			component.Technology, component.Description).Scan(&componentID)
		if err != nil {
			return "", err
		}
		return componentID, nil
	}

	return componentID, nil
}

func (imp *Importer) importRequirement(tx database.Tx, projectID, componentID string, req *models.Requirement, parentID string, overwrite bool) error {
	// Marshal acceptance criteria to JSON
	criteriaJSON, err := req.MarshalAcceptanceCriteriaJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal acceptance criteria: %w", err)
	}

	var parentIDPtr *string
	if parentID != "" {
		parentIDPtr = &parentID
	}

	var reqIDStr string

	if overwrite {
		// In overwrite mode, always insert (old data was already cleaned up)
		query := `INSERT INTO requirements (project_id, component_id, parent_requirement_id, requirement_key,
			  requirement_type, title, description, category, priority, status, acceptance_criteria)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			  RETURNING id`

		err := tx.QueryRow(query, projectID, componentID, parentIDPtr, req.ID, req.RequirementType,
			req.Title, req.Description, req.Category, req.Priority, req.Status, criteriaJSON).Scan(&reqIDStr)
		if err != nil {
			return err
		}
	} else {
		// In update mode, check if requirement exists and update or insert
		var existingID string
		err := tx.QueryRow("SELECT id FROM requirements WHERE project_id = ? AND requirement_key = ?",
			projectID, req.ID).Scan(&existingID)

		if err != nil {
			// Requirement doesn't exist, insert new
			query := `INSERT INTO requirements (project_id, component_id, parent_requirement_id, requirement_key,
				  requirement_type, title, description, category, priority, status, acceptance_criteria)
				  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
				  RETURNING id`

			err := tx.QueryRow(query, projectID, componentID, parentIDPtr, req.ID, req.RequirementType,
				req.Title, req.Description, req.Category, req.Priority, req.Status, criteriaJSON).Scan(&reqIDStr)
			if err != nil {
				return err
			}
		} else {
			// Requirement exists, update it
			query := `UPDATE requirements SET component_id = ?, parent_requirement_id = ?, requirement_type = ?,
				  title = ?, description = ?, category = ?, priority = ?, status = ?, acceptance_criteria = ?,
				  updated_at = datetime('now')
				  WHERE id = ?`

			_, err = tx.Exec(query, componentID, parentIDPtr, req.RequirementType,
				req.Title, req.Description, req.Category, req.Priority, req.Status, criteriaJSON, existingID)
			if err != nil {
				return err
			}
			reqIDStr = existingID

			// Clean up existing implementation and test data for this requirement
			if err := imp.cleanupRequirementData(tx, reqIDStr); err != nil {
				return fmt.Errorf("failed to cleanup existing requirement data: %w", err)
			}
		}
	}

	// Import implementation if present
	if req.Implementation != nil {
		if err := imp.importImplementation(tx, reqIDStr, req.Implementation); err != nil {
			return fmt.Errorf("failed to import implementation: %w", err)
		}
	}

	// Import test coverage if present
	if req.Tests != nil {
		if err := imp.importTestCoverage(tx, projectID, reqIDStr, req.Tests); err != nil {
			return fmt.Errorf("failed to import test coverage: %w", err)
		}
	}

	// Recursively import children
	for _, child := range req.Children {
		if err := imp.importRequirement(tx, projectID, componentID, &child, reqIDStr, overwrite); err != nil {
			return fmt.Errorf("failed to import child requirement %s: %w", child.ID, err)
		}
	}

	return nil
}

func (imp *Importer) importImplementation(tx database.Tx, requirementID string, impl *models.Implementation) error {
	// Import backend implementation
	if impl.Backend != nil {
		for _, file := range impl.Backend.Files {
			functionsJSON, err := models.MarshalStringSliceJSON(file.Functions)
			if err != nil {
				return err
			}

			query := `INSERT INTO implementations (requirement_id, layer, file_path, functions)
					  VALUES (?, 'backend', ?, ?)`
			_, err = tx.Exec(query, requirementID, file.Path, functionsJSON)
			if err != nil {
				return err
			}
		}
	}

	// Import frontend implementation
	if impl.Frontend != nil {
		for _, file := range impl.Frontend.Files {
			functionsJSON, err := models.MarshalStringSliceJSON(file.Functions)
			if err != nil {
				return err
			}

			query := `INSERT INTO implementations (requirement_id, layer, file_path, functions)
					  VALUES (?, 'frontend', ?, ?)`
			_, err = tx.Exec(query, requirementID, file.Path, functionsJSON)
			if err != nil {
				return err
			}
		}
	}

	// Import database implementation
	if impl.Database != nil {
		for _, file := range impl.Database.Files {
			functionsJSON, err := models.MarshalStringSliceJSON(file.Functions)
			if err != nil {
				return err
			}

			query := `INSERT INTO implementations (requirement_id, layer, file_path, functions)
					  VALUES (?, 'database', ?, ?)`
			_, err = tx.Exec(query, requirementID, file.Path, functionsJSON)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (imp *Importer) importTestCoverage(tx database.Tx, projectID, requirementID string, tests *models.TestCoverage) error {
	// Import backend tests
	if err := imp.importTestFiles(tx, projectID, requirementID, "backend", tests.Backend); err != nil {
		return err
	}

	// Import frontend tests
	if err := imp.importTestFiles(tx, projectID, requirementID, "frontend", tests.Frontend); err != nil {
		return err
	}

	return nil
}

func (imp *Importer) importTestFiles(tx database.Tx, projectID, requirementID, layer string, testFiles []models.TestFile) error {
	for _, testFile := range testFiles {
		// Ensure test file exists in test_files table
		testFileID, err := imp.ensureTestFile(tx, projectID, testFile.File, layer)
		if err != nil {
			return err
		}

		// Import individual test functions
		for _, testFunc := range testFile.Functions {
			testCaseID, err := imp.ensureTestCase(tx, testFileID, testFunc)
			if err != nil {
				return err
			}

			// Link test case to requirement
			query := `INSERT OR IGNORE INTO requirement_test_coverage (requirement_id, test_case_id, coverage_type)
					  VALUES (?, ?, 'requirement')`
			_, err = tx.Exec(query, requirementID, testCaseID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (imp *Importer) ensureTestFile(tx database.Tx, projectID, filePath, layer string) (string, error) {
	var testFileID string
	err := tx.QueryRow("SELECT id FROM test_files WHERE project_id = ? AND file_path = ?",
		projectID, filePath).Scan(&testFileID)

	if err != nil {
		// Insert new test file
		query := `INSERT INTO test_files (project_id, file_path, test_type, layer, framework)
				  VALUES (?, ?, 'unit', ?, ?)
				  RETURNING id`

		framework := "Go testing"
		if layer == "frontend" {
			framework = "Jest"
		}

		err := tx.QueryRow(query, projectID, filePath, layer, framework).Scan(&testFileID)
		if err != nil {
			return "", err
		}
		return testFileID, nil
	}

	return testFileID, nil
}

func (imp *Importer) ensureTestCase(tx database.Tx, testFileID, testName string) (string, error) {
	var testCaseID string
	err := tx.QueryRow("SELECT id FROM test_cases WHERE test_file_id = ? AND test_name = ?",
		testFileID, testName).Scan(&testCaseID)

	if err != nil {
		// Insert new test case
		query := `INSERT INTO test_cases (test_file_id, test_name, test_type)
				  VALUES (?, ?, 'unit')
				  RETURNING id`

		err := tx.QueryRow(query, testFileID, testName).Scan(&testCaseID)
		if err != nil {
			return "", err
		}
		return testCaseID, nil
	}

	return testCaseID, nil
}

func (imp *Importer) importAPIEndpoint(tx database.Tx, projectID string, endpoint *models.APIEndpoint) error {
	query := `INSERT OR IGNORE INTO api_endpoints (project_id, method, path, handler_file, handler_function, description)
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := tx.Exec(query, projectID, endpoint.Method, endpoint.Path,
		endpoint.Handler, endpoint.Handler, endpoint.Description)

	return err
}

