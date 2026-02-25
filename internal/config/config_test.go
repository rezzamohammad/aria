package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Project.Name != "aria" {
		t.Errorf("expected project name 'aria', got %q", cfg.Project.Name)
	}
	if cfg.Project.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got %q", cfg.Project.Version)
	}
	if cfg.Database.Path != ".aria/aria.db" {
		t.Errorf("expected db path '.aria/aria.db', got %q", cfg.Database.Path)
	}
	if cfg.Agents.MaxPoolSize != 20 {
		t.Errorf("expected max pool size 20, got %d", cfg.Agents.MaxPoolSize)
	}
	if cfg.Agents.HeartbeatIntervalSec != 10 {
		t.Errorf("expected heartbeat interval 10, got %d", cfg.Agents.HeartbeatIntervalSec)
	}
	if cfg.Verification.MinScore != 80 {
		t.Errorf("expected min score 80, got %d", cfg.Verification.MinScore)
	}
	if cfg.Verification.MaxIterations != 10 {
		t.Errorf("expected max iterations 10, got %d", cfg.Verification.MaxIterations)
	}
	if cfg.Context.WarnPct != 0.70 {
		t.Errorf("expected warn pct 0.70, got %f", cfg.Context.WarnPct)
	}
	if cfg.Context.CriticalPct != 0.90 {
		t.Errorf("expected critical pct 0.90, got %f", cfg.Context.CriticalPct)
	}
	if cfg.Worktree.BaseDir != ".aria/worktrees" {
		t.Errorf("expected worktree base dir '.aria/worktrees', got %q", cfg.Worktree.BaseDir)
	}
}

func TestDefaultConfigToolMapping(t *testing.T) {
	cfg := DefaultConfig()

	expectedRoles := []string{
		"orchestrator", "worker", "verifier", "architect", "researcher",
		"pm", "qa", "security", "librarian", "doc_system", "release_ops",
	}

	for _, role := range expectedRoles {
		tools, ok := cfg.ToolMapping[role]
		if !ok {
			t.Errorf("missing tool mapping for role %q", role)
			continue
		}
		if len(tools) == 0 {
			t.Errorf("empty tool chain for role %q", role)
		}
	}
}

func TestDefaultConfigVerificationWeights(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Verification.Weights["technical"] != 50 {
		t.Errorf("expected technical weight 50, got %d", cfg.Verification.Weights["technical"])
	}
	if cfg.Verification.Weights["content"] != 35 {
		t.Errorf("expected content weight 35, got %d", cfg.Verification.Weights["content"])
	}
	if cfg.Verification.Weights["aesthetic"] != 15 {
		t.Errorf("expected aesthetic weight 15, got %d", cfg.Verification.Weights["aesthetic"])
	}

	// Sum should be 100
	total := 0
	for _, w := range cfg.Verification.Weights {
		total += w
	}
	if total != 100 {
		t.Errorf("expected verification weights sum to 100, got %d", total)
	}
}

func TestWriteDefaultAndLoad(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "aria.toml")

	// Write default config
	if err := WriteDefault(configPath); err != nil {
		t.Fatalf("WriteDefault error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Load it back
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Project.Name != "aria" {
		t.Errorf("expected project name 'aria', got %q", cfg.Project.Name)
	}
	if cfg.Agents.MaxPoolSize != 20 {
		t.Errorf("expected max pool size 20, got %d", cfg.Agents.MaxPoolSize)
	}
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error when aria.toml not found")
	}
}

func TestFindConfigFileWalksUp(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(subdir, 0o755)

	// Write config at root
	WriteDefault(filepath.Join(dir, "aria.toml"))

	// Should find it from subdir
	found, err := findConfigFile(subdir)
	if err != nil {
		t.Fatalf("findConfigFile error: %v", err)
	}
	expected := filepath.Join(dir, "aria.toml")
	if found != expected {
		t.Errorf("expected %q, got %q", expected, found)
	}
}
