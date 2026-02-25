package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ClaudeAdapter wraps the `claude` CLI (Claude Code)
type ClaudeAdapter struct{}

func (a *ClaudeAdapter) Name() string { return "claude-code" }

func (a *ClaudeAdapter) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (a *ClaudeAdapter) Capabilities() []Capability {
	return []Capability{CapCode, CapRefactor, CapReview, CapTest, CapSecurity, CapDocs}
}

func (a *ClaudeAdapter) Execute(ctx context.Context, job Job) (*Result, error) {
	start := time.Now()

	// Inject CLAUDE.md into the worktree so Claude has full task context
	if job.WorktreeDir != "" && job.ClaudeMD != "" {
		mdPath := filepath.Join(job.WorktreeDir, "CLAUDE.md")
		if err := os.WriteFile(mdPath, []byte(job.ClaudeMD), 0644); err != nil {
			return nil, fmt.Errorf("claude: write CLAUDE.md: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx,
		"claude",
		"--dangerously-skip-permissions",
		"--print",
		job.Prompt,
	)
	if job.WorktreeDir != "" {
		cmd.Dir = job.WorktreeDir
	}

	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start).Seconds()

	status := "success"
	exitCode := 0
	if err != nil {
		status = "failed"
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
	}

	return &Result{
		TaskID:   job.TaskID,
		Agent:    a.Name(),
		Output:   string(out),
		Status:   status,
		Duration: elapsed,
		ExitCode: exitCode,
		Error:    err,
	}, nil
}
