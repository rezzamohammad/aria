package commands

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/spf13/cobra"
)

// TaskCmd is the parent command for task management.
var TaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
}

var taskListStatus string
var taskAddTitle string
var taskAddRole string
var taskAddPriority int
var taskAddEpic string
var taskAddDeps string
var taskAddDesc string

func init() {
	// task list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		RunE:  runTaskList,
	}
	listCmd.Flags().StringVar(&taskListStatus, "status", "", "Filter by status")
	TaskCmd.AddCommand(listCmd)

	// task add
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new task",
		RunE:  runTaskAdd,
	}
	addCmd.Flags().StringVar(&taskAddTitle, "title", "", "Task title (required)")
	addCmd.Flags().StringVar(&taskAddRole, "role", "worker", "Agent role")
	addCmd.Flags().IntVar(&taskAddPriority, "priority", 50, "Priority (0=critical, 100=low)")
	addCmd.Flags().StringVar(&taskAddEpic, "epic", "", "Epic ID")
	addCmd.Flags().StringVar(&taskAddDeps, "deps", "", "Comma-separated dependency task IDs")
	addCmd.Flags().StringVar(&taskAddDesc, "desc", "", "Task description")
	addCmd.MarkFlagRequired("title")
	TaskCmd.AddCommand(addCmd)

	// task show
	showCmd := &cobra.Command{
		Use:   "show [task-id]",
		Short: "Show task details",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskShow,
	}
	TaskCmd.AddCommand(showCmd)

	// task retry
	retryCmd := &cobra.Command{
		Use:   "retry [task-id]",
		Short: "Retry a failed task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskRetry,
	}
	TaskCmd.AddCommand(retryCmd)

	// task cancel
	cancelCmd := &cobra.Command{
		Use:   "cancel [task-id]",
		Short: "Cancel a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskCancel,
	}
	TaskCmd.AddCommand(cancelCmd)
}

func runTaskList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	tasks, err := queries.ListTasks(database, taskListStatus)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID\tSTATUS\tPRIORITY\tROLE\tSCORE\tTITLE\n")
	fmt.Fprintf(w, "──\t──────\t────────\t────\t─────\t─────\n")

	for _, t := range tasks {
		scoreStr := "—"
		if t.Score.Valid {
			scoreStr = fmt.Sprintf("%d", t.Score.Int64)
		}

		priorityStr := formatPriority(t.Priority)

		title := t.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			t.ID, t.Status, priorityStr, t.Role, scoreStr, title)
	}

	w.Flush()

	// Summary
	counts, _ := queries.CountTasksByStatus(database)
	fmt.Printf("\nTotal: %d", len(tasks))
	for status, count := range counts {
		fmt.Printf(" | %s: %d", status, count)
	}
	fmt.Println()

	return nil
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	taskID := generateTaskID()

	t := &queries.Task{
		ID:       taskID,
		Title:    taskAddTitle,
		Role:     taskAddRole,
		Status:   "pending",
		Priority: taskAddPriority,
	}

	if taskAddEpic != "" {
		t.EpicID = sql.NullString{String: taskAddEpic, Valid: true}
	}
	if taskAddDesc != "" {
		t.Description = sql.NullString{String: taskAddDesc, Valid: true}
	}

	if err := queries.InsertTask(database, t); err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	fmt.Printf("✓ Created task [%s] %s (role: %s, priority: %s)\n",
		taskID, taskAddTitle, taskAddRole, formatPriority(taskAddPriority))

	return nil
}

func runTaskShow(cmd *cobra.Command, args []string) error {
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

	qt, err := queries.GetTask(database, taskID)
	if err != nil {
		return fmt.Errorf("task %s not found: %w", taskID, err)
	}

	fmt.Printf("Task: %s\n", qt.ID)
	fmt.Printf("Title: %s\n", qt.Title)
	fmt.Printf("Status: %s\n", qt.Status)
	fmt.Printf("Role: %s\n", qt.Role)
	fmt.Printf("Priority: %s (%d)\n", formatPriority(qt.Priority), qt.Priority)

	if qt.EpicID.Valid {
		fmt.Printf("Epic: %s\n", qt.EpicID.String)
	}
	if qt.Description.Valid {
		fmt.Printf("Description: %s\n", qt.Description.String)
	}
	if qt.AgentID.Valid {
		fmt.Printf("Agent: %s\n", qt.AgentID.String)
	}
	if qt.CLITool.Valid {
		fmt.Printf("CLI Tool: %s\n", qt.CLITool.String)
	}
	if qt.Score.Valid {
		fmt.Printf("Score: %d/100\n", qt.Score.Int64)
	}
	fmt.Printf("Recovery: %d / %d\n", qt.RecoveryCount, qt.MaxRetries)
	fmt.Printf("Created: %s\n", qt.CreatedAt)
	if qt.WorktreePath.Valid {
		fmt.Printf("Worktree: %s\n", qt.WorktreePath.String)
	}

	return nil
}

func runTaskRetry(cmd *cobra.Command, args []string) error {
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

	if err := queries.UpdateTaskStatus(database, taskID, "pending"); err != nil {
		return fmt.Errorf("failed to retry task: %w", err)
	}

	fmt.Printf("✓ Task %s reset to pending\n", taskID)
	return nil
}

func runTaskCancel(cmd *cobra.Command, args []string) error {
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

	if err := queries.UpdateTaskStatus(database, taskID, "cancelled"); err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Printf("✓ Task %s cancelled\n", taskID)
	return nil
}

func formatPriority(p int) string {
	switch {
	case p <= 10:
		return "P0-critical"
	case p <= 30:
		return "P1-high"
	case p <= 60:
		return "P2-medium"
	default:
		return "P3-low"
	}
}

func parsePriority(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(s, "p0") || s == "critical":
		return 0
	case strings.HasPrefix(s, "p1") || s == "high":
		return 20
	case strings.HasPrefix(s, "p2") || s == "medium":
		return 50
	case strings.HasPrefix(s, "p3") || s == "low":
		return 80
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
		return 50
	}
}
