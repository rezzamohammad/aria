package adapters

import (
	"context"
	"os/exec"
	"time"
)

// GeminiAdapter wraps the `gemini` CLI (Google Gemini CLI)
type GeminiAdapter struct{}

func (a *GeminiAdapter) Name() string { return "gemini-cli" }

func (a *GeminiAdapter) IsAvailable() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (a *GeminiAdapter) Capabilities() []Capability {
	return []Capability{CapResearch, CapDocs, CapCode}
}

func (a *GeminiAdapter) Execute(ctx context.Context, job Job) (*Result, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx,
		"gemini",
		"-p", job.Prompt,
		"--yolo",
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
