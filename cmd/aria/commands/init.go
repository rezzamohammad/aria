package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aria-cli/aria/internal/config"
	"github.com/aria-cli/aria/internal/db"
	"github.com/spf13/cobra"
)

// InitCmd bootstraps an ARIA project in the current directory.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize ARIA project (creates aria.toml + database)",
	Long:  "Creates an aria.toml configuration file and initializes the SQLite database with all required tables.",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(cwd, "aria.toml")

	// Check if already initialized
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("aria.toml already exists in %s — project already initialized", cwd)
	}

	// Write default config
	if err := config.WriteDefault(configPath); err != nil {
		return fmt.Errorf("failed to write aria.toml: %w", err)
	}
	fmt.Println("✓ Created aria.toml")

	// Create .aria directory
	ariaDir := filepath.Join(cwd, ".aria")
	if err := os.MkdirAll(ariaDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .aria directory: %w", err)
	}

	// Create worktrees directory
	wtDir := filepath.Join(cwd, ".aria", "worktrees")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}
	fmt.Println("✓ Created .aria/ directory")

	// Initialize database
	dbPath := filepath.Join(cwd, ".aria", "aria.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	database.Close()
	fmt.Println("✓ Initialized SQLite database at .aria/aria.db")

	// Create .gitignore for .aria
	gitignorePath := filepath.Join(ariaDir, ".gitignore")
	gitignoreContent := "aria.db\naria.db-wal\naria.db-shm\nworktrees/\n*.log\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}
	fmt.Println("✓ Created .aria/.gitignore")

	fmt.Println("\nARIA project initialized. Run 'aria run' to launch the TUI dashboard.")
	return nil
}
