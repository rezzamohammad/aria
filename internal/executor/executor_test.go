package executor

import (
	"testing"

	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/task"
)

func TestBuildCLIArgsClaude(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("claude-code", tk, "handoff content")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "claude" {
		t.Errorf("expected 'claude', got %q", args[0])
	}
	// Should have --print flag
	found := false
	for _, a := range args {
		if a == "--print" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected --print flag for claude-code")
	}
}

func TestBuildCLIArgsCodex(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("codex", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "codex" {
		t.Errorf("expected 'codex', got %q", args[0])
	}
}

func TestBuildCLIArgsGemini(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "researcher",
	}

	args := buildCLIArgs("gemini-cli", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "gemini" {
		t.Errorf("expected 'gemini', got %q", args[0])
	}
}

func TestBuildCLIArgsCrush(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("crush", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "crush" {
		t.Errorf("expected 'crush', got %q", args[0])
	}
}

func TestBuildCLIArgsDroids(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "release_ops",
	}

	args := buildCLIArgs("droids", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "droids" {
		t.Errorf("expected 'droids', got %q", args[0])
	}
	if args[1] != "run" {
		t.Errorf("expected 'run' subcommand, got %q", args[1])
	}
}

func TestBuildCLIArgsKiro(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("kiro", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args")
	}
	if args[0] != "kiro" {
		t.Errorf("expected 'kiro', got %q", args[0])
	}
}

func TestBuildCLIArgsGenericFallback(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("my-custom-tool", tk, "handoff")
	if len(args) == 0 {
		t.Fatal("expected non-empty args for generic fallback")
	}
	if args[0] != "my-custom-tool" {
		t.Errorf("expected 'my-custom-tool', got %q", args[0])
	}
}

func TestBuildCLIArgsSystemPromptIncluded(t *testing.T) {
	tk := &task.Task{
		ID:    "T-001",
		Title: "Test",
		Role:  "worker",
	}

	args := buildCLIArgs("claude-code", tk, "handoff content")

	// The system prompt should be in the args
	workerPrompt := roles.SystemPrompt(roles.RoleWorker)
	found := false
	for _, a := range args {
		if a == workerPrompt {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system prompt in claude-code args")
	}
}

func TestBuildCLIArgsAllToolsReturnNonEmpty(t *testing.T) {
	tools := []string{"claude-code", "codex", "gemini-cli", "crush", "kiro", "droids"}
	tk := &task.Task{ID: "T-001", Title: "Test", Role: "worker"}

	for _, tool := range tools {
		args := buildCLIArgs(tool, tk, "handoff")
		if len(args) == 0 {
			t.Errorf("expected non-empty args for tool %q", tool)
		}
	}
}

func TestBuildCLIArgsHandoffContent(t *testing.T) {
	tk := &task.Task{ID: "T-001", Title: "Test", Role: "worker"}
	handoff := "This is the handoff content with details"

	args := buildCLIArgs("claude-code", tk, handoff)

	// Handoff content should appear somewhere in the args
	found := false
	for _, a := range args {
		if len(a) >= len(handoff) {
			for i := 0; i <= len(a)-len(handoff); i++ {
				if a[i:i+len(handoff)] == handoff {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected handoff content to appear in CLI args")
	}
}
