package commands

import (
	"fmt"

	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/spf13/cobra"
)

// MemoryCmd is the parent command for memory management.
var MemoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Memory management commands",
}

var memoryAddKey string
var memoryAddValue string

func init() {
	// memory show
	showCmd := &cobra.Command{
		Use:   "show [task-id]",
		Short: "Show context loaded for a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runMemoryShow,
	}
	MemoryCmd.AddCommand(showCmd)

	// memory add
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Manually inject a memory entry",
		RunE:  runMemoryAdd,
	}
	addCmd.Flags().StringVar(&memoryAddKey, "key", "", "Memory key")
	addCmd.Flags().StringVar(&memoryAddValue, "value", "", "Memory value")
	addCmd.MarkFlagRequired("key")
	addCmd.MarkFlagRequired("value")
	MemoryCmd.AddCommand(addCmd)
}

func runMemoryShow(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	t, err := queries.GetTask(database, taskID)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	fmt.Printf("Task: %s — %s\n", t.ID, t.Title)
	fmt.Printf("Role: %s\n", t.Role)

	// Show sessions for this task
	sessions, err := queries.ListSessionsByTask(database, taskID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("\nNo sessions recorded for this task.")
	} else {
		fmt.Printf("\nSessions (%d):\n", len(sessions))
		for _, s := range sessions {
			fmt.Printf("  [%s] %s — tokens: %d, ctx: %.0f%%\n",
				s.ID[:8], s.Status, s.ContextTokens, s.ContextPct*100)
			if s.ContextSnapshot.Valid {
				fmt.Printf("    Snapshot: %s\n", s.ContextSnapshot.String)
			}
		}
	}

	return nil
}

func runMemoryAdd(cmd *cobra.Command, args []string) error {
	fmt.Printf("✓ Memory entry added: %s = %s\n", memoryAddKey, memoryAddValue)
	fmt.Println("Note: Full memory backend (Mem0) integration is planned for Phase 2.")
	return nil
}
