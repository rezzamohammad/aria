package handoff

import (
	"strings"
	"testing"

	"github.com/aria-cli/aria/internal/task"
)

func TestGenerateHandoff(t *testing.T) {
	tk := &task.Task{
		ID:           "T-001",
		Title:        "Implement login",
		Description:  "Create a secure login system",
		Role:         "worker",
		Priority:     10,
		CLITool:      "claude-code",
		Dependencies: []string{"T-000"},
		ContextRefs:  []string{"auth-spec"},
	}

	result := GenerateHandoff(tk)

	// Should have YAML frontmatter
	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected YAML frontmatter start")
	}

	// Should contain key fields
	checks := []string{
		`handoff_id: "T-001"`,
		`from: "orchestrator"`,
		`to: "worker"`,
		`task: "Implement login"`,
		`priority: "P0-critical"`,
		`cli_tool: "claude-code"`,
		"status: \"delegating\"",
		"# Task Handoff: T-001",
		"## Context",
		"Create a secure login system",
		"## Goal",
		"## Dependencies",
		"- T-000 (must be done)",
		"## Verification Requirements",
		"Minimum score: 80/100",
		"## Recovery Info",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("handoff missing %q", check)
		}
	}
}

func TestGenerateHandoffNoDependencies(t *testing.T) {
	tk := &task.Task{
		ID:       "T-001",
		Title:    "Simple task",
		Role:     "worker",
		Priority: 50,
	}

	result := GenerateHandoff(tk)
	if !strings.Contains(result, "- None") {
		t.Error("expected '- None' for tasks with no dependencies")
	}
}

func TestGenerateHandoffWithRecovery(t *testing.T) {
	tk := &task.Task{
		ID:            "T-001",
		Title:         "Retry task",
		Role:          "worker",
		Priority:      50,
		RecoveryCount: 2,
		MaxRetries:    3,
		FailureReason: "compile error in main.go",
	}

	result := GenerateHandoff(tk)
	if !strings.Contains(result, "Recovery count: 2 / 3") {
		t.Error("expected recovery count info")
	}
	if !strings.Contains(result, "Previous failure: compile error in main.go") {
		t.Error("expected previous failure reason")
	}
}

func TestGenerateReport(t *testing.T) {
	tk := &task.Task{
		ID:      "T-001",
		Title:   "Test task",
		Role:    "worker",
		Status:  task.StatusDone,
		CLITool: "claude-code",
	}

	result := GenerateReport(tk, 95, "Well implemented")

	checks := []string{
		`task_id: "T-001"`,
		`score: 95`,
		`role: "worker"`,
		"# Report: T-001",
		"## Feedback",
		"Well implemented",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("report missing %q", check)
		}
	}
}
