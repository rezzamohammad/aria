package adapters

import (
	"context"
	"os/exec"
	"time"
)

// DroidsAdapter wraps the Factory `droids` CLI
type DroidsAdapter struct{}

func (a *DroidsAdapter) Name() string { return "droids" }

func (a *DroidsAdapter) IsAvailable() bool {
	_, err := exec.LookPath("droids")
	return err == nil
}

func (a *DroidsAdapter) Capabilities() []Capability {
	return []Capability{CapRelease, CapCode}
}

func (a *DroidsAdapter) Execute(ctx context.Context, job Job) (*Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "droids", "run", "--task", job.Prompt)
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
