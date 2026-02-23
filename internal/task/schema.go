package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusPending    Status = "pending"
	StatusClaimed    Status = "claimed"
	StatusRunning    Status = "running"
	StatusVerifying  Status = "verifying"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
	StatusBlocked    Status = "blocked"
	StatusCancelled  Status = "cancelled"
)

// ValidStatuses contains all valid task statuses.
var ValidStatuses = []Status{
	StatusPending, StatusClaimed, StatusRunning, StatusVerifying,
	StatusDone, StatusFailed, StatusBlocked, StatusCancelled,
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if string(v) == s {
			return true
		}
	}
	return false
}

// Task is the in-memory representation used by the queue and graph.
type Task struct {
	ID             string   `json:"id"`
	EpicID         string   `json:"epic_id,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Role           string   `json:"role"`
	Status         Status   `json:"status"`
	Priority       int      `json:"priority"`
	Dependencies   []string `json:"dependencies,omitempty"`
	WorktreePath   string   `json:"worktree_path,omitempty"`
	SessionID      string   `json:"session_id,omitempty"`
	AgentID        string   `json:"agent_id,omitempty"`
	CLITool        string   `json:"cli_tool,omitempty"`
	ContextRefs    []string `json:"context_refs,omitempty"`
	Score          int      `json:"score,omitempty"`
	ScoreBreakdown string   `json:"score_breakdown,omitempty"`
	RecoveryCount  int      `json:"recovery_count"`
	MaxRetries     int      `json:"max_retries"`
	FailureReason  string   `json:"failure_reason,omitempty"`
	HandoffPath    string   `json:"handoff_path,omitempty"`
	ReportPath     string   `json:"report_path,omitempty"`
	CreatedAt      string   `json:"created_at"`
	ClaimedAt      string   `json:"claimed_at,omitempty"`
	StartedAt      string   `json:"started_at,omitempty"`
	CompletedAt    string   `json:"completed_at,omitempty"`
	EstimatedHours float64  `json:"estimated_hours,omitempty"`
	ActualHours    float64  `json:"actual_hours,omitempty"`
}

// PriorityLabel returns a human-readable priority label.
func (t *Task) PriorityLabel() string {
	switch {
	case t.Priority <= 10:
		return "P0-critical"
	case t.Priority <= 30:
		return "P1-high"
	case t.Priority <= 60:
		return "P2-medium"
	default:
		return "P3-low"
	}
}

// ContextRefsJSON returns the context refs as a JSON array string.
func (t *Task) ContextRefsJSON() string {
	if len(t.ContextRefs) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(t.ContextRefs)
	return string(b)
}

// DependenciesStr returns a comma-separated list of dependency IDs.
func (t *Task) DependenciesStr() string {
	if len(t.Dependencies) == 0 {
		return "none"
	}
	return strings.Join(t.Dependencies, ", ")
}

// StatusIcon returns a display icon for the task status.
func (t *Task) StatusIcon() string {
	switch t.Status {
	case StatusPending:
		return "○"
	case StatusClaimed:
		return "◐"
	case StatusRunning:
		return "●"
	case StatusVerifying:
		return "◉"
	case StatusDone:
		return "✓"
	case StatusFailed:
		return "✗"
	case StatusBlocked:
		return "⊘"
	case StatusCancelled:
		return "—"
	default:
		return "?"
	}
}

// FormatOneLiner returns a single-line summary of the task.
func (t *Task) FormatOneLiner() string {
	scoreStr := ""
	if t.Score > 0 {
		scoreStr = fmt.Sprintf(" %d✓", t.Score)
	}
	return fmt.Sprintf("[%s] %s %s  %-12s %-10s %s%s",
		t.ID, t.StatusIcon(), t.PriorityLabel(), t.Role, t.Status, t.Title, scoreStr)
}
