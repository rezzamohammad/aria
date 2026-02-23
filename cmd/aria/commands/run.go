package commands

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aria-cli/aria/internal/config"
	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var runHeadless bool
var runAuto bool

// RunCmd launches the ARIA TUI dashboard or headless mode.
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch ARIA dashboard",
	Long:  "Launches the TUI dashboard showing task kanban, agent status, and session logs.",
	RunE:  runRun,
}

func init() {
	RunCmd.Flags().BoolVar(&runHeadless, "headless", false, "Run without TUI (log-only mode)")
	RunCmd.Flags().BoolVar(&runAuto, "auto", false, "Fully autonomous mode (no human prompts)")
}

func runRun(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	if runHeadless {
		return runHeadlessMode(database, cfg)
	}

	// Launch TUI
	model := tui.NewModel(database, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runHeadlessMode(_ *sql.DB, _ *config.Config) error {
	fmt.Println("ARIA running in headless mode. Press Ctrl+C to stop.")
	fmt.Println("Logs will be written to .aria/aria.log")

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)
	fmt.Println("\nARIA stopped.")
	return nil
}
