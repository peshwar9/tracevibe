package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tracevibe",
	Short: "TraceVibe - AI-assisted development workflow management",
	Long: `TraceVibe is a standalone CLI tool for Requirements Traceability Matrix (RTM) management.
It helps developers track requirements, implementation files, and test cases across projects.

Features:
- Generate RTM guidelines for LLM-assisted analysis
- Import RTM YAML/JSON data into local SQLite database
- Serve web-based admin UI for visualization
- Track hierarchical requirements: Scope -> User Stories -> Tech Specs
- Execute and monitor test cases
- Git branch integration for change tracking`,
	Version: "1.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
}