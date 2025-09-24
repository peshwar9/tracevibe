package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/peshwar9/tracevibe/internal/database"
	"github.com/peshwar9/tracevibe/internal/importer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import [RTM_FILE]",
	Short: "Import RTM data from YAML/JSON file",
	Long: `Import Requirements Traceability Matrix data from YAML or JSON file into the local SQLite database.

The RTM file should follow the hierarchical structure:
- System Components (deployable units)
- Requirements: Scope -> User Stories -> Tech Specs
- Implementation mappings to source files
- Test coverage mapping

Import Modes:
- Default (update): Add new requirements and update existing ones by requirement key
- --overwrite: Delete all existing project data and reimport everything fresh

Example:
  tracevibe import my-project-rtm.yaml --project my-project
  tracevibe import rtm-data.json --project statsly --overwrite
  tracevibe import rtm-data.json --project statsly --db-path /custom/path/tracevibe.db`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rtmFile := args[0]
		projectKey, _ := cmd.Flags().GetString("project")
		dbPath, _ := cmd.Flags().GetString("db-path")
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		if projectKey == "" {
			fmt.Fprintf(os.Stderr, "Error: --project flag is required\n")
			os.Exit(1)
		}

		// Validate RTM file exists
		if _, err := os.Stat(rtmFile); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: RTM file does not exist: %s\n", rtmFile)
			os.Exit(1)
		}

		if err := runImport(rtmFile, projectKey, dbPath, overwrite); err != nil {
			fmt.Fprintf(os.Stderr, "Error importing RTM data: %v\n", err)
			os.Exit(1)
		}

		mode := "updated"
		if overwrite {
			mode = "overwritten and reimported"
		}
		fmt.Printf("Successfully %s RTM data for project '%s'\n", mode, projectKey)
		fmt.Printf("Database: %s\n", dbPath)
		fmt.Printf("Use 'tracevibe serve' to view the data in the admin UI\n")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringP("project", "p", "", "Project key/identifier (required)")
	importCmd.Flags().StringP("db-path", "d", getDefaultDBPath(), "SQLite database path")
	importCmd.Flags().Bool("overwrite", false, "Delete existing project data before import (default: update mode)")

	importCmd.MarkFlagRequired("project")
}

func runImport(rtmFile, projectKey, dbPath string, overwrite bool) error {
	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Initialize schema if needed
	if err := db.InitSchema(); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	// Create importer and run import
	imp := importer.New(db)
	if err := imp.ImportRTMFile(rtmFile, projectKey, overwrite); err != nil {
		return fmt.Errorf("failed to import RTM data: %w", err)
	}

	return nil
}

func getDefaultDBPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./tracevibe.db"
	}
	return filepath.Join(homeDir, ".tracevibe", "tracevibe.db")
}