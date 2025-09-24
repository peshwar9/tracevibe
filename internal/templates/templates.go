package templates

import "embed"

//go:embed assets/*.md
var FS embed.FS

// GetGuidelinesTemplate returns the RTM guidelines template content
func GetGuidelinesTemplate() ([]byte, error) {
	return FS.ReadFile("assets/rtm-guidelines-template.md")
}

// GetLLMPromptTemplate returns the LLM prompt template content
func GetLLMPromptTemplate() ([]byte, error) {
	return FS.ReadFile("assets/llm-prompt-template.md")
}