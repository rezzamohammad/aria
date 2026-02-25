package task

import (
	"testing"
)

func TestPriorityLabel(t *testing.T) {
	tests := []struct {
		priority int
		expected string
	}{
		{0, "P0-critical"},
		{5, "P0-critical"},
		{10, "P0-critical"},
		{11, "P1-high"},
		{30, "P1-high"},
		{31, "P2-medium"},
		{50, "P2-medium"},
		{60, "P2-medium"},
		{61, "P3-low"},
		{100, "P3-low"},
	}

	for _, tt := range tests {
		task := &Task{Priority: tt.priority}
		got := task.PriorityLabel()
		if got != tt.expected {
			t.Errorf("PriorityLabel(%d) = %q, want %q", tt.priority, got, tt.expected)
		}
	}
}

func TestContextRefsJSON(t *testing.T) {
	tests := []struct {
		refs     []string
		expected string
	}{
		{nil, "[]"},
		{[]string{}, "[]"},
		{[]string{"ref1"}, `["ref1"]`},
		{[]string{"ref1", "ref2"}, `["ref1","ref2"]`},
	}

	for _, tt := range tests {
		task := &Task{ContextRefs: tt.refs}
		got := task.ContextRefsJSON()
		if got != tt.expected {
			t.Errorf("ContextRefsJSON(%v) = %q, want %q", tt.refs, got, tt.expected)
		}
	}
}

func TestDependenciesStr(t *testing.T) {
	tests := []struct {
		deps     []string
		expected string
	}{
		{nil, "none"},
		{[]string{}, "none"},
		{[]string{"T-001"}, "T-001"},
		{[]string{"T-001", "T-002"}, "T-001, T-002"},
	}

	for _, tt := range tests {
		task := &Task{Dependencies: tt.deps}
		got := task.DependenciesStr()
		if got != tt.expected {
			t.Errorf("DependenciesStr(%v) = %q, want %q", tt.deps, got, tt.expected)
		}
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "○"},
		{StatusClaimed, "◐"},
		{StatusRunning, "●"},
		{StatusVerifying, "◉"},
		{StatusDone, "✓"},
		{StatusFailed, "✗"},
		{StatusBlocked, "⊘"},
		{StatusCancelled, "—"},
		{Status("unknown"), "?"},
	}

	for _, tt := range tests {
		task := &Task{Status: tt.status}
		got := task.StatusIcon()
		if got != tt.expected {
			t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, got, tt.expected)
		}
	}
}

func TestIsValidStatus(t *testing.T) {
	valid := []string{"pending", "claimed", "running", "verifying", "done", "failed", "blocked", "cancelled"}
	for _, s := range valid {
		if !IsValidStatus(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}

	invalid := []string{"", "unknown", "started", "completed"}
	for _, s := range invalid {
		if IsValidStatus(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestFormatOneLiner(t *testing.T) {
	task := &Task{
		ID:       "T-001",
		Title:    "Test task",
		Status:   StatusPending,
		Priority: 5,
		Role:     "worker",
	}

	result := task.FormatOneLiner()
	if result == "" {
		t.Error("expected non-empty one-liner")
	}

	// Should contain key info
	if !containsStr(result, "T-001") {
		t.Error("expected one-liner to contain task ID")
	}
	if !containsStr(result, "Test task") {
		t.Error("expected one-liner to contain title")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
