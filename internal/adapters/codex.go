package adapters

import (
	"context"
	"os/exec"
	"time"
)

// CodexAdapter wraps the `codex` CLI (OpenAI Codex CLI)
type CodexAdapter struct{}

func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) IsAvailable() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (a *CodexAdapter) Capabilities() []Capability {
	return []Capability{CapCode, CapTest, CapRefactor}
}

func (a *CodexAdapter) Execute(ctx context.Context, job Job) (*Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx,
		"codex",
		"--approval-mode", "full-auto",
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
