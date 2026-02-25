package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the full aria.toml configuration.
type Config struct {
	Project      ProjectConfig       `toml:"project"`
	Database     DatabaseConfig      `toml:"database"`
	Agents       AgentsConfig        `toml:"agents"`
	Verification VerificationConfig  `toml:"verification"`
	Context      ContextConfig       `toml:"context"`
	Worktree     WorktreeConfig      `toml:"worktree"`
	ToolMapping  map[string][]string `toml:"tool_mapping"`
}

type ProjectConfig struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
	Author      string `toml:"author"`
}

type DatabaseConfig struct {
	Path string `toml:"path"`
}

type AgentsConfig struct {
	MaxPoolSize          int `toml:"max_pool_size"`
	HeartbeatIntervalSec int `toml:"heartbeat_interval_sec"`
	DefaultMaxRetries    int `toml:"default_max_retries"`
}

type VerificationConfig struct {
	MinScore      int            `toml:"min_score"`
	Weights       map[string]int `toml:"weights"`
	MaxIterations int            `toml:"max_iterations"`
}

type ContextConfig struct {
	WarnPct     float64 `toml:"warn_pct"`
	CriticalPct float64 `toml:"critical_pct"`
}

type WorktreeConfig struct {
	BaseDir     string `toml:"base_dir"`
	AutoCleanup bool   `toml:"auto_cleanup"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name:        "aria",
			Version:     "0.1.0",
			Description: "Autonomous multi-agent CLI/TUI orchestration layer",
			Author:      "you",
		},
		Database: DatabaseConfig{
			Path: ".aria/aria.db",
		},
		Agents: AgentsConfig{
			MaxPoolSize:          20,
			HeartbeatIntervalSec: 10,
			DefaultMaxRetries:    3,
		},
		Verification: VerificationConfig{
			MinScore:      80,
			Weights:       map[string]int{"technical": 50, "content": 35, "aesthetic": 15},
			MaxIterations: 10,
		},
		Context: ContextConfig{
			WarnPct:     0.70,
			CriticalPct: 0.90,
		},
		Worktree: WorktreeConfig{
			BaseDir:     ".aria/worktrees",
			AutoCleanup: true,
		},
		ToolMapping: map[string][]string{
			"orchestrator": {"claude-code"},
			"worker":       {"claude-code", "codex", "crush"},
			"verifier":     {"claude-code"},
			"architect":    {"claude-code", "gemini-cli"},
			"researcher":   {"gemini-cli", "claude-code"},
			"pm":           {"claude-code"},
			"qa":           {"codex", "claude-code"},
			"security":     {"claude-code"},
			"librarian":    {"claude-code"},
			"doc_system":   {"claude-code"},
			"release_ops":  {"droids", "claude-code"},
		},
	}
}

// Load reads aria.toml from the given directory (or walks up to find it).
func Load(dir string) (*Config, error) {
	configPath, err := findConfigFile(dir)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	return cfg, nil
}

// findConfigFile walks up from dir looking for aria.toml.
func findConfigFile(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(absDir, "aria.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(absDir)
		if parent == absDir {
			break
		}
		absDir = parent
	}

	return "", fmt.Errorf("aria.toml not found (searched from %s upward)", dir)
}

// WriteDefault writes a default aria.toml to the given path.
func WriteDefault(path string) error {
	cfg := DefaultConfig()
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(cfg)
}
