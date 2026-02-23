package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/aria-cli/aria/internal/tui/styles"
)

// LogEntry represents a single log line.
type LogEntry struct {
	Timestamp time.Time
	Level     string // info, warn, error
	Source    string // agent ID or system
	Message   string
}

// LogPanelView renders the real-time session log panel.
type LogPanelView struct {
	Width   int
	MaxRows int
}

// NewLogPanelView creates a new log panel view.
func NewLogPanelView(width, maxRows int) *LogPanelView {
	if maxRows <= 0 {
		maxRows = 6
	}
	return &LogPanelView{Width: width, MaxRows: maxRows}
}

// Render produces the log panel display.
func (v *LogPanelView) Render(entries []LogEntry) string {
	header := styles.PanelTitleStyle.Render("LOG")

	var lines []string
	lines = append(lines, header)

	// Show last N entries
	start := 0
	if len(entries) > v.MaxRows {
		start = len(entries) - v.MaxRows
	}

	for _, entry := range entries[start:] {
		line := renderLogEntry(entry)
		lines = append(lines, line)
	}

	if len(entries) == 0 {
		lines = append(lines, styles.LogInfo.Render("  No log entries yet"))
	}

	content := strings.Join(lines, "\n")
	return styles.PanelStyle.Width(v.Width).Render(content)
}

// renderLogEntry formats a single log entry.
func renderLogEntry(entry LogEntry) string {
	ts := styles.LogTimestamp.Render(entry.Timestamp.Format("15:04:05"))

	var msgStyle = styles.LogInfo
	switch entry.Level {
	case "warn":
		msgStyle = styles.LogWarn
	case "error":
		msgStyle = styles.LogError
	}

	source := ""
	if entry.Source != "" {
		source = fmt.Sprintf("[%s] ", entry.Source)
	}

	return fmt.Sprintf("  %s %s%s", ts, source, msgStyle.Render(entry.Message))
}
