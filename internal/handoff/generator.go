package handoff

import (
	"fmt"
	"strings"
	"time"

	"github.com/aria-cli/aria/internal/task"
)

// GenerateHandoff creates a consistent YAML handoff envelope for task delegation.
func GenerateHandoff(t *task.Task) string {
	contextRefs := t.ContextRefsJSON()
	now := time.Now().UTC().Format(time.RFC3339)

	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("handoff_id: %q\n", t.ID))
	sb.WriteString(fmt.Sprintf("from: \"orchestrator\"\n"))
	sb.WriteString(fmt.Sprintf("to: %q\n", t.Role))
	sb.WriteString(fmt.Sprintf("task: %q\n", t.Title))
	sb.WriteString(fmt.Sprintf("priority: %q\n", t.PriorityLabel()))
	sb.WriteString(fmt.Sprintf("cli_tool: %q\n", t.CLITool))
	if t.WorktreePath != "" {
		sb.WriteString(fmt.Sprintf("worktree: %q\n", t.WorktreePath))
	}
	if t.SessionID != "" {
		sb.WriteString(fmt.Sprintf("session_id: %q\n", t.SessionID))
	}
	sb.WriteString(fmt.Sprintf("created_at: %q\n", now))
	sb.WriteString("status: \"delegating\"\n")
	sb.WriteString(fmt.Sprintf("context_refs: %s\n", contextRefs))
	sb.WriteString("---\n\n")

	// Markdown body
	sb.WriteString(fmt.Sprintf("# Task Handoff: %s — %s\n\n", t.ID, t.Title))

	if t.Description != "" {
		sb.WriteString("## Context\n")
		sb.WriteString(t.Description)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Goal\n")
	sb.WriteString(fmt.Sprintf("Complete task %s: %s\n\n", t.ID, t.Title))

	sb.WriteString("## Dependencies\n")
	if len(t.Dependencies) > 0 {
		for _, dep := range t.Dependencies {
			sb.WriteString(fmt.Sprintf("- %s (must be done)\n", dep))
		}
	} else {
		sb.WriteString("- None\n")
	}
	sb.WriteString("\n")

	sb.WriteString("## Verification Requirements\n")
	sb.WriteString("- Minimum score: 80/100\n")
	sb.WriteString("- Scoring weights: technical 50%, content 35%, aesthetic 15%\n")
	sb.WriteString("- Max iterations: 10\n\n")

	sb.WriteString("## Recovery Info\n")
	sb.WriteString(fmt.Sprintf("- Recovery count: %d / %d\n", t.RecoveryCount, t.MaxRetries))
	if t.FailureReason != "" {
		sb.WriteString(fmt.Sprintf("- Previous failure: %s\n", t.FailureReason))
	}
	sb.WriteString("\n")

	return sb.String()
}

// GenerateReport creates a structured completion report template.
func GenerateReport(t *task.Task, score int, feedback string) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("task_id: %q\n", t.ID))
	sb.WriteString(fmt.Sprintf("status: %q\n", t.Status))
	sb.WriteString(fmt.Sprintf("score: %d\n", score))
	sb.WriteString(fmt.Sprintf("role: %q\n", t.Role))
	sb.WriteString(fmt.Sprintf("cli_tool: %q\n", t.CLITool))
	sb.WriteString(fmt.Sprintf("completed_at: %q\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# Report: %s — %s\n\n", t.ID, t.Title))

	if feedback != "" {
		sb.WriteString("## Feedback\n")
		sb.WriteString(feedback)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
