package agent

import (
	"testing"

	"github.com/aria-cli/aria/internal/agent/roles"
)

func TestNewRouterDefault(t *testing.T) {
	r := NewRouter(nil)
	if r == nil {
		t.Fatal("expected non-nil router")
	}

	// Should have default mappings
	tools := r.GetToolChain(roles.RoleWorker)
	if len(tools) == 0 {
		t.Error("expected non-empty tool chain for worker")
	}
}

func TestNewRouterCustomMapping(t *testing.T) {
	custom := map[string][]string{
		"worker": {"custom-tool", "fallback-tool"},
	}
	r := NewRouter(custom)

	tools := r.GetToolChain(roles.RoleWorker)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0] != "custom-tool" {
		t.Errorf("expected first tool 'custom-tool', got %q", tools[0])
	}
}

func TestResolveToolOrDefault(t *testing.T) {
	r := NewRouter(nil)

	// Should return something even if tools aren't installed
	tool := r.ResolveToolOrDefault(roles.RoleWorker)
	if tool == "" {
		t.Error("expected non-empty tool")
	}
}

func TestResolveToolOrDefaultUnknownRole(t *testing.T) {
	r := NewRouter(nil)

	// Unknown role should fallback to "claude-code"
	tool := r.ResolveToolOrDefault(roles.Role("nonexistent"))
	if tool != "claude-code" {
		t.Errorf("expected 'claude-code' fallback, got %q", tool)
	}
}

func TestSetToolMapping(t *testing.T) {
	r := NewRouter(nil)
	r.SetToolMapping(roles.RoleWorker, []string{"new-tool"})

	tools := r.GetToolChain(roles.RoleWorker)
	if len(tools) != 1 || tools[0] != "new-tool" {
		t.Errorf("expected [new-tool], got %v", tools)
	}
}

func TestGetToolChainNonexistent(t *testing.T) {
	r := NewRouter(nil)
	tools := r.GetToolChain(roles.Role("nonexistent"))
	if tools != nil {
		t.Errorf("expected nil for nonexistent role, got %v", tools)
	}
}
