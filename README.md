# TraceVibe

TraceVibe helps you create and manage Requirements Traceability Matrix (RTM) for projects with AI-generated code.

## Features

- **AI-First Workflow**: Generate RTM documentation from existing codebases using LLMs
- **Component-Based Organization**: Track requirements by system components with tags
- **Hierarchical Requirements**: Scope → User Stories → Tech Specs structure
- **Web UI**: Built-in interface for viewing and managing RTM data
- **Self-Contained**: SQLite database, no external dependencies

## Installation & Usage

### Option 1: Docker (Recommended)

```bash
# Clone and build
git clone <your-repo-url>
cd tracevibe
docker build -t tracevibe:latest .

# Run with persistent data
docker run -p 8080:8080 -v tracevibe-data:/app/data tracevibe:latest

# Run in background
docker run -d --name tracevibe -p 8080:8080 -v tracevibe-data:/app/data --restart unless-stopped tracevibe:latest
```

### Option 2: Build from Source

```bash
# Prerequisites: Go 1.24+
git clone <your-repo-url>
cd tracevibe
go build -o tracevibe

# Run the server
./tracevibe serve --port 8080
```

## How to Use TraceVibe

There are three ways to get started:

### 1. Create a New Project from Scratch
Start with a blank project and manually add components, requirements, and test cases through the UI.

### 2. Import an Existing Project
Import a project that was previously exported from TraceVibe (YAML or JSON format).

### 3. Reverse Engineer Existing Codebase
For existing codebases, use AI to generate RTM documentation:

1. **Generate guidelines**: `tracevibe guidelines`
2. Use the generated `rtm-guidelines.md` and prompt with any LLM
3. Provide the LLM with your repository link and the guidelines
4. Ask the LLM to analyze your code and generate TraceVibe-compatible YAML/JSON
5. Import the generated file using the web UI or CLI: `tracevibe import your-rtm.yaml --project your-project`

## CLI Commands

```bash
# Generate RTM guidelines for LLMs
tracevibe guidelines

# Import RTM data
tracevibe import project-rtm.json --project myproject

# Start web server
tracevibe serve --port 8080

# Custom database location
tracevibe serve --db-path ./custom.db
```

## Web Interface

Access the web UI at `http://localhost:8080` to:
- View project dashboard with statistics
- Browse components and their requirements
- Filter components by tags
- Export projects in multiple formats
- Import/create new projects

## RTM Structure

```
Component (backend-api, frontend-app, database)
  └── Scope (high-level functionality)
      └── User Stories (user journeys)
          └── Tech Specs (detailed implementation requirements)
```

## Database Storage

TraceVibe stores data in SQLite:
- **Docker**: `/app/data/tracevibe.db` (mounted volume)
- **Source**: `~/.tracevibe/tracevibe.db` (macOS/Linux) or `%USERPROFILE%\.tracevibe\tracevibe.db` (Windows)

## License

MIT