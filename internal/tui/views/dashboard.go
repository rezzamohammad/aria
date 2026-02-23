package views

import (
	"fmt"
	"strings"

	"github.com/aria-cli/aria/internal/task"
	"github.com/aria-cli/aria/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// DashboardView renders the main kanban board view.
type DashboardView struct {
	Width  int
	Height int
}

// NewDashboardView creates a new dashboard view.
func NewDashboardView(width, height int) *DashboardView {
	return &DashboardView{Width: width, Height: height}
}

// Render produces the kanban board layout from the given tasks.
func (d *DashboardView) Render(tasks []*task.Task, projectName string) string {
	// Group tasks by status columns
	columns := map[string][]*task.Task{
		"pending":   {},
		"running":   {},
		"verifying": {},
		"done":      {},
	}

	for _, t := range tasks {
		switch t.Status {
		case task.StatusPending, task.StatusBlocked:
			columns["pending"] = append(columns["pending"], t)
		case task.StatusClaimed, task.StatusRunning:
			columns["running"] = append(columns["running"], t)
		case task.StatusVerifying:
			columns["verifying"] = append(columns["verifying"], t)
		case task.StatusDone:
			columns["done"] = append(columns["done"], t)
		}
	}

	// Count active vs done
	active := len(columns["running"]) + len(columns["verifying"])
	done := len(columns["done"])

	// Header
	header := styles.TitleStyle.Render(fmt.Sprintf(
		"ARIA v0.1  │  Project: %s  │  Tasks: %d active / %d done",
		projectName, active, done))

	// Column width
	colWidth := 18
	if d.Width > 80 {
		colWidth = (d.Width - 8) / 4
	}

	// Render each column
	pendingCol := renderColumn("PENDING", columns["pending"], colWidth)
	runningCol := renderColumn("RUNNING", columns["running"], colWidth)
	verifyCol := renderColumn("VERIFYING", columns["verifying"], colWidth)
	doneCol := renderColumn("DONE", columns["done"], colWidth)

	board := lipgloss.JoinHorizontal(lipgloss.Top, pendingCol, runningCol, verifyCol, doneCol)

	return lipgloss.JoinVertical(lipgloss.Left, header, board)
}

// renderColumn renders a single kanban column.
func renderColumn(title string, tasks []*task.Task, width int) string {
	header := styles.PanelTitleStyle.Render(title)

	var items []string
	items = append(items, header)
	items = append(items, strings.Repeat("─", width))

	maxItems := 8
	for i, t := range tasks {
		if i >= maxItems {
			items = append(items, styles.BaseStyle.Render(
				fmt.Sprintf("  ... +%d more", len(tasks)-maxItems)))
			break
		}
		items = append(items, renderTaskCard(t, width))
	}

	if len(tasks) == 0 {
		items = append(items, styles.StatusPending.Render("  (empty)"))
	}

	content := strings.Join(items, "\n")
	return styles.PanelStyle.Width(width).Render(content)
}

// renderTaskCard renders a single task in the kanban column.
func renderTaskCard(t *task.Task, width int) string {
	idLine := fmt.Sprintf("[%s] %s", t.ID, styles.FormatPriority(t.Priority))
	titleLine := truncate(t.Title, width-4)
	roleLine := styles.BaseStyle.Render(t.Role)

	statusStyle := styles.FormatStatusStyle(string(t.Status))

	var extraLine string
	if t.CLITool != "" {
		extraLine = styles.BaseStyle.Render(t.CLITool)
	}
	if t.Score > 0 {
		extraLine += fmt.Sprintf(" %d✓", t.Score)
	}

	lines := []string{
		statusStyle.Render(idLine),
		titleLine,
		roleLine,
	}
	if extraLine != "" {
		lines = append(lines, extraLine)
	}

	return strings.Join(lines, "\n") + "\n"
}

// truncate shortens a string to fit within maxLen.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
