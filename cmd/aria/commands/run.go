package commands

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aria-cli/aria/internal/agent"
	"github.com/aria-cli/aria/internal/config"
	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/logger"
	"github.com/aria-cli/aria/internal/orchestrator"
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

	// Create shared logger
	eventLog := logger.New(1000)

	// Create agent pool and router
	router := agent.NewRouter(cfg.ToolMapping)
	pool := agent.NewPool(database, router, cfg.Agents.MaxPoolSize)

	// Create orchestrator
	orch := orchestrator.New(database, cfg, pool, router, eventLog, runAuto)

	if runHeadless {
		return runHeadlessMode(database, cfg, orch, eventLog)
	}

	// Start orchestrator in background
	orch.Start()
	defer orch.Stop()

	// Launch TUI with logger integration
	model := tui.NewModel(database, cfg, eventLog)
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Wire logger events to TUI
	model.SetProgram(p)

	eventLog.Info("system", "ARIA TUI started (auto=%v)", runAuto)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runHeadlessMode(_ *sql.DB, _ *config.Config, orch *orchestrator.Orchestrator, eventLog *logger.Logger) error {
	fmt.Println("ARIA running in headless mode. Press Ctrl+C to stop.")

	// Subscribe to log events and print them to stdout
	eventLog.Subscribe(func(entry logger.Entry) {
		fmt.Printf("[%s] [%s] %s: %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Level,
			entry.Source,
			entry.Message)
	})

	// Start orchestrator
	orch.Start()
	eventLog.Info("system", "ARIA headless mode started")

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)
	orch.Stop()
	fmt.Println("\nARIA stopped.")
	return nil
}
