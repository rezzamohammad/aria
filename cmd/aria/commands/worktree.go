package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// WorktreeCmd is the parent command for worktree management.
var WorktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Git worktree management commands",
}

func init() {
	// worktree list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all active worktrees",
		RunE:  runWorktreeList,
	}
	WorktreeCmd.AddCommand(listCmd)

	// worktree clean
	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove merged/abandoned worktrees",
		RunE:  runWorktreeClean,
	}
	WorktreeCmd.AddCommand(cleanCmd)
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if !isGitRepo() {
		return fmt.Errorf("not a git repository — worktree commands require git")
	}

	out, err := exec.Command("git", "worktree", "list").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree list failed: %w\n%s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) <= 1 {
		fmt.Println("No additional worktrees. Only the main worktree exists.")
		return nil
	}

	fmt.Printf("Active worktrees (%d):\n\n", len(lines))
	for _, line := range lines {
		fmt.Println("  " + line)
	}

	return nil
}

func runWorktreeClean(cmd *cobra.Command, args []string) error {
	if !isGitRepo() {
		return fmt.Errorf("not a git repository — worktree commands require git")
	}

	// Prune stale worktrees
	out, err := exec.Command("git", "worktree", "prune").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree prune failed: %w\n%s", err, string(out))
	}

	fmt.Println("✓ Pruned stale worktree references")

	// List remaining worktrees
	listOut, err := exec.Command("git", "worktree", "list").CombinedOutput()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(listOut)), "\n")
		fmt.Printf("  %d worktree(s) remaining\n", len(lines))
	}

	return nil
}

// isGitRepo checks if the current directory is inside a git repository.
func isGitRepo() bool {
	_, err := exec.Command("git", "rev-parse", "--git-dir").CombinedOutput()
	return err == nil
}

// CreateWorktree creates a new git worktree for a task.
func CreateWorktree(baseDir string, taskID string, branch string) (string, error) {
	wtPath := fmt.Sprintf("%s/%s", baseDir, taskID)

	// Create branch and worktree
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	return wtPath, nil
}

// RemoveWorktree removes a git worktree.
func RemoveWorktree(wtPath string) error {
	cmd := exec.Command("git", "worktree", "remove", wtPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
