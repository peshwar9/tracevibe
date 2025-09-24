# RTM Generation Request

I need you to analyze my codebase and generate a Requirements Traceability Matrix (RTM) in JSON format following the provided guidelines.

## Context About My Project

Please analyze the codebase I'm providing and:
1. Identify all deployable components (binaries, services, frontends, databases)
2. Extract requirements hierarchically (Scope → User Stories → Tech Specs)
3. Map implementation files and functions to each requirement
4. Identify test files and map them to requirements
5. Follow the exact JSON structure shown in the guidelines

## What I Need From You

Generate a complete RTM JSON file that includes:

### 1. Project Information
- Project name and ID
- Technology stack details
- Repository information

### 2. System Components
Identify REAL deployable components only:
- API servers (separate binaries)
- Frontend applications
- Database systems
- CLI tools
- Worker services
- Microservices

Do NOT create fake components for logical code organization.

### 3. Hierarchical Requirements

For each component, create requirements following this hierarchy:

**SCOPE** (High-level functionality)
└── **USER STORIES** (User journey steps)
    └── **TECH SPECS** (Detailed technical requirements)

Requirements should be:
- Each API endpoint = separate Tech Spec
- Each UI page/screen = separate User Story or Tech Spec
- Each database table = separate Tech Spec
- Each major business function = appropriate level requirement

### 4. Implementation Mapping
For each requirement, identify:
- Source files implementing it
- Specific functions/methods
- Layer (backend/frontend/database)

### 5. Test Coverage
Map test files and functions:
- System/Integration tests → Scope requirements
- Acceptance tests → User Stories
- Unit tests → Tech Specs

## Important Guidelines

1. **Be Specific**: Use actual file paths and function names from the codebase
2. **Be Complete**: Include ALL endpoints, pages, and major functionality
3. **Be Accurate**: Only reference files and functions that actually exist
4. **Follow Structure**: Use the exact JSON format from the guidelines document

## Output Format

Please provide the complete RTM in JSON format that can be directly imported into TraceVibe using:
`tracevibe import your-output.json --project [project-name]`

Start your analysis now and generate the complete RTM JSON.