# TraceVibe - Standalone Requirements Traceability CLI

TraceVibe is a lightweight, standalone CLI tool for managing Requirements Traceability Matrix (RTM) data. It uses SQLite for storage and provides an embedded web UI for visualization.

## Features

- **LLM-First Workflow**: Generate guidelines for LLMs to analyze codebases and create RTM data
- **SQLite Database**: Self-contained, no external database required
- **Embedded Web UI**: Built-in HTML templates, no separate frontend stack
- **Hierarchical Requirements**: Scope → User Stories → Tech Specs structure
- **Component-Based Organization**: Track requirements by deployable components
- **Test Mapping**: System tests → Scope, Acceptance tests → User Stories, Unit tests → Tech Specs

## Installation

```bash
# Build from source
go build -o tracevibe

# Or install globally
go install github.com/peshwar9/statsly/tracevibe@latest
```

## Usage

### 1. Generate RTM Guidelines

Generate guidelines document for LLM-assisted RTM creation:

```bash
tracevibe guidelines -o rtm-guidelines.md
```

### 2. Create RTM Data with LLM

Provide the guidelines and your codebase to an LLM (Claude, GPT-4, etc.) with this prompt:

```
Please analyze my codebase and create an RTM JSON/YAML file following the guidelines in rtm-guidelines.md.

Project: [your-project-name]
Repository: [path-to-code]
Tech Stack: [languages/frameworks]
```

### 3. Import RTM Data

Import the LLM-generated RTM file:

```bash
tracevibe import project-rtm.json --project myproject

# Or specify custom database path
tracevibe import project-rtm.yaml --project myproject --db-path ./custom.db
```

### 4. View in Web UI

Start the embedded web server:

```bash
tracevibe serve

# Or specify custom port
tracevibe serve --port 8081
```

Open browser to `http://localhost:8080` (or your specified port)

## RTM Data Structure

### Hierarchical Requirements

```
Component (backend-api, frontend-app, database)
  └── Scope (high-level functionality)
      └── User Stories (user journeys)
          └── Tech Specs (detailed requirements)
```

### Example RTM JSON

```json
{
  "project": {
    "id": "myproject",
    "name": "My Project",
    "description": "Project description"
  },
  "system_components": [
    {
      "id": "backend-api",
      "name": "Backend API Server",
      "component_type": "api_server",
      "technology": "Go"
    }
  ],
  "requirements": [
    {
      "id": "SCOPE-API-1",
      "component_id": "backend-api",
      "requirement_type": "scope",
      "title": "User Authentication",
      "category": "backend_api",
      "children": [
        {
          "id": "STORY-API-1.1",
          "requirement_type": "user_story",
          "title": "User Registration",
          "children": [
            {
              "id": "SPEC-API-1.1.1",
              "requirement_type": "tech_spec",
              "title": "Email validation",
              "implementation": {
                "backend": {
                  "files": [
                    {
                      "path": "internal/auth/register.go",
                      "functions": ["ValidateEmail"]
                    }
                  ]
                }
              },
              "tests": {
                "backend": [
                  {
                    "file": "internal/auth/register_test.go",
                    "functions": ["TestValidateEmail"]
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}
```

## Database Location

By default, TraceVibe stores its SQLite database at:
- macOS/Linux: `~/.tracevibe/tracevibe.db`
- Windows: `%USERPROFILE%\.tracevibe\tracevibe.db`

## Web UI Features

- **Dashboard**: Overview of all imported projects
- **Project View**: Component list and requirements hierarchy
- **Component Details**: Deep dive into component requirements
- **Requirements Tree**: Interactive expandable tree structure
- **Implementation Tracking**: View source files implementing each requirement
- **Test Coverage**: See test cases mapped to requirements

## Architecture

TraceVibe is built with:
- **Go**: Core CLI application
- **Cobra**: Command-line interface framework
- **SQLite**: Embedded database (via go-sqlite3)
- **HTML Templates**: Go's html/template for UI
- **Embedded Assets**: Templates embedded in binary using embed.FS

## Development

### Prerequisites
- Go 1.21+
- SQLite3 (comes with go-sqlite3)

### Building from Source

```bash
git clone https://github.com/peshwar9/statsly.git
cd statsly/rtm-system/tracevibe
go mod download
go build -o tracevibe
```

### Project Structure

```
tracevibe/
├── main.go                 # Entry point
├── cmd/                    # Cobra commands
│   ├── root.go
│   ├── guidelines.go       # Generate guidelines
│   ├── import.go          # Import RTM data
│   ├── serve.go           # Web server
│   └── web/templates/     # HTML templates
├── internal/
│   ├── database/          # SQLite operations
│   │   └── schema.sql     # Database schema
│   ├── models/            # Data structures
│   └── importer/          # Import logic
└── schema.sql             # SQLite schema
```

## Comparison with PostgreSQL Version

The original RTM system used:
- PostgreSQL database
- Next.js/React frontend
- Separate deployment requirements

TraceVibe provides:
- SQLite (embedded, no setup)
- Go HTML templates (single binary)
- Zero-dependency deployment

## Future Enhancements

- [ ] Git integration for branch-aware testing
- [ ] Test execution from UI
- [ ] Requirement editing in UI
- [ ] Export to various formats (PDF, Markdown)
- [ ] Multi-project comparison views
- [ ] Automated RTM updates via CI/CD

## License

MIT