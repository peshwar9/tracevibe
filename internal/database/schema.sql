-- Requirements Traceability Matrix (RTM) Database Schema - SQLite Version
-- Generic schema to support multiple projects and tech stacks

-- Enable foreign key constraints
PRAGMA foreign_keys = ON;

-- Projects table - stores information about each project being tracked
CREATE TABLE projects (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_key TEXT UNIQUE NOT NULL, -- e.g., 'statsly', 'project-alpha'
    name TEXT NOT NULL,
    description TEXT,
    repository_url TEXT,
    version TEXT,
    status TEXT DEFAULT 'active', -- active, archived, deprecated
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Tech stack configuration for each project
CREATE TABLE project_tech_stacks (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    layer TEXT NOT NULL, -- 'backend', 'frontend', 'mobile', 'database'
    language TEXT, -- 'Go', 'TypeScript', 'Python', etc.
    framework TEXT, -- 'Next.js', 'React', 'Express', etc.
    testing_framework TEXT, -- 'Jest', 'Go testing', 'pytest', etc.
    additional_info TEXT, -- Store any additional tech stack details as JSON
    created_at TEXT DEFAULT (datetime('now'))
);

-- System components for each project (real deployable units)
CREATE TABLE system_components (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    component_key TEXT NOT NULL, -- 'backend-api', 'frontend-app', 'database'
    name TEXT NOT NULL,
    component_type TEXT NOT NULL, -- 'api_server', 'frontend', 'database', 'cli_tool'
    technology TEXT, -- 'Go', 'React', 'PostgreSQL'
    description TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, component_key)
);

-- Development phases for each project
CREATE TABLE phases (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phase_key TEXT NOT NULL, -- 'phase-1', 'mvp', 'v2.0'
    name TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'planning', -- planning, in_progress, completed, cancelled
    start_date TEXT,
    end_date TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, phase_key)
);

-- Hierarchical requirements structure: Scope -> User Stories -> Tech Specs
CREATE TABLE requirements (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    component_id TEXT NOT NULL REFERENCES system_components(id) ON DELETE CASCADE,
    phase_id TEXT REFERENCES phases(id) ON DELETE SET NULL,
    parent_requirement_id TEXT REFERENCES requirements(id) ON DELETE CASCADE,
    requirement_key TEXT NOT NULL, -- 'SCOPE-1', 'STORY-1.1', 'SPEC-1.1.1'
    requirement_type TEXT NOT NULL, -- 'scope', 'user_story', 'tech_spec'
    title TEXT NOT NULL,
    description TEXT,
    category TEXT NOT NULL, -- 'database', 'backend_api', 'frontend_ui', 'security'
    priority TEXT DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    status TEXT DEFAULT 'not_started', -- 'not_started', 'in_progress', 'completed', 'blocked'
    acceptance_criteria TEXT, -- JSON array as text
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, requirement_key)
);

-- Code implementations - tracks which files implement each requirement
CREATE TABLE implementations (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    requirement_id TEXT NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
    layer TEXT NOT NULL, -- 'backend', 'frontend', 'database', 'infrastructure'
    file_path TEXT NOT NULL,
    functions TEXT, -- JSON array as text
    line_ranges TEXT, -- JSON array as text like ['10-25', '45-60']
    components TEXT, -- For frontend: component names as JSON array
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- API endpoints tracking
CREATE TABLE api_endpoints (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    method TEXT NOT NULL, -- 'GET', 'POST', 'PUT', 'DELETE'
    path TEXT NOT NULL,
    handler_file TEXT,
    handler_function TEXT,
    description TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, method, path)
);

-- API usage tracking - which frontend components use which endpoints
CREATE TABLE api_usage (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    endpoint_id TEXT NOT NULL REFERENCES api_endpoints(id) ON DELETE CASCADE,
    implementation_id TEXT NOT NULL REFERENCES implementations(id) ON DELETE CASCADE,
    usage_context TEXT, -- 'form submission', 'data fetch', 'real-time update'
    created_at TEXT DEFAULT (datetime('now'))
);

-- Test files and functions
CREATE TABLE test_files (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    test_type TEXT, -- 'unit', 'integration', 'e2e', 'component'
    layer TEXT, -- 'backend', 'frontend'
    framework TEXT, -- 'Go testing', 'Jest', 'Cypress'
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, file_path)
);

-- Individual test functions/cases
CREATE TABLE test_cases (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    test_file_id TEXT NOT NULL REFERENCES test_files(id) ON DELETE CASCADE,
    test_name TEXT NOT NULL,
    test_type TEXT, -- 'unit', 'integration', 'e2e', 'system', 'acceptance'
    description TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(test_file_id, test_name)
);

-- Links test cases to requirements (with hierarchical mapping)
-- System tests -> Scope, Acceptance tests -> User Stories, Unit tests -> Tech Specs
CREATE TABLE requirement_test_coverage (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    requirement_id TEXT NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
    test_case_id TEXT NOT NULL REFERENCES test_cases(id) ON DELETE CASCADE,
    coverage_type TEXT, -- 'requirement', 'implementation', 'integration'
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(requirement_id, test_case_id)
);

-- Frontend components catalog
CREATE TABLE frontend_components (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    component_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    component_type TEXT, -- 'page', 'component', 'context', 'hook', 'utility'
    description TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    UNIQUE(project_id, file_path)
);

-- Tracks which components use which API endpoints
CREATE TABLE component_api_dependencies (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    component_id TEXT NOT NULL REFERENCES frontend_components(id) ON DELETE CASCADE,
    endpoint_id TEXT NOT NULL REFERENCES api_endpoints(id) ON DELETE CASCADE,
    dependency_type TEXT, -- 'direct', 'indirect', 'conditional'
    usage_description TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Audit trail for requirement changes
CREATE TABLE requirement_changes (
    id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
    requirement_id TEXT NOT NULL REFERENCES requirements(id) ON DELETE CASCADE,
    change_type TEXT, -- 'created', 'updated', 'status_changed', 'deleted'
    old_values TEXT, -- JSON as text
    new_values TEXT, -- JSON as text
    changed_by TEXT,
    change_reason TEXT,
    created_at TEXT DEFAULT (datetime('now'))
);

-- Indexes for performance
CREATE INDEX idx_requirements_project_id ON requirements(project_id);
CREATE INDEX idx_requirements_component_id ON requirements(component_id);
CREATE INDEX idx_requirements_phase_id ON requirements(phase_id);
CREATE INDEX idx_requirements_parent_id ON requirements(parent_requirement_id);
CREATE INDEX idx_requirements_category ON requirements(category);
CREATE INDEX idx_requirements_status ON requirements(status);
CREATE INDEX idx_requirements_type ON requirements(requirement_type);
CREATE INDEX idx_implementations_requirement_id ON implementations(requirement_id);
CREATE INDEX idx_implementations_layer ON implementations(layer);
CREATE INDEX idx_api_endpoints_project_id ON api_endpoints(project_id);
CREATE INDEX idx_test_files_project_id ON test_files(project_id);
CREATE INDEX idx_test_cases_test_file_id ON test_cases(test_file_id);
CREATE INDEX idx_frontend_components_project_id ON frontend_components(project_id);

-- Views for common queries

-- Requirement overview with implementation and test status
CREATE VIEW requirement_overview AS
SELECT
    r.id,
    p.project_key,
    c.component_key,
    r.requirement_key,
    r.requirement_type,
    r.title,
    r.category,
    r.status,
    r.priority,
    ph.name as phase_name,
    COUNT(DISTINCT child.id) as child_requirements_count,
    COUNT(DISTINCT i.id) as implementation_count,
    COUNT(DISTINCT rtc.test_case_id) as test_cases_count
FROM requirements r
JOIN projects p ON r.project_id = p.id
JOIN system_components c ON r.component_id = c.id
LEFT JOIN phases ph ON r.phase_id = ph.id
LEFT JOIN requirements child ON r.id = child.parent_requirement_id
LEFT JOIN implementations i ON r.id = i.requirement_id
LEFT JOIN requirement_test_coverage rtc ON r.id = rtc.requirement_id
GROUP BY r.id, p.project_key, c.component_key, r.requirement_key, r.requirement_type, r.title, r.category, r.status, r.priority, ph.name;

-- API endpoint usage tracking
CREATE VIEW api_endpoint_usage AS
SELECT
    ae.method,
    ae.path,
    ae.handler_file,
    ae.handler_function,
    COUNT(DISTINCT fc.id) as frontend_components_using,
    GROUP_CONCAT(DISTINCT fc.component_name) as component_names
FROM api_endpoints ae
LEFT JOIN component_api_dependencies cad ON ae.id = cad.endpoint_id
LEFT JOIN frontend_components fc ON cad.component_id = fc.id
GROUP BY ae.id, ae.method, ae.path, ae.handler_file, ae.handler_function;

-- Test coverage summary by requirement
CREATE VIEW test_coverage_summary AS
SELECT
    r.requirement_key,
    r.title,
    r.requirement_type,
    COUNT(DISTINCT rtc.test_case_id) as total_test_cases,
    COUNT(DISTINCT CASE WHEN tf.layer = 'backend' THEN rtc.test_case_id END) as backend_tests,
    COUNT(DISTINCT CASE WHEN tf.layer = 'frontend' THEN rtc.test_case_id END) as frontend_tests
FROM requirements r
LEFT JOIN requirement_test_coverage rtc ON r.id = rtc.requirement_id
LEFT JOIN test_cases tc ON rtc.test_case_id = tc.id
LEFT JOIN test_files tf ON tc.test_file_id = tf.id
GROUP BY r.id, r.requirement_key, r.title, r.requirement_type;

-- Component summary view
CREATE VIEW component_summary AS
SELECT
    c.id,
    c.component_key,
    c.name,
    c.component_type,
    c.technology,
    p.project_key,
    COUNT(DISTINCT r.id) as total_requirements,
    COUNT(DISTINCT CASE WHEN r.requirement_type = 'scope' THEN r.id END) as scope_count,
    COUNT(DISTINCT CASE WHEN r.requirement_type = 'user_story' THEN r.id END) as user_story_count,
    COUNT(DISTINCT CASE WHEN r.requirement_type = 'tech_spec' THEN r.id END) as tech_spec_count,
    COUNT(DISTINCT i.id) as implementation_count,
    COUNT(DISTINCT rtc.test_case_id) as test_case_count
FROM system_components c
JOIN projects p ON c.project_id = p.id
LEFT JOIN requirements r ON c.id = r.component_id
LEFT JOIN implementations i ON r.id = i.requirement_id
LEFT JOIN requirement_test_coverage rtc ON r.id = rtc.requirement_id
GROUP BY c.id, c.component_key, c.name, c.component_type, c.technology, p.project_key;