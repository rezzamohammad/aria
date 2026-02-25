package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aria-cli/aria/internal/adapters"
	"github.com/aria-cli/aria/internal/claudemd"
	"github.com/aria-cli/aria/internal/task"
)

const (
	defaultMaxParallel = 20
	defaultTimeoutSecs = 1800 // 30 min per task
	defaultProjectName = "aria"
)

// Dispatcher ties the adapter registry to the task queue.
// It runs tasks in parallel up to MaxParallel, routing each to the best adapter.
type Dispatcher struct {
	registry    *adapters.Registry
	splitter    *claudemd.Splitter
	maxParallel int
	sem         chan struct{}
	ProjectName string
}

// New creates a Dispatcher with a pre-built adapter registry.
func New(registry *adapters.Registry, maxParallel int) *Dispatcher {
	if maxParallel <= 0 {
		maxParallel = defaultMaxParallel
	}
	return &Dispatcher{
		registry:    registry,
		splitter:    claudemd.NewSplitter(),
		maxParallel: maxParallel,
		sem:         make(chan struct{}, maxParallel),
		ProjectName: defaultProjectName,
	}
}

// RunTask executes a single task synchronously and returns the result.
func (d *Dispatcher) RunTask(ctx context.Context, t *task.Task, priorWork string) *adapters.Result {
	cap, ok := adapters.RoleToCapability[t.Role]
	if !ok {
		cap = adapters.CapCode // sensible fallback
	}

	adapter, err := d.registry.BestFor(cap)
	if err != nil {
		return &adapters.Result{
			TaskID: t.ID,
			Status: "no_agent",
			Error:  fmt.Errorf("dispatcher: %w", err),
		}
	}

	// Build CLAUDE.md and take only the first chunk to stay within context limits
	md := claudemd.Generate(t, d.ProjectName, priorWork)
	mdChunk := d.splitter.FirstChunk(md)

	prompt := buildPrompt(t)

	job := adapters.Job{
		TaskID:      t.ID,
		TaskRole:    t.Role,
		Prompt:      prompt,
		WorktreeDir: t.WorktreePath,
		ClaudeMD:    mdChunk,
		TimeoutSecs: defaultTimeoutSecs,
	}

	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(defaultTimeoutSecs)*time.Second)
	defer cancel()

	result, execErr := adapter.Execute(taskCtx, job)
	if execErr != nil && result == nil {
		return &adapters.Result{
			TaskID: t.ID,
			Agent:  adapter.Name(),
			Status: "failed",
			Error:  execErr,
		}
	}

	return result
}

// RunBatch dispatches multiple tasks concurrently, capped by MaxParallel.
// Results are returned in the same order as the input slice.
func (d *Dispatcher) RunBatch(ctx context.Context, tasks []*task.Task, priorWork string) []*adapters.Result {
	results := make([]*adapters.Result, len(tasks))
	var wg sync.WaitGroup

	for i, t := range tasks {
		wg.Add(1)
		go func(idx int, tsk *task.Task) {
			defer wg.Done()
			d.sem <- struct{}{}         // acquire parallelism slot
			defer func() { <-d.sem }() // release on done

			results[idx] = d.RunTask(ctx, tsk, priorWork)
		}(i, t)
	}

	wg.Wait()
	return results
}

// AvailableAdapters returns the names of all discovered CLI tools.
func (d *Dispatcher) AvailableAdapters() []string {
	names := make([]string, 0)
	for _, a := range d.registry.Available() {
		names = append(names, a.Name())
	}
	return names
}

// buildPrompt constructs the full prompt string from a Task.
func buildPrompt(t *task.Task) string {
	if t.Description != "" {
		return fmt.Sprintf("%s\n\n%s", t.Title, t.Description)
	}
	return t.Title
}
