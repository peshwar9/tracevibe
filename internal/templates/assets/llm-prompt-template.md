# RTM Generation Request

**IMPORTANT**: Please read and follow ALL the detailed guidelines in the `rtm-guidelines.md` file that was generated alongside this prompt.

## Required Project Information

Before analyzing the codebase, please use these specific project details in your RTM JSON:

- **Project Name**: [PLEASE SPECIFY YOUR PROJECT NAME]
- **Project ID**: [PLEASE SPECIFY A PROJECT KEY/ID - lowercase, no spaces, e.g., "my-app"]
- **Repository URL**: [PLEASE SPECIFY YOUR REPOSITORY URL if available]
- **Project Description**: [PLEASE PROVIDE A BRIEF PROJECT DESCRIPTION]

## Task

Analyze the provided codebase and generate a complete Requirements Traceability Matrix (RTM) in JSON format following the guidelines document.

**Critical Requirements:**
- Use the nested scopes format (Format 2) from the guidelines
- Every scope MUST have a valid `component_id` referencing an existing component
- Map actual files, functions, and tests from the codebase
- Follow the exact JSON structure shown in the guidelines

## Output

Provide the complete RTM JSON that can be imported using:
```
tracevibe import your-output.json --project [your-project-id]
```