package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupGitRepo creates a temporary git repo for testing.
func setupGitRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	// Initialize a git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup failed (%v): %s", args, string(out))
		}
	}

	// Create initial commit
	f := filepath.Join(dir, "README.md")
	os.WriteFile(f, []byte("# Test\n"), 0o644)
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s", string(out))
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestNewManager(t *testing.T) {
	m := NewManager(".aria/worktrees", "/tmp/repo")
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.baseDir != ".aria/worktrees" {
		t.Errorf("expected baseDir '.aria/worktrees', got %q", m.baseDir)
	}
}

func TestCreateAndRemoveWorktree(t *testing.T) {
	repoDir, cleanup := setupGitRepo(t)
	defer cleanup()

	m := NewManager(".aria/worktrees", repoDir)

	// Create worktree
	wtPath, err := m.Create("T-001")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Verify it exists
	if !m.Exists(wtPath) {
		t.Error("worktree should exist after creation")
	}

	// Verify the path is correct
	expected := filepath.Join(repoDir, ".aria", "worktrees", "T-001")
	if wtPath != expected {
		t.Errorf("expected path %q, got %q", expected, wtPath)
	}

	// Remove it
	if err := m.Remove(wtPath); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	// Verify it's gone
	if m.Exists(wtPath) {
		t.Error("worktree should not exist after removal")
	}
}

func TestListWorktrees(t *testing.T) {
	repoDir, cleanup := setupGitRepo(t)
	defer cleanup()

	m := NewManager(".aria/worktrees", repoDir)

	// Initial list (just the main worktree)
	initial, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	initialCount := len(initial)

	// Create a worktree
	wtPath, err := m.Create("T-001")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// List should have one more
	after, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(after) != initialCount+1 {
		t.Errorf("expected %d worktrees, got %d", initialCount+1, len(after))
	}

	// Clean up
	m.Remove(wtPath)
}

func TestPrune(t *testing.T) {
	repoDir, cleanup := setupGitRepo(t)
	defer cleanup()

	m := NewManager(".aria/worktrees", repoDir)

	// Prune should not error even with no stale worktrees
	if err := m.Prune(); err != nil {
		t.Fatalf("Prune error: %v", err)
	}
}

func TestCreateNotGitRepo(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(".aria/worktrees", dir)

	_, err := m.Create("T-001")
	if err == nil {
		t.Error("expected error when creating worktree in non-git dir")
	}
}

func TestParseWorktreeList(t *testing.T) {
	output := `worktree /home/user/repo
HEAD abc123def456
branch refs/heads/main

worktree /home/user/repo/.aria/worktrees/T-001
HEAD def456abc123
branch refs/heads/aria/T-001

`

	worktrees := parseWorktreeList(output)
	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
	}

	if worktrees[0].Path != "/home/user/repo" {
		t.Errorf("expected first path '/home/user/repo', got %q", worktrees[0].Path)
	}
	if worktrees[0].Branch != "refs/heads/main" {
		t.Errorf("expected branch 'refs/heads/main', got %q", worktrees[0].Branch)
	}
	if worktrees[1].Path != "/home/user/repo/.aria/worktrees/T-001" {
		t.Errorf("unexpected second worktree path: %q", worktrees[1].Path)
	}
}

func TestExistsNonexistent(t *testing.T) {
	m := NewManager(".aria/worktrees", "/tmp")
	if m.Exists("/nonexistent/path/that/doesnt/exist") {
		t.Error("Exists should return false for nonexistent path")
	}
}
