package handoff

import (
	"testing"
)

func TestParseReport(t *testing.T) {
	content := `---
task_id: "T-001"
status: "done"
score: "95"
role: "worker"
cli_tool: "claude-code"
completed_at: "2025-01-01T00:00:00Z"
---

# Report: T-001 — Implement login

## Feedback
Well implemented with proper error handling.
`

	report, err := ParseReport(content)
	if err != nil {
		t.Fatalf("ParseReport error: %v", err)
	}

	if report.TaskID != "T-001" {
		t.Errorf("expected task_id T-001, got %q", report.TaskID)
	}
	if report.Status != "done" {
		t.Errorf("expected status done, got %q", report.Status)
	}
	if report.Score != "95" {
		t.Errorf("expected score 95, got %q", report.Score)
	}
	if report.Role != "worker" {
		t.Errorf("expected role worker, got %q", report.Role)
	}
	if report.CLITool != "claude-code" {
		t.Errorf("expected cli_tool claude-code, got %q", report.CLITool)
	}
	if report.Body == "" {
		t.Error("expected non-empty body")
	}
}

func TestParseReportMissingTaskID(t *testing.T) {
	content := `---
status: "done"
score: "95"
---

Some body text.
`

	_, err := ParseReport(content)
	if err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestParseReportNoFrontmatter(t *testing.T) {
	content := `Just some text without frontmatter`

	_, err := ParseReport(content)
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseReportFieldsMap(t *testing.T) {
	content := `---
task_id: "T-001"
custom_field: "custom_value"
another: "test"
---

Body
`

	report, err := ParseReport(content)
	if err != nil {
		t.Fatalf("ParseReport error: %v", err)
	}

	if report.Fields["custom_field"] != "custom_value" {
		t.Errorf("expected custom_field = custom_value, got %q", report.Fields["custom_field"])
	}
	if report.Fields["another"] != "test" {
		t.Errorf("expected another = test, got %q", report.Fields["another"])
	}
}
