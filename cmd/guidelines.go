package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/peshwar9/tracevibe/internal/templates"
	"github.com/spf13/cobra"
)

var guidelinesCmd = &cobra.Command{
	Use:   "guidelines",
	Short: "Generate RTM guidelines for LLM analysis",
	Long: `Generate RTM creation guidelines that can be provided to LLMs (Claude, GPT-4, etc.)
to automatically analyze your codebase and create RTM YAML/JSON data.

The guidelines include:
- Component identification rules (real vs logical components)
- Requirement granularity principles
- Hierarchical structure: Scope -> User Stories -> Tech Specs
- Test mapping strategies
- JSON/YAML format specifications`,
	Run: func(cmd *cobra.Command, args []string) {
		outputFile, _ := cmd.Flags().GetString("output")
		promptFile, _ := cmd.Flags().GetString("prompt-file")
		includePrompt, _ := cmd.Flags().GetBool("with-prompt")

		if err := generateGuidelines(outputFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating guidelines: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ RTM guidelines generated: %s\n", outputFile)

		// Generate LLM prompt if requested
		if includePrompt || promptFile != "" {
			prompt := generateLLMPrompt()
			if promptFile != "" {
				if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing prompt file: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("✅ LLM prompt generated: %s\n", promptFile)
			} else {
				fmt.Printf("\n📋 Copy this prompt for your LLM:\n%s\n", strings.Repeat("=", 60))
				fmt.Println(prompt)
				fmt.Println(strings.Repeat("=", 60))
			}
		}

		fmt.Printf("\n🚀 Next steps:\n")
		fmt.Printf("1. Copy the guidelines file: %s\n", outputFile)
		fmt.Printf("2. Use the prompt above with your LLM (Claude, GPT-4, etc.)\n")
		fmt.Printf("3. Provide your codebase to the LLM\n")
		fmt.Printf("4. Import generated RTM: tracevibe import <rtm-file> --project <name>\n")
	},
}

func init() {
	rootCmd.AddCommand(guidelinesCmd)
	guidelinesCmd.Flags().StringP("output", "o", "rtm-guidelines.md", "Output file for guidelines")
	guidelinesCmd.Flags().BoolP("with-prompt", "p", false, "Display LLM prompt to console")
	guidelinesCmd.Flags().String("prompt-file", "", "Save LLM prompt to specified file")
}

func generateGuidelines(outputFile string) error {
	// Read the template from embedded filesystem
	templateContent, err := templates.GetGuidelinesTemplate()
	if err != nil {
		return fmt.Errorf("failed to read guidelines template: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(templateContent)
	if err != nil {
		return fmt.Errorf("failed to write guidelines: %w", err)
	}

	return nil
}

func generateLLMPrompt() string {
	// Read the LLM prompt template from embedded filesystem
	promptContent, err := templates.GetLLMPromptTemplate()
	if err != nil {
		// Fallback to a basic prompt if template can't be read
		return "# RTM Generation Request\n\nPlease analyze my codebase and generate a Requirements Traceability Matrix (RTM) in JSON format following the provided guidelines."
	}
	return string(promptContent)
}