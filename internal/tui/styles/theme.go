package styles

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary    = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary  = lipgloss.Color("#06B6D4") // Cyan
	ColorSuccess    = lipgloss.Color("#10B981") // Green
	ColorWarning    = lipgloss.Color("#F59E0B") // Amber
	ColorDanger     = lipgloss.Color("#EF4444") // Red
	ColorMuted      = lipgloss.Color("#6B7280") // Gray
	ColorText       = lipgloss.Color("#E5E7EB") // Light gray
	ColorBackground = lipgloss.Color("#1F2937") // Dark
	ColorSurface    = lipgloss.Color("#374151") // Slightly lighter dark
	ColorBorder     = lipgloss.Color("#4B5563") // Border gray
)

// Base styles
var (
	BaseStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1).
			PaddingRight(1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(ColorBorder)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurface).
			PaddingLeft(1).
			PaddingRight(1)
)

// Panel styles
var (
	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1)
)

// Task status styles
var (
	StatusPending = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StatusRunning = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	StatusVerifying = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	StatusDone = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StatusFailed = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	StatusBlocked = lipgloss.NewStyle().
			Foreground(ColorWarning)
)

// Agent status styles
var (
	AgentBusy    = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	AgentIdle    = lipgloss.NewStyle().Foreground(ColorMuted)
	AgentCrashed = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
)

// Context bar styles
var (
	ContextOK       = lipgloss.NewStyle().Foreground(ColorSuccess)
	ContextWarning  = lipgloss.NewStyle().Foreground(ColorWarning)
	ContextCritical = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
)

// Log styles
var (
	LogTimestamp = lipgloss.NewStyle().Foreground(ColorMuted)
	LogInfo      = lipgloss.NewStyle().Foreground(ColorText)
	LogWarn      = lipgloss.NewStyle().Foreground(ColorWarning)
	LogError     = lipgloss.NewStyle().Foreground(ColorDanger)
)

// Priority styles
var (
	PriorityP0 = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
	PriorityP1 = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	PriorityP2 = lipgloss.NewStyle().Foreground(ColorSecondary)
	PriorityP3 = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Help style
var HelpStyle = lipgloss.NewStyle().
	Foreground(ColorMuted).
	PaddingLeft(1)

// FormatStatusStyle returns the appropriate style for a task status.
func FormatStatusStyle(status string) lipgloss.Style {
	switch status {
	case "pending":
		return StatusPending
	case "claimed", "running":
		return StatusRunning
	case "verifying":
		return StatusVerifying
	case "done":
		return StatusDone
	case "failed":
		return StatusFailed
	case "blocked":
		return StatusBlocked
	default:
		return BaseStyle
	}
}

// FormatContextBar returns a colored context usage bar.
func FormatContextBar(pct float64) string {
	filled := int(pct * 10)
	if filled > 10 {
		filled = 10
	}
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	style := ContextOK
	if pct >= 0.9 {
		style = ContextCritical
	} else if pct >= 0.7 {
		style = ContextWarning
	}

	return style.Render(bar)
}

// FormatPriority returns a styled priority label.
func FormatPriority(priority int) string {
	switch {
	case priority <= 10:
		return PriorityP0.Render("P0")
	case priority <= 30:
		return PriorityP1.Render("P1")
	case priority <= 60:
		return PriorityP2.Render("P2")
	default:
		return PriorityP3.Render("P3")
	}
}
