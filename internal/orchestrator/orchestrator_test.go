package orchestrator

import (
	"testing"

	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/task"
)

func TestClassifyFailure(t *testing.T) {
	tests := []struct {
		reason   string
		expected string
	}{
		{"context deadline exceeded", "timeout"},
		{"timeout waiting for response", "timeout"},
		{"context canceled by user", "timeout"},
		{"build failed: syntax error in main.go", "compile_error"},
		{"compile error: undefined variable", "compile_error"},
		{"syntax error on line 42", "compile_error"},
		{"file not found: config.yaml", "missing_context"},
		{"no such file or directory", "missing_context"},
		{"missing dependency xyz", "missing_context"},
		{"process killed by signal", "tool_crash"},
		{"segfault in codex", "tool_crash"},
		{"agent crashed unexpectedly", "tool_crash"},
		{"score below threshold: 45", "score_below_threshold"},
		{"something completely different", "unknown"},
	}

	for _, tt := range tests {
		got := classifyFailure(tt.reason)
		if got != tt.expected {
			t.Errorf("classifyFailure(%q) = %q, want %q", tt.reason, got, tt.expected)
		}
	}
}

func TestSuggestRecovery(t *testing.T) {
	tests := []struct {
		failureType string
		nonEmpty    bool
	}{
		{"timeout", true},
		{"compile_error", true},
		{"missing_context", true},
		{"tool_crash", true},
		{"score_below_threshold", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		got := suggestRecovery(tt.failureType)
		if tt.nonEmpty && got == "" {
			t.Errorf("suggestRecovery(%q) returned empty string", tt.failureType)
		}
	}
}

func TestSuggestRecoveryContent(t *testing.T) {
	// Each failure type should have a distinct recovery suggestion
	suggestions := map[string]string{}
	types := []string{"timeout", "compile_error", "missing_context", "tool_crash", "score_below_threshold", "unknown"}

	for _, ft := range types {
		s := suggestRecovery(ft)
		suggestions[ft] = s
	}

	// At least timeout and compile_error should have different suggestions
	if suggestions["timeout"] == suggestions["compile_error"] {
		t.Error("timeout and compile_error should have different recovery suggestions")
	}
}

func TestTruncateOutput(t *testing.T) {
	short := "hello"
	if truncateOutput(short, 10) != "hello" {
		t.Error("short string should not be truncated")
	}

	long := "abcdefghijklmnop"
	result := truncateOutput(long, 5)
	if len(result) <= 5 {
		t.Error("truncated result should include ellipsis marker")
	}
	if result[:5] != "abcde" {
		t.Errorf("expected 'abcde' prefix, got %q", result[:5])
	}
}

func TestTruncateOutputExactLength(t *testing.T) {
	exact := "12345"
	result := truncateOutput(exact, 5)
	if result != "12345" {
		t.Errorf("exact length string should not be truncated, got %q", result)
	}
}

func TestTruncateOutputEmpty(t *testing.T) {
	result := truncateOutput("", 10)
	if result != "" {
		t.Errorf("empty string should remain empty, got %q", result)
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("hello world", "world") {
		t.Error("expected 'world' to be found in 'hello world'")
	}
	if !containsAny("hello world", "foo", "world") {
		t.Error("expected 'world' to be found with multiple substrs")
	}
	if containsAny("hello world", "foo", "bar") {
		t.Error("expected no match for 'foo' and 'bar'")
	}
	if containsAny("", "foo") {
		t.Error("expected no match in empty string")
	}
}

func TestShouldVerify(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"worker", true},
		{"architect", true},
		{"researcher", true},
		{"pm", true},
		{"qa", true},
		{"security", true},
		{"verifier", false},     // verifiers don't re-verify
		{"orchestrator", false}, // orchestrator decisions don't need verification
	}

	for _, tt := range tests {
		tk := &task.Task{Role: tt.role}
		got := shouldVerify(tk)
		if got != tt.expected {
			t.Errorf("shouldVerify(role=%q) = %v, want %v", tt.role, got, tt.expected)
		}
	}
}

func TestShouldVerifyAllRoles(t *testing.T) {
	// All roles except verifier and orchestrator should need verification
	noVerify := map[roles.Role]bool{
		roles.RoleVerifier:     true,
		roles.RoleOrchestrator: true,
	}

	for _, role := range roles.AllRoles {
		tk := &task.Task{Role: string(role)}
		result := shouldVerify(tk)
		if noVerify[role] && result {
			t.Errorf("role %q should NOT need verification", role)
		}
		if !noVerify[role] && !result {
			t.Errorf("role %q SHOULD need verification", role)
		}
	}
}
