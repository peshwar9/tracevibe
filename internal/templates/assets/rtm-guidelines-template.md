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

### Basic Requirement Template
```json
{
  "id": "REQ-CATEGORY-XXX",
  "phase_id": "phase-x",
  "component_id": "component-identifier",
  "category": "api_endpoint|frontend_ui|database|service|infrastructure",
  "title": "Descriptive Title",
  "description": "Detailed description of functionality",
  "priority": "low|medium|high|critical",
  "status": "not_started|in_progress|completed|blocked",
  "acceptance_criteria": [
    "Specific, testable criteria",
    "Another acceptance criterion"
  ],
  "parent_requirement_id": "optional-parent-id",
  "dependencies": ["list-of-dependent-requirements"],
  "sub_requirements": [...]
}
```

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

This guideline ensures comprehensive, maintainable, and useful Requirements Traceability Matrices that provide real value for project management, development, and quality assurance.