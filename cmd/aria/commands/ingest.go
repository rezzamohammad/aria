package commands

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aria-cli/aria/internal/db"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var ingestTaskmaster bool

// IngestCmd ingests tasks from documents.
var IngestCmd = &cobra.Command{
	Use:   "ingest [file]",
	Short: "Ingest tasks from a document (PRD, CLAUDE.md, spec, etc.)",
	Long: `Parses a markdown document and extracts structured tasks into the ARIA queue.
Supports PRDs, CLAUDE.md files, specs, and any structured markdown with task-like items.`,
	Args: cobra.ExactArgs(1),
	RunE: runIngest,
}

func init() {
	IngestCmd.Flags().BoolVar(&ingestTaskmaster, "taskmaster", false, "Use Task Master for breakdown")
}

func runIngest(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	tasks := extractTasks(string(content), filePath)

	if len(tasks) == 0 {
		fmt.Println("No tasks extracted from", filePath)
		return nil
	}

	for _, t := range tasks {
		if err := queries.InsertTask(database, t); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to insert task %s: %v\n", t.ID, err)
			continue
		}
		fmt.Printf("✓ [%s] %s (%s)\n", t.ID, t.Title, t.Role)
	}

	fmt.Printf("\nIngested %d tasks from %s\n", len(tasks), filePath)
	return nil
}

// extractTasks parses markdown content to find task-like items.
func extractTasks(content string, sourceFile string) []*queries.Task {
	var tasks []*queries.Task

	scanner := bufio.NewScanner(strings.NewReader(content))
	currentEpic := ""
	taskCounter := 0

	// Patterns for task extraction
	taskPattern := regexp.MustCompile(`^[-*]\s+\*\*?(E\d+-T\d+|T-\d+)\*?\*?:?\s*(.+)`)
	headingPattern := regexp.MustCompile(`^#{1,3}\s+(.+)`)
	checkboxPattern := regexp.MustCompile(`^[-*]\s+\[[ x]\]\s+(.+)`)
	epicPattern := regexp.MustCompile(`(?i)epic\s+(\d+)|###\s+.*epic`)

	for scanner.Scan() {
		line := scanner.Text()

		// Track epic headings
		if matches := epicPattern.FindStringSubmatch(line); len(matches) > 0 {
			if matches[1] != "" {
				currentEpic = "E" + matches[1]
			}
			continue
		}

		// Track section headings for context
		if matches := headingPattern.FindStringSubmatch(line); len(matches) > 1 {
			// Use heading as epic if no explicit epic
			if currentEpic == "" {
				currentEpic = sanitizeID(matches[1])
			}
			continue
		}

		// Extract explicit task IDs (E1-T1 format or T-001 format)
		if matches := taskPattern.FindStringSubmatch(line); len(matches) > 2 {
			taskCounter++
			t := &queries.Task{
				ID:     matches[1],
				Title:  strings.TrimSpace(matches[2]),
				Role:   inferRole(matches[2]),
				Status: "pending",
			}
			if currentEpic != "" {
				t.EpicID = sql.NullString{String: currentEpic, Valid: true}
			}
			tasks = append(tasks, t)
			continue
		}

		// Extract checkbox items as tasks
		if matches := checkboxPattern.FindStringSubmatch(line); len(matches) > 1 {
			taskCounter++
			taskID := fmt.Sprintf("T-%03d", taskCounter)
			t := &queries.Task{
				ID:     taskID,
				Title:  strings.TrimSpace(matches[1]),
				Role:   inferRole(matches[1]),
				Status: "pending",
			}
			if currentEpic != "" {
				t.EpicID = sql.NullString{String: currentEpic, Valid: true}
			}
			tasks = append(tasks, t)
		}
	}

	// If no structured tasks found, try bullet points
	if len(tasks) == 0 {
		tasks = extractBulletTasks(content)
	}

	return tasks
}

// extractBulletTasks extracts tasks from simple bullet-point lists.
func extractBulletTasks(content string) []*queries.Task {
	var tasks []*queries.Task
	counter := 0

	bulletPattern := regexp.MustCompile(`^[-*]\s+(.{10,})`)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := bulletPattern.FindStringSubmatch(line); len(matches) > 1 {
			counter++
			title := strings.TrimSpace(matches[1])
			// Skip if it looks like a description, not a task
			if len(title) > 200 || strings.HasPrefix(title, "Note:") {
				continue
			}
			tasks = append(tasks, &queries.Task{
				ID:     fmt.Sprintf("T-%03d", counter),
				Title:  title,
				Role:   inferRole(title),
				Status: "pending",
			})
		}
	}

	return tasks
}

// inferRole guesses the appropriate agent role from the task description.
func inferRole(description string) string {
	lower := strings.ToLower(description)
	switch {
	case strings.Contains(lower, "test") || strings.Contains(lower, "qa") || strings.Contains(lower, "quality"):
		return "qa"
	case strings.Contains(lower, "document") || strings.Contains(lower, "doc") || strings.Contains(lower, "readme"):
		return "doc_system"
	case strings.Contains(lower, "security") || strings.Contains(lower, "auth") || strings.Contains(lower, "vulnerability"):
		return "security"
	case strings.Contains(lower, "architect") || strings.Contains(lower, "design") || strings.Contains(lower, "api contract"):
		return "architect"
	case strings.Contains(lower, "research") || strings.Contains(lower, "evaluate") || strings.Contains(lower, "investigate"):
		return "researcher"
	case strings.Contains(lower, "deploy") || strings.Contains(lower, "release") || strings.Contains(lower, "ci/cd"):
		return "release_ops"
	case strings.Contains(lower, "prd") || strings.Contains(lower, "requirement") || strings.Contains(lower, "acceptance"):
		return "pm"
	case strings.Contains(lower, "orchestrat") || strings.Contains(lower, "decompose") || strings.Contains(lower, "delegate"):
		return "orchestrator"
	case strings.Contains(lower, "verify") || strings.Contains(lower, "review") || strings.Contains(lower, "score"):
		return "verifier"
	default:
		return "worker"
	}
}

// sanitizeID creates a clean ID from a heading.
func sanitizeID(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "-")
	if len(s) > 20 {
		s = s[:20]
	}
	return s
}

// generateTaskID creates a unique task ID.
func generateTaskID() string {
	id := uuid.New().String()
	return "T-" + id[:6]
}
