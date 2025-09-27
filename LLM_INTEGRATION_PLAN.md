# TraceVibe LLM Integration Implementation Plan

## Overview
This feature adds LLM code generation capabilities to TraceVibe by generating context-aware prompts from the RTM database.

## Database Changes ✅ COMPLETED
1. **projects table**: Added `base_llm_prompt` column for storing customizable base prompts
2. **system_components table**: Added `component_path` column for filesystem location

## API Endpoints to Add

### 1. LLM Prompt Generation
```
GET /api/llm-prompt/component/{component_id}
GET /api/llm-prompt/story/{story_id}
GET /api/llm-prompt/spec/{spec_id}
```

Response format:
```json
{
  "prompt": "Generated prompt text...",
  "context_level": "component|story|spec",
  "included_items": {
    "stories": 5,
    "specs": 12
  }
}
```

### 2. Source Tree Viewing
```
GET /api/component/{component_id}/source-tree
```

Response format:
```json
{
  "component_path": "/components/backend-api",
  "tree": [
    {
      "path": "cmd/main.go",
      "type": "file",
      "rtm_refs": ["STORY-1.1"]
    },
    {
      "path": "internal/auth",
      "type": "directory",
      "children": [...]
    }
  ]
}
```

### 3. Test Suite Viewing
```
GET /api/component/{component_id}/test-suite
```

Response format:
```json
{
  "tests": [
    {
      "file": "auth_test.go",
      "test_count": 5,
      "test_names": [
        "TestValidateEmail",
        "TestValidatePassword"
      ],
      "rtm_refs": ["SPEC-1.1.1", "SPEC-1.1.2"]
    }
  ],
  "total_tests": 23
}
```

### 4. Base Prompt Management
```
GET /api/project/{project_id}/base-prompt
PUT /api/project/{project_id}/base-prompt
```

## UI Components to Add

### 1. Buttons in Project View
- Component level: [📁 Source Tree] [🧪 Test Suite] [🤖 Generate Code]
- Story level: [🤖 Generate Story Code]
- Spec level: [🤖 Generate Spec Code]

### 2. Modals

#### LLM Prompt Modal
- Displays generated prompt with hierarchy context
- Editable text area
- Copy to clipboard button
- Shows count of included requirements

#### Source Tree Modal
- Interactive file tree display
- Shows RTM references for each file
- Updates implementation mappings in DB

#### Test Suite Modal
- List of test files with counts
- Shows test names/descriptions
- Links to RTM requirements

### 3. Base Prompt Editor
- In project settings or dashboard
- Textarea for editing base prompt template
- Save button to update database

## Prompt Generation Logic

### Component Level
```
Base Project Prompt
+ Component Context (name, type, tech, description)
+ All Scopes in component
  + All User Stories in each scope
    + All Tech Specs in each story
```

### Story Level
```
Base Project Prompt
+ Parent Component Context
+ Parent Scope Context
+ Story Context
  + All Tech Specs in story
```

### Spec Level
```
Base Project Prompt
+ Parent Component Context
+ Parent Scope Context
+ Parent Story Context
+ Tech Spec Context
```

## Implementation Steps

1. ✅ Update database schema (DONE)
2. ⏳ Create API handlers in serve.go
3. ⏳ Add UI buttons to project-page.html
4. ⏳ Create modals for prompt/tree/test views
5. ⏳ Add JavaScript for modal interactions
6. ⏳ Add base prompt editor UI
7. ⏳ Test with sample data

## Files to Modify

- `/cmd/serve.go` - Add API handlers
- `/cmd/web/templates/project-page.html` - Add buttons and modals
- `/internal/database/sqlite.go` - Add query methods
- `/internal/models/` - Add structs for API responses

## Sample Generated Prompt

```
PROJECT: StatsLy Analytics Platform
ARCHITECTURE: Component-based monorepo
CODING STANDARDS:
- Use descriptive variable names
- Include error handling and logging
- Add RTM reference comments /* RTM: [SPEC_ID] */
- Write tests for all functions
- Follow language-specific conventions

COMPONENT: Backend API Server
TYPE: api_server
TECHNOLOGY: Go
DESCRIPTION: RESTful API server for analytics data processing
TAGS: ["backend", "go", "api"]

REQUIREMENTS TO IMPLEMENT:
└── Scope: User Authentication (SCOPE-API-1)
    └── User Story: User Registration (STORY-API-1.1)
        ├── Tech Spec: Email validation (SPEC-API-1.1.1)
        │   Requirements: Validate email format, check uniqueness
        └── Tech Spec: Password validation (SPEC-API-1.1.2)
            Requirements: Min 8 chars, uppercase, lowercase, number

DELIVERABLES:
1. Implementation code for all tech specs
2. Unit tests with RTM comments
3. Integration test for complete user story
4. Error handling and logging

Generate production-ready Go code following the above specifications.
```

## Benefits

1. **Consistency**: All generated code follows same structure
2. **Context-Aware**: LLM gets full requirement hierarchy
3. **Traceability**: RTM references in generated code
4. **Flexibility**: Editable prompts before sending to LLM
5. **Efficiency**: One-click prompt generation from UI