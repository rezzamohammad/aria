package views

import (
	"fmt"
	"strings"

	"github.com/aria-cli/aria/internal/task"
	"github.com/aria-cli/aria/internal/tui/styles"
)

// TaskDetailView renders a detailed view of a single task.
type TaskDetailView struct {
	Width  int
	Height int
}

// NewTaskDetailView creates a new task detail view.
func NewTaskDetailView(width, height int) *TaskDetailView {
	return &TaskDetailView{Width: width, Height: height}
}

// Render produces the detailed task panel.
func (v *TaskDetailView) Render(t *task.Task) string {
	if t == nil {
		return styles.PanelStyle.Width(v.Width).Render("No task selected")
	}

	var lines []string

	// Title section
	header := fmt.Sprintf("Task: %s — %s", t.ID, t.Title)
	lines = append(lines, styles.HeaderStyle.Render(header))
	lines = append(lines, "")

	// Status & metadata
	statusStyle := styles.FormatStatusStyle(string(t.Status))
	lines = append(lines, fmt.Sprintf("  Status:     %s", statusStyle.Render(string(t.Status))))
	lines = append(lines, fmt.Sprintf("  Priority:   %s", styles.FormatPriority(t.Priority)))
	lines = append(lines, fmt.Sprintf("  Role:       %s", t.Role))

	if t.CLITool != "" {
		lines = append(lines, fmt.Sprintf("  CLI Tool:   %s", t.CLITool))
	}
	if t.AgentID != "" {
		lines = append(lines, fmt.Sprintf("  Agent:      %s", t.AgentID[:8]))
	}
	if t.EpicID != "" {
		lines = append(lines, fmt.Sprintf("  Epic:       %s", t.EpicID))
	}

	lines = append(lines, "")

	// Dependencies
	lines = append(lines, styles.PanelTitleStyle.Render("Dependencies"))
	if len(t.Dependencies) > 0 {
		for _, dep := range t.Dependencies {
			lines = append(lines, fmt.Sprintf("  → %s", dep))
		}
	} else {
		lines = append(lines, "  None")
	}
	lines = append(lines, "")

	// Description
	if t.Description != "" {
		lines = append(lines, styles.PanelTitleStyle.Render("Description"))
		for _, line := range strings.Split(t.Description, "\n") {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	// Score
	if t.Score > 0 {
		lines = append(lines, styles.PanelTitleStyle.Render("Verification"))
		lines = append(lines, fmt.Sprintf("  Score: %d/100", t.Score))
		if t.ScoreBreakdown != "" {
			lines = append(lines, fmt.Sprintf("  Breakdown: %s", t.ScoreBreakdown))
		}
		lines = append(lines, "")
	}

	// Recovery
	if t.RecoveryCount > 0 {
		lines = append(lines, styles.PanelTitleStyle.Render("Recovery"))
		lines = append(lines, fmt.Sprintf("  Attempts: %d / %d", t.RecoveryCount, t.MaxRetries))
		if t.FailureReason != "" {
			lines = append(lines, fmt.Sprintf("  Last failure: %s", t.FailureReason))
		}
		lines = append(lines, "")
	}

	// Timestamps
	lines = append(lines, styles.PanelTitleStyle.Render("Timeline"))
	lines = append(lines, fmt.Sprintf("  Created:  %s", t.CreatedAt))
	if t.ClaimedAt != "" {
		lines = append(lines, fmt.Sprintf("  Claimed:  %s", t.ClaimedAt))
	}
	if t.StartedAt != "" {
		lines = append(lines, fmt.Sprintf("  Started:  %s", t.StartedAt))
	}
	if t.CompletedAt != "" {
		lines = append(lines, fmt.Sprintf("  Completed: %s", t.CompletedAt))
	}

	// Worktree
	if t.WorktreePath != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("  Worktree: %s", t.WorktreePath))
	}

	content := strings.Join(lines, "\n")
	return styles.PanelStyle.Width(v.Width).Render(content)
}
