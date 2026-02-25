package adapters

import (
	"context"
	"os/exec"
	"time"
)

// CrushAdapter wraps Charm's `crush` CLI
type CrushAdapter struct{}

func (a *CrushAdapter) Name() string { return "crush" }

func (a *CrushAdapter) IsAvailable() bool {
	_, err := exec.LookPath("crush")
	return err == nil
}

func (a *CrushAdapter) Capabilities() []Capability {
	return []Capability{CapCode, CapRefactor}
}

func (a *CrushAdapter) Execute(ctx context.Context, job Job) (*Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "crush", job.Prompt)
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
