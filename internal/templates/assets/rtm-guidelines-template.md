# Requirements Traceability Matrix (RTM) Guidelines

## Overview
This document provides comprehensive guidelines for creating detailed and granular Requirements Traceability Matrix (RTM) documents for software projects. The goal is to ensure complete traceability from high-level features down to individual code implementations and test cases.

## Core Principles

### 1. Granularity Level
- **Each discrete functionality should be its own requirement**
- Avoid grouping multiple distinct features into single requirements
- Requirements should be testable and implementable independently
- Use parent-child relationships to maintain hierarchical organization

### 2. Component-Based Organization
- Organize requirements by system components (microservices, frontend modules, databases, etc.)
- Each component should have clear boundaries and responsibilities
- Components should align with actual system architecture

## Requirement Categories and Guidelines

### API/Backend Requirements

#### API Endpoints
- **Rule**: Each API endpoint is a separate requirement
- **Format**: `API-[Component]-[Sequence]` (e.g., `API-AUTH-001`, `API-USER-001`)
- **Examples**:
  - `API-AUTH-001`: POST /auth/register (User Registration)
  - `API-AUTH-002`: POST /auth/login (User Login)
  - `API-AUTH-003`: POST /auth/logout (User Logout)
  - `API-AUTH-004`: GET /auth/verify (Token Verification)
  - `API-USER-001`: GET /users/profile (Get User Profile)
  - `API-USER-002`: PUT /users/profile (Update User Profile)

#### Sub-Requirements for API Endpoints
- Request validation
- Authentication/authorization checks
- Business logic processing
- Database operations
- Response formatting
- Error handling

### Authentication Requirements

#### Core Authentication Functions
- **Rule**: Each authentication action is a separate requirement
- **Examples**:
  - `AUTH-REG-001`: User Registration Process
  - `AUTH-LOGIN-001`: User Login Process
  - `AUTH-LOGOUT-001`: User Logout Process
  - `AUTH-RESET-001`: Password Reset Process
  - `AUTH-VERIFY-001`: Email Verification Process

#### Authentication Sub-Requirements
- **Password Security**: Password hashing (bcrypt, scrypt, etc.)
- **Token Management**: JWT generation, validation, refresh
- **Session Management**: Session creation, validation, cleanup
- **Security Measures**: Rate limiting, brute force protection
- **Multi-factor Authentication**: SMS, email, authenticator app support

### Frontend/UI Requirements

#### Screen/Page Requirements
- **Rule**: Each distinct screen or page is a requirement
- **Format**: `UI-[Module]-[Sequence]`
- **Examples**:
  - `UI-AUTH-001`: Registration Page
  - `UI-AUTH-002`: Login Page
  - `UI-AUTH-003`: Password Reset Page
  - `UI-DASH-001`: Main Dashboard
  - `UI-PROFILE-001`: User Profile Page

#### Component Requirements
- **Rule**: Each reusable UI component is a requirement
- **Examples**:
  - `UI-COMP-001`: Navigation Header Component
  - `UI-COMP-002`: Data Table Component
  - `UI-COMP-003`: Modal Dialog Component
  - `UI-COMP-004`: Form Validation Component

#### Interaction Requirements
- **Rule**: Each significant user interaction is a sub-requirement
- **Examples**:
  - Button clicks with distinct actions
  - Form submissions
  - Data filtering/sorting
  - Modal open/close actions
  - Navigation events

### Database Requirements

#### Table Requirements
- **Rule**: Each database table is a requirement
- **Format**: `DB-[Table]-[Sequence]`
- **Examples**:
  - `DB-USER-001`: Users Table Schema
  - `DB-ANCHOR-001`: Anchors Table Schema
  - `DB-SHORTLINK-001`: Short Links Table Schema
  - `DB-ANALYTICS-001`: Analytics Events Table Schema

#### Database Sub-Requirements
- **Schema Definition**: Column types, constraints, indexes
- **Data Migrations**: Schema changes, data transformations
- **CRUD Operations**: Create, Read, Update, Delete operations
- **Query Optimization**: Indexes, query performance
- **Data Validation**: Constraints, triggers, stored procedures

### Service/Business Logic Requirements

#### Service Operations
- **Rule**: Each business service or operation is a requirement
- **Format**: `SVC-[Service]-[Sequence]`
- **Examples**:
  - `SVC-LINK-001`: Short Link Generation Service
  - `SVC-ANALYTICS-001`: Click Analytics Service
  - `SVC-EMAIL-001`: Email Notification Service
  - `SVC-VALIDATION-001`: Data Validation Service

### Infrastructure/Utility Requirements

#### System Utilities
- **Examples**:
  - `UTIL-CONFIG-001`: Configuration Management
  - `UTIL-LOG-001`: Logging System
  - `UTIL-MONITOR-001`: Health Monitoring
  - `UTIL-BACKUP-001`: Database Backup System

## Requirement Structure

### Complete RTM JSON Structure for TraceviBe Import

TraceviBe supports two JSON formats for requirements organization:

#### Format 1: Flat Requirements Array (Legacy)
Use this format for simple requirement lists with explicit parent-child relationships.

#### Format 2: Nested Scopes Hierarchy (Recommended)
Use this format for organized scope → user stories → tech specs structure.

When generating RTM files for TraceviBe import, use one of these exact JSON structures:

```json
{
  "rtm_version": "1.0.0",
  "metadata": {
    "generated_at": "2024-09-24T12:00:00Z",
    "generated_by": "Claude Code RTM Generator",
    "project": {
      "name": "Your Project Name",
      "id": "your-project-key",
      "repository": "github.com/user/repo",
      "version": "1.0.0",
      "description": "Project description"
    }
  },
  "components": [
    {
      "id": "COMP-001",
      "name": "API Server",
      "type": "backend_service",
      "deployment_unit": "server",
      "path": "cmd/server",
      "technology": "Go",
      "description": "Main REST API server",
      "entry_point": "cmd/server/main.go"
    },
    {
      "id": "COMP-002",
      "name": "Frontend Application",
      "type": "web_application",
      "deployment_unit": "frontend",
      "path": "frontend",
      "technology": "Next.js",
      "description": "Web application for user interaction",
      "entry_point": "frontend/src/app/page.tsx"
    }
  ],
  "requirements": [
    {
      "id": "REQ-001",
      "type": "SCOPE",
      "name": "User Authentication and Management",
      "description": "System shall provide secure user authentication",
      "component_id": "COMP-001",
      "priority": "HIGH",
      "status": "IMPLEMENTED",
      "children": [
        {
          "id": "REQ-001-US-001",
          "type": "USER_STORY",
          "name": "User Registration",
          "description": "As a new user, I want to create an account",
          "children": [
            {
              "id": "REQ-001-US-001-TS-001",
              "type": "TECH_SPEC",
              "name": "Register API Endpoint",
              "description": "POST /auth/register endpoint for user registration",
              "implementation": {
                "files": [
                  {
                    "path": "internal/api/server.go",
                    "functions": ["handleRegister"],
                    "lines": "295-368"
                  }
                ],
                "layer": "backend"
              },
              "test_coverage": {
                "unit_tests": [
                  {
                    "file": "internal/api/server_test.go",
                    "functions": ["TestRegisterEndpoint"]
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ],
  "traceability_matrix": {
    "summary": {
      "total_requirements": {
        "SCOPE": 5,
        "USER_STORY": 15,
        "TECH_SPEC": 25
      },
      "total_test_cases": {
        "unit_tests": 25,
        "integration_tests": 12,
        "e2e_tests": 5,
        "total": 42
      },
      "total_implementations": {
        "backend_files": 15,
        "frontend_files": 8,
        "database_files": 5,
        "total_files": 28
      }
    }
  }
}
```

#### Format 2: Nested Scopes Hierarchy Structure

```json
{
  "rtm_version": "1.0.0",
  "metadata": {
    "generated_at": "2024-09-24T12:00:00Z",
    "generated_by": "Claude Code RTM Generator",
    "project": {
      "name": "Your Project Name",
      "id": "your-project-key",
      "repository": "github.com/user/repo",
      "version": "1.0.0",
      "description": "Project description"
    }
  },
  "components": [
    {
      "id": "COMP-001",
      "name": "API Server",
      "type": "backend_service",
      "deployment_unit": "server",
      "path": "cmd/server",
      "technology": "Go",
      "description": "Main REST API server",
      "entry_point": "cmd/server/main.go"
    },
    {
      "id": "COMP-002",
      "name": "Frontend Application",
      "type": "web_application",
      "deployment_unit": "frontend",
      "path": "frontend",
      "technology": "Next.js",
      "description": "Web application for user interface",
      "entry_point": "frontend/src/app/page.tsx"
    }
  ],
  "scopes": [
    {
      "id": "SCOPE-001",
      "component_id": "COMP-001",
      "name": "User Authentication and Management",
      "description": "System shall provide secure user authentication",
      "priority": "HIGH",
      "status": "IMPLEMENTED",
      "user_stories": [
        {
          "id": "US-001",
          "name": "User Registration",
          "description": "As a new user, I want to create an account so that I can access the system",
          "priority": "HIGH",
          "status": "IMPLEMENTED",
          "tech_specs": [
            {
              "id": "TS-001-001",
              "name": "Register API Endpoint",
              "description": "POST /auth/register endpoint for user registration",
              "priority": "HIGH",
              "status": "IMPLEMENTED",
              "acceptance_criteria": [
                "Endpoint accepts valid user registration data",
                "Returns appropriate success/error responses",
                "Validates input data format"
              ],
              "implementation": {
                "files": [
                  {
                    "path": "internal/api/handlers.go",
                    "functions": ["HandleUserRegistration", "ValidateUserInput"],
                    "lines": "45-78"
                  }
                ],
                "layer": "backend"
              },
              "test_coverage": {
                "unit_tests": [
                  {
                    "file": "internal/api/handlers_test.go",
                    "functions": ["TestHandleUserRegistration", "TestValidateUserInput"]
                  }
                ]
              }
            },
            {
              "id": "TS-001-002",
              "name": "User Registration Form",
              "description": "Frontend form for user registration",
              "priority": "HIGH",
              "status": "IMPLEMENTED",
              "implementation": {
                "files": [
                  {
                    "path": "src/components/RegistrationForm.tsx",
                    "functions": ["RegistrationForm", "handleSubmit"],
                    "lines": "12-45"
                  }
                ],
                "layer": "frontend"
              }
            }
          ]
        },
        {
          "id": "US-002",
          "name": "User Login",
          "description": "As a registered user, I want to log into the system",
          "priority": "HIGH",
          "status": "IMPLEMENTED",
          "tech_specs": [
            {
              "id": "TS-002-001",
              "name": "Login API Endpoint",
              "description": "POST /auth/login endpoint for user authentication",
              "priority": "HIGH",
              "status": "IMPLEMENTED"
            }
          ]
        }
      ]
    },
    {
      "id": "SCOPE-002",
      "component_id": "COMP-002",
      "name": "User Interface Management",
      "description": "System shall provide intuitive user interface",
      "priority": "MEDIUM",
      "status": "IN_PROGRESS",
      "user_stories": [
        {
          "id": "US-003",
          "name": "Dashboard Display",
          "description": "As a user, I want to see a dashboard with key information",
          "tech_specs": [
            {
              "id": "TS-003-001",
              "name": "Main Dashboard Component",
              "description": "React component for main user dashboard"
            }
          ]
        }
      ]
    }
  ]
}
```

### Key Field Mappings for TraceviBe Import

**IMPORTANT**: Use these exact field names for successful import:

**Top Level Structure:**
- `metadata.project` - Project information
- `components` - System components (NOT `system_components`)
- `requirements` - Requirements array (Format 1: Flat structure)
- `scopes` - Scopes array (Format 2: Nested structure)

### Format 1: Flat Requirements Structure

**Requirements (Flat Format):**
- `type` - Requirement type: `"SCOPE"`, `"USER_STORY"`, or `"TECH_SPEC"` (NOT `requirement_type`)
- `name` - Requirement title (NOT `title`)
- `description` - Requirement description
- `component_id` - **MANDATORY** Reference to component ID
- `children` - Child requirements array

### Format 2: Nested Scopes Structure

**Scopes (Nested Format):**
- `id` - **MANDATORY** Unique scope identifier
- `component_id` - **MANDATORY** Reference to component ID (e.g., "COMP-001")
- `name` - Scope title
- `description` - Scope description
- `priority` - Priority level
- `status` - Implementation status
- `user_stories` - **MANDATORY** Array of user stories within this scope

**User Stories (within Scopes):**
- `id` - **MANDATORY** Unique user story identifier
- `name` - User story title
- `description` - User story description (should follow "As a [user], I want [goal]" format)
- `priority` - Priority level
- `status` - Implementation status
- `tech_specs` - **MANDATORY** Array of tech specs implementing this user story

**Tech Specs (within User Stories):**
- `id` - **MANDATORY** Unique tech spec identifier
- `name` - Tech spec title
- `description` - Technical specification description
- `priority` - Priority level
- `status` - Implementation status
- `acceptance_criteria` - Array of acceptance criteria strings
- `implementation` - Implementation details (files, functions, layers)
- `test_coverage` - Test coverage information

### Mandatory Field Requirements

**Critical for TraceviBe Import:**

1. **Component Assignment**: Every scope MUST have a valid `component_id` that references an existing component
2. **Hierarchical Structure**:
   - Each scope MUST contain `user_stories` array
   - Each user story MUST contain `tech_specs` array
   - Empty arrays are allowed but the fields must be present
3. **Unique IDs**: All IDs must be unique within their scope (scope IDs, user story IDs, tech spec IDs)
4. **Parent-Child Relationships**: The nested structure automatically establishes parent-child relationships

**Components:**
- `type` - Component type like `"backend_service"`, `"web_application"` (NOT `component_type`)
- `deployment_unit` - Deployment identifier
- `entry_point` - Main file path

**Implementation Tracking:**
- `implementation.files` - Array of files implementing this requirement
- `test_coverage.unit_tests` - Unit tests covering this requirement

### Parent-Child Relationships

#### Feature-Level Parents
- High-level features that encompass multiple requirements
- **Example**: "User Management Feature" contains registration, login, profile management

#### Component-Level Parents
- Major system components that contain multiple sub-features
- **Example**: "Authentication System" contains all auth-related requirements

#### Flow-Level Parents
- User workflows that span multiple screens/endpoints
- **Example**: "User Onboarding Flow" contains registration, verification, profile setup

## Component Assignment Guidelines

### System Components
Define clear system components that align with actual deployable units or major architectural boundaries:

**Components represent:**
- Separate deployable binaries/services
- Independent frontend applications
- External systems/databases
- Major infrastructure boundaries

**Components do NOT represent:**
- Logical code modules within a binary
- Database tables or schemas (these are requirements)
- Code packages or namespaces
- Functional groupings (these become requirement categories)

#### Examples for Different Architectures:

**Monolithic Application**:
- `backend-api`: Single API server binary
- `frontend-app`: Frontend application
- `database`: External PostgreSQL database
- `migration-tool`: Database migration utility (if separate binary)

**Microservices Architecture**:
- `user-service`: User management microservice
- `auth-service`: Authentication microservice (only if actually separate)
- `product-service`: Product management microservice
- `frontend-app`: Main frontend application
- `admin-frontend`: Admin dashboard (if separate deployment)
- `message-queue`: Redis/RabbitMQ external system

**Correct Assignment Examples**:
- ✅ User registration API endpoint → `backend-api` component
- ✅ User login page → `frontend-app` component
- ✅ Database migration → `migration-tool` component (or `backend-api` if same binary)
- ✅ Users table schema → `backend-api` component (as a database requirement)

**Incorrect Assignment Examples**:
- ❌ "Authentication service" as separate component (when it's just code within main API)
- ❌ "Database layer" as separate component (when it's just ORM code within API)
- ❌ "Data models" as separate component (when they're just structs/classes)

## Implementation Traceability

### Code Mapping
Each requirement should map to:
- **Specific files**: Exact file paths in codebase
- **Functions/methods**: Specific functions that implement the requirement
- **Line ranges**: Approximate line numbers (for reference)
- **Components**: For frontend, specific React/Vue components

### Test Traceability
Each requirement should have:
- **Unit tests**: Testing individual functions/methods
- **Integration tests**: Testing requirement end-to-end
- **API tests**: For backend endpoints
- **UI tests**: For frontend interactions
- **E2E tests**: Testing complete user workflows

## Quality Guidelines

### Requirement Quality Checklist
- [ ] Is the requirement independently testable?
- [ ] Is the requirement independently implementable?
- [ ] Does the requirement have clear acceptance criteria?
- [ ] Are all dependencies clearly identified?
- [ ] Is the requirement assigned to the correct component?
- [ ] Are implementation files and tests properly mapped?

### Common Anti-Patterns to Avoid
- **Overly Broad Requirements**: Grouping multiple distinct features
- **Implementation Details in Requirements**: Focus on "what" not "how"
- **Missing Dependencies**: Not identifying requirement relationships
- **Inconsistent Granularity**: Mixing high-level and low-level requirements
- **Poor Component Assignment**: Requirements assigned to wrong components

## Maintenance Guidelines

### Regular Updates
- Review RTM during sprint planning
- Update status as requirements are implemented
- Add new requirements as features are added
- Update implementation mappings as code changes

### Validation Procedures
- Verify all requirements have implementations
- Ensure all implementations have tests
- Check for orphaned requirements (no implementation)
- Validate component assignments match actual architecture

## Tools and Automation

### Recommended Tools
- **JSON Schema Validation**: Ensure RTM data structure consistency
- **Automated Sync**: Scripts to sync with code changes
- **Dashboard Views**: Visual RTM management interface
- **Report Generation**: Status reports and metrics

### Integration Points
- Link with project management tools (Jira, GitHub Issues)
- Integration with CI/CD for automated validation
- Code analysis tools for implementation discovery
- Test coverage tools for test traceability

## LLM RTM Generation Instructions

### Generation Process
1. **Analyze the codebase** - Examine source code, file structure, and existing documentation
2. **Identify system components** - Map to actual deployable units and architectural boundaries
3. **Choose format** - Use nested scopes format (recommended) or flat requirements format (legacy)
4. **Extract requirements hierarchically** - SCOPE → USER_STORY → TECH_SPEC
5. **Assign components** - Every scope/requirement MUST have a valid component_id
6. **Map implementations** - Link requirements to actual code files and functions
7. **Generate valid JSON** - Use exact field names and structure for TraceviBe import

### Requirement Type Guidelines

**SCOPE Requirements (Format 1: Flat) / Scopes (Format 2: Nested):**
- High-level system features or modules
- Business capabilities or functional areas
- Each major system component should have 3-8 SCOPE requirements
- **MANDATORY**: Must have valid `component_id` referencing existing component
- Examples: "User Management", "Payment Processing", "Content Management"

**USER_STORY Requirements:**
- User-facing functionality within each SCOPE
- Should follow "As a [user], I want [goal] so that [benefit]" format
- Each SCOPE should have 2-5 USER_STORY requirements
- Examples: "User Registration", "Password Reset", "Profile Management"

**TECH_SPEC Requirements:**
- Technical implementation details for each USER_STORY
- API endpoints, UI components, database operations
- Each USER_STORY should have 1-4 TECH_SPEC requirements
- Examples: "Register API Endpoint", "Login Form Component", "User Table Schema"

### Implementation Mapping Rules

**For Backend/API Requirements:**
```json
"implementation": {
  "files": [
    {
      "path": "internal/api/handlers.go",
      "functions": ["HandleUserRegistration", "ValidateUserInput"],
      "lines": "45-78"
    }
  ],
  "layer": "backend"
}
```

**For Frontend Requirements:**
```json
"implementation": {
  "files": [
    {
      "path": "src/components/LoginForm.tsx",
      "functions": ["LoginForm", "handleSubmit"],
      "lines": "12-45"
    }
  ],
  "layer": "frontend"
}
```

**For Database Requirements:**
```json
"implementation": {
  "files": [
    {
      "path": "internal/models/user.go",
      "functions": ["CreateUser", "ValidateUser"],
      "lines": "25-60"
    }
  ],
  "layer": "database"
}
```

### Test Coverage Mapping

**Unit Tests:**
```json
"test_coverage": {
  "unit_tests": [
    {
      "file": "internal/api/handlers_test.go",
      "functions": ["TestHandleUserRegistration", "TestValidateUserInput"]
    }
  ]
}
```

**Integration Tests:**
```json
"test_coverage": {
  "integration_tests": [
    {
      "file": "tests/integration/user_test.go",
      "functions": ["TestUserRegistrationFlow"]
    }
  ]
}
```

### Component Identification Guide

**Backend Services:**
- Type: `"backend_service"`, `"api_server"`, `"microservice"`
- Look for: main.go files, server binaries, API routes
- Examples: REST APIs, GraphQL servers, gRPC services

**Frontend Applications:**
- Type: `"web_application"`, `"mobile_app"`, `"desktop_app"`
- Look for: package.json, index.html, main component files
- Examples: React apps, Vue apps, mobile apps

**Databases:**
- Type: `"database"`, `"data_store"`
- Look for: schema files, migration files, database models
- Examples: PostgreSQL, MongoDB, Redis

**CLI Tools:**
- Type: `"cli_tool"`, `"utility"`
- Look for: cmd/ directories, command-line interfaces
- Examples: Migration tools, admin utilities

**Infrastructure:**
- Type: `"infrastructure"`, `"deployment"`
- Look for: Docker files, K8s configs, deployment scripts
- Examples: Container orchestration, CI/CD pipelines

### Quality Checklist for Generated RTM

- [ ] All requirement IDs follow consistent naming pattern
- [ ] Each SCOPE has multiple USER_STORY children
- [ ] Each USER_STORY has multiple TECH_SPEC children
- [ ] All requirements have proper `component_id` references
- [ ] Implementation mappings point to actual files
- [ ] Test coverage references real test files
- [ ] Component types match actual system architecture
- [ ] JSON structure matches TraceviBe import format exactly

### Common Generation Mistakes to Avoid

❌ **Wrong field names**: Using `"requirement_type"` instead of `"type"`
❌ **Flat structure**: Not using hierarchical parent-child relationships
❌ **Generic implementations**: Not mapping to specific files and functions
❌ **Missing component assignments**: Requirements without `component_id`
❌ **Inconsistent naming**: Mixed naming patterns for requirement IDs
❌ **Overly broad requirements**: Grouping multiple distinct features
❌ **Missing test coverage**: No test mapping for implemented features

This guideline ensures comprehensive, maintainable, and useful Requirements Traceability Matrices that provide real value for project management, development, and quality assurance.