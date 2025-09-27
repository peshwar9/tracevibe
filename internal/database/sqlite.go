package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

type DB struct {
	*sql.DB
}

// Interfaces for transaction support
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type Row interface {
	Scan(dest ...interface{}) error
}

type Tx interface {
	Exec(query string, args ...interface{}) (Result, error)
	QueryRow(query string, args ...interface{}) Row
	Commit() error
	Rollback() error
}

// Wrapper for sql.Tx to implement our interface
type txWrapper struct {
	*sql.Tx
}

func (tx *txWrapper) Exec(query string, args ...interface{}) (Result, error) {
	return tx.Tx.Exec(query, args...)
}

func (tx *txWrapper) QueryRow(query string, args ...interface{}) Row {
	return tx.Tx.QueryRow(query, args...)
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) InitSchema() error {
	// Check if tables already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='projects'").Scan(&count)
	if err == nil && count > 0 {
		// Tables already exist, run migrations
		db.runMigrations()
		return nil
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// runMigrations adds any missing columns to existing databases
func (db *DB) runMigrations() {
	// Check if tags column exists in system_components
	var tagCount int
	err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('system_components') WHERE name='tags'").Scan(&tagCount)
	if err == nil && tagCount == 0 {
		// Tags column doesn't exist, add it
		db.Exec("ALTER TABLE system_components ADD COLUMN tags TEXT")
	}

	// Check if project_context column exists in projects
	var projectContextCount int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('projects') WHERE name='project_context'").Scan(&projectContextCount)
	if err == nil && projectContextCount == 0 {
		// Project context column doesn't exist, add it
		db.Exec("ALTER TABLE projects ADD COLUMN project_context TEXT")
	}

	// Check if tool_settings table exists
	var settingsTableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tool_settings'").Scan(&settingsTableCount)
	if err == nil && settingsTableCount == 0 {
		// Tool settings table doesn't exist, create it
		db.Exec(`CREATE TABLE tool_settings (
			id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
			setting_key TEXT UNIQUE NOT NULL,
			setting_value TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`)

		// Insert default methodology
		defaultMethodology := `# TraceVibe Code Generation Methodology

## Overview
TraceVibe follows a component-based architecture where each component is an independently deployable unit with its own source code, tests, build configuration, and deployment setup.

## Code Organization Structure

### Component-Based Architecture
` + "```" + `
project-root/
├── components/
│   ├── component-1/
│   │   ├── src/                 # Source code
│   │   ├── tests/               # Unit & integration tests
│   │   ├── config/              # Component-specific config
│   │   ├── build/               # Build scripts & artifacts
│   │   ├── deploy/              # Deployment configurations
│   │   └── docs/                # Component documentation
│   ├── component-2/
│   │   └── [same structure]
│   └── shared/                  # Shared utilities & libraries
├── docs/                        # Project-level documentation
├── scripts/                     # Build & deployment orchestration
└── infrastructure/              # Infrastructure as code
` + "```" + `

## Implementation Guidelines

### 1. Component Independence
- Each component should be independently:
  - **Testable**: Complete test suite in component/tests/
  - **Buildable**: Build scripts in component/build/
  - **Deployable**: Deployment config in component/deploy/
  - **Configurable**: Environment-specific config in component/config/

### 2. Code Generation Approach
When implementing requirements:

**Component Level**:
- Generate complete component structure
- Include all architectural layers (API, business logic, data access)
- Create comprehensive test suite
- Add build and deployment configurations

**Scope Level**:
- Implement functional scope across relevant components
- Ensure cross-component integration
- Create integration tests
- Document scope-level architecture decisions

**Story Level**:
- Implement user journey end-to-end
- Include frontend, backend, and data layers as needed
- Create user acceptance tests
- Document user story completion criteria

**Tech Spec Level**:
- Focus on specific technical implementation
- Include detailed unit tests
- Document technical decisions and trade-offs
- Ensure integration with related specs

### 3. Testing Strategy
- **Unit Tests**: In component/tests/unit/
- **Integration Tests**: In component/tests/integration/
- **End-to-End Tests**: In component/tests/e2e/
- **Contract Tests**: For API boundaries between components

### 4. Documentation Standards
- **README.md**: Component overview and quick start
- **API.md**: API documentation for components with interfaces
- **ARCHITECTURE.md**: Component architecture decisions
- **TESTING.md**: How to run and maintain tests

### 5. RTM Integration
- Add RTM reference comments in code: ` + "`/* RTM: [SPEC_ID] */`" + `
- Map requirements to implementation files
- Maintain traceability from code back to requirements
- Update RTM when implementation deviates from specs

## Deployment Considerations
- Each component can be deployed independently
- Use container-based deployment when possible
- Include health checks and monitoring
- Implement circuit breakers for component communication
- Plan for graceful degradation when components are unavailable

## Quality Gates
Before considering implementation complete:
1. ✅ All RTM requirements mapped to code
2. ✅ Test coverage meets project standards
3. ✅ Component can be built independently
4. ✅ Component can be deployed independently
5. ✅ Documentation is complete and up-to-date
6. ✅ Security review completed (if applicable)
7. ✅ Performance benchmarks meet requirements

This methodology ensures that code generated from TraceVibe RTM follows consistent, maintainable, and scalable patterns.`

		db.Exec("INSERT INTO tool_settings (setting_key, setting_value) VALUES (?, ?)", "methodology", defaultMethodology)
	}
}

func (db *DB) GetProjectByKey(projectKey string) (*Project, error) {
	var p Project
	query := `SELECT id, project_key, name, description, repository_url, version, status, project_context, created_at, updated_at
			  FROM projects WHERE project_key = ?`

	err := db.QueryRow(query, projectKey).Scan(
		&p.ID, &p.ProjectKey, &p.Name, &p.Description,
		&p.RepositoryURL, &p.Version, &p.Status, &p.ProjectContext, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &p, err
}

func (db *DB) Begin() (Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &txWrapper{tx}, nil
}

func (db *DB) CreateProject(p *Project) error {
	query := `INSERT INTO projects (project_key, name, description, repository_url, version, status, project_context)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, p.ProjectKey, p.Name, p.Description, p.RepositoryURL, p.Version, p.Status, p.ProjectContext)
	return err
}

// GetMethodology retrieves the tool-level methodology setting
func (db *DB) GetMethodology() (string, error) {
	var methodology string
	query := `SELECT setting_value FROM tool_settings WHERE setting_key = 'methodology'`

	err := db.QueryRow(query).Scan(&methodology)
	if err == sql.ErrNoRows {
		// Return default methodology if not found
		return getDefaultMethodology(), nil
	}
	return methodology, err
}

// SaveMethodology saves the tool-level methodology setting
func (db *DB) SaveMethodology(methodology string) error {
	query := `INSERT OR REPLACE INTO tool_settings (setting_key, setting_value, updated_at)
		  VALUES ('methodology', ?, datetime('now'))`

	_, err := db.Exec(query, methodology)
	return err
}

// SaveProjectContext saves the project-specific context
func (db *DB) SaveProjectContext(projectKey, context string) error {
	query := `UPDATE projects SET project_context = ?, updated_at = datetime('now') WHERE project_key = ?`

	_, err := db.Exec(query, context, projectKey)
	return err
}

func getDefaultMethodology() string {
	return `# TraceVibe Code Generation Methodology

## Overview
TraceVibe follows a component-based architecture where each component is an independently deployable unit with its own source code, tests, build configuration, and deployment setup.

## Code Organization Structure

### Component-Based Architecture
` + "```" + `
project-root/
├── components/
│   ├── component-1/
│   │   ├── src/                 # Source code
│   │   ├── tests/               # Unit & integration tests
│   │   ├── config/              # Component-specific config
│   │   ├── build/               # Build scripts & artifacts
│   │   ├── deploy/              # Deployment configurations
│   │   └── docs/                # Component documentation
│   ├── component-2/
│   │   └── [same structure]
│   └── shared/                  # Shared utilities & libraries
├── docs/                        # Project-level documentation
├── scripts/                     # Build & deployment orchestration
└── infrastructure/              # Infrastructure as code
` + "```" + `

## Implementation Guidelines

### 1. Component Independence
- Each component should be independently:
  - **Testable**: Complete test suite in component/tests/
  - **Buildable**: Build scripts in component/build/
  - **Deployable**: Deployment config in component/deploy/
  - **Configurable**: Environment-specific config in component/config/

### 2. Code Generation Approach
When implementing requirements:

**Component Level**:
- Generate complete component structure
- Include all architectural layers (API, business logic, data access)
- Create comprehensive test suite
- Add build and deployment configurations

**Scope Level**:
- Implement functional scope across relevant components
- Ensure cross-component integration
- Create integration tests
- Document scope-level architecture decisions

**Story Level**:
- Implement user journey end-to-end
- Include frontend, backend, and data layers as needed
- Create user acceptance tests
- Document user story completion criteria

**Tech Spec Level**:
- Focus on specific technical implementation
- Include detailed unit tests
- Document technical decisions and trade-offs
- Ensure integration with related specs

### 3. Testing Strategy
- **Unit Tests**: In component/tests/unit/
- **Integration Tests**: In component/tests/integration/
- **End-to-End Tests**: In component/tests/e2e/
- **Contract Tests**: For API boundaries between components

### 4. Documentation Standards
- **README.md**: Component overview and quick start
- **API.md**: API documentation for components with interfaces
- **ARCHITECTURE.md**: Component architecture decisions
- **TESTING.md**: How to run and maintain tests

### 5. RTM Integration
- Add RTM reference comments in code: ` + "`/* RTM: [SPEC_ID] */`" + `
- Map requirements to implementation files
- Maintain traceability from code back to requirements
- Update RTM when implementation deviates from specs

## Deployment Considerations
- Each component can be deployed independently
- Use container-based deployment when possible
- Include health checks and monitoring
- Implement circuit breakers for component communication
- Plan for graceful degradation when components are unavailable

## Quality Gates
Before considering implementation complete:
1. ✅ All RTM requirements mapped to code
2. ✅ Test coverage meets project standards
3. ✅ Component can be built independently
4. ✅ Component can be deployed independently
5. ✅ Documentation is complete and up-to-date
6. ✅ Security review completed (if applicable)
7. ✅ Performance benchmarks meet requirements

This methodology ensures that code generated from TraceVibe RTM follows consistent, maintainable, and scalable patterns.`
}

type Project struct {
	ID             string  `json:"id"`
	ProjectKey     string  `json:"project_key"`
	Name           string  `json:"name"`
	Description    *string `json:"description"`
	RepositoryURL  *string `json:"repository_url"`
	Version        *string `json:"version"`
	Status         string  `json:"status"`
	ProjectContext *string `json:"project_context"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}