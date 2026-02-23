package views

import (
	"fmt"
	"strings"

	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/aria-cli/aria/internal/tui/styles"
)

// AgentPanelView renders the live agent status panel.
type AgentPanelView struct {
	Width int
}

// NewAgentPanelView creates a new agent panel view.
func NewAgentPanelView(width int) *AgentPanelView {
	return &AgentPanelView{Width: width}
}

// Render produces the agent status display.
func (v *AgentPanelView) Render(agents []*queries.Agent, sessions map[string]*queries.Session) string {
	// Count by status
	counts := map[string]int{}
	for _, a := range agents {
		counts[a.Status]++
	}

	header := fmt.Sprintf("AGENTS  [%d active / %d idle / %d crashed]",
		counts["busy"], counts["idle"], counts["crashed"])

	var lines []string
	lines = append(lines, styles.PanelTitleStyle.Render(header))

	for _, a := range agents {
		line := renderAgentLine(a, sessions)
		lines = append(lines, line)
	}

	if len(agents) == 0 {
		lines = append(lines, styles.AgentIdle.Render("  No agents spawned"))
	}

	content := strings.Join(lines, "\n")
	return styles.PanelStyle.Width(v.Width).Render(content)
}

// renderAgentLine renders a single agent status line.
func renderAgentLine(a *queries.Agent, sessions map[string]*queries.Session) string {
	var icon string
	var statusStyle = styles.AgentIdle

	switch a.Status {
	case "busy":
		icon = "●"
		statusStyle = styles.AgentBusy
	case "idle":
		icon = "○"
		statusStyle = styles.AgentIdle
	case "crashed", "offline":
		icon = "✗"
		statusStyle = styles.AgentCrashed
	default:
		icon = "?"
	}

	taskStr := "—"
	if a.CurrentTaskID.Valid && a.CurrentTaskID.String != "" {
		taskStr = a.CurrentTaskID.String
	}

	ctxStr := ""
	if a.CurrentSession.Valid {
		if sess, ok := sessions[a.CurrentSession.String]; ok {
			ctxStr = fmt.Sprintf("ctx:%.0f%% %s", sess.ContextPct*100, styles.FormatContextBar(sess.ContextPct))
		}
	}

	rolePadded := fmt.Sprintf("%-14s", a.Role)
	statusPadded := fmt.Sprintf("%-8s", a.Status)
	taskPadded := fmt.Sprintf("%-8s", taskStr)
	toolPadded := fmt.Sprintf("%-14s", a.CLITool)

	return statusStyle.Render(fmt.Sprintf("  %s %s %s %s %s %s",
		icon, rolePadded, statusPadded, taskPadded, toolPadded, ctxStr))
}
