package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Manager handles git worktree lifecycle for tasks.
type Manager struct {
	baseDir string
	repoDir string
}

// NewManager creates a new worktree manager.
// baseDir is the directory where worktrees are stored (e.g., ".aria/worktrees").
// repoDir is the root of the git repository.
func NewManager(baseDir, repoDir string) *Manager {
	return &Manager{
		baseDir: baseDir,
		repoDir: repoDir,
	}
}

// Create creates a new git worktree for the given task.
// Returns the absolute path to the worktree.
func (m *Manager) Create(taskID string) (string, error) {
	if !m.isGitRepo() {
		return "", fmt.Errorf("not a git repository at %s", m.repoDir)
	}

	// Ensure base directory exists
	absBase, err := filepath.Abs(filepath.Join(m.repoDir, m.baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to resolve base dir: %w", err)
	}
	if err := os.MkdirAll(absBase, 0o755); err != nil {
		return "", fmt.Errorf("failed to create worktree base dir: %w", err)
	}

	wtPath := filepath.Join(absBase, taskID)
	branch := fmt.Sprintf("aria/%s", taskID)

	// Create worktree with new branch
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w\n%s", err, string(output))
	}

	return wtPath, nil
}

// Remove removes a git worktree.
func (m *Manager) Remove(wtPath string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", wtPath)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\n%s", err, string(output))
	}
	return nil
}

// List returns all active worktrees.
func (m *Manager) List() ([]WorktreeInfo, error) {
	if !m.isGitRepo() {
		return nil, fmt.Errorf("not a git repository at %s", m.repoDir)
	}

	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w\n%s", err, string(output))
	}

	return parseWorktreeList(string(output)), nil
}

// Prune removes stale worktree references.
func (m *Manager) Prune() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %w\n%s", err, string(output))
	}
	return nil
}

// Exists checks if a worktree exists at the given path.
func (m *Manager) Exists(wtPath string) bool {
	info, err := os.Stat(wtPath)
	return err == nil && info.IsDir()
}

func (m *Manager) isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = m.repoDir
	return cmd.Run() == nil
}

// WorktreeInfo represents info about a single worktree.
type WorktreeInfo struct {
	Path   string
	HEAD   string
	Branch string
}

// parseWorktreeList parses git worktree list --porcelain output.
func parseWorktreeList(output string) []WorktreeInfo {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			current.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.Branch = strings.TrimPrefix(line, "branch ")
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees
}
