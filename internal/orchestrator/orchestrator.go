package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/aria-cli/aria/internal/agent"
	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/config"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/aria-cli/aria/internal/executor"
	"github.com/aria-cli/aria/internal/logger"
	"github.com/aria-cli/aria/internal/task"
	"github.com/aria-cli/aria/internal/worktree"
)

// Orchestrator is the central brain of ARIA.
// It manages the scheduling loop: assigning tasks to agents,
// monitoring progress, handling verification, and driving recovery.
type Orchestrator struct {
	db      *sql.DB
	cfg     *config.Config
	pool    *agent.Pool
	router  *agent.Router
	wtMgr   *worktree.Manager
	log     *logger.Logger
	auto    bool // fully autonomous mode
	mu      sync.Mutex
	running map[string]context.CancelFunc // agentID -> cancel
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// New creates a new Orchestrator.
func New(db *sql.DB, cfg *config.Config, pool *agent.Pool, router *agent.Router, log *logger.Logger, auto bool) *Orchestrator {
	repoDir := "."
	return &Orchestrator{
		db:      db,
		cfg:     cfg,
		pool:    pool,
		router:  router,
		wtMgr:   worktree.NewManager(cfg.Worktree.BaseDir, repoDir),
		log:     log,
		auto:    auto,
		running: make(map[string]context.CancelFunc),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start begins the orchestration loop.
func (o *Orchestrator) Start() {
	go o.loop()
}

// Stop gracefully shuts down the orchestrator.
func (o *Orchestrator) Stop() {
	close(o.stopCh)

	// Cancel all running executions
	o.mu.Lock()
	for id, cancel := range o.running {
		o.log.Info("orchestrator", "cancelling agent %s", id[:8])
		cancel()
	}
	o.mu.Unlock()

	// Wait for loop to finish
	<-o.doneCh
	o.log.Info("orchestrator", "stopped")
}

// loop is the main scheduling loop.
func (o *Orchestrator) loop() {
	defer close(o.doneCh)

	ticker := time.NewTicker(time.Duration(o.cfg.Agents.HeartbeatIntervalSec) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	o.scheduleTick()

	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.scheduleTick()
		}
	}
}

// scheduleTick runs one iteration of the scheduling loop.
func (o *Orchestrator) scheduleTick() {
	// 1. Update heartbeats for all active agents
	o.pool.Heartbeat()

	// 2. Check for idle agents and assign tasks
	o.assignTasks()

	// 3. Check for stale agents (no heartbeat for 3x interval)
	o.checkStaleAgents()
}

// assignTasks finds idle agents and assigns them pending tasks.
func (o *Orchestrator) assignTasks() {
	idleAgents := o.pool.IdleAgents()
	if len(idleAgents) == 0 {
		return
	}

	for _, ap := range idleAgents {
		// Try to claim a task for this agent's role
		t, err := task.ClaimNextTask(o.db, ap.ID, string(ap.Role))
		if err != nil {
			o.log.Error("orchestrator", "failed to claim task for agent %s: %v", ap.ID[:8], err)
			continue
		}
		if t == nil {
			// No task available for this role, try any role if agent is a worker
			continue
		}

		o.log.Info("orchestrator", "assigned task %s to agent %s (%s)", t.ID, ap.ID[:8], ap.Role)

		// Create worktree for this task
		wtPath := ""
		if o.wtMgr != nil {
			path, err := o.wtMgr.Create(t.ID)
			if err != nil {
				o.log.Warn("orchestrator", "failed to create worktree for %s: %v (continuing without)", t.ID, err)
			} else {
				wtPath = path
				o.log.Info("orchestrator", "created worktree for %s at %s", t.ID, wtPath)
				// Update task with worktree path
				o.db.Exec("UPDATE tasks SET worktree_path = ? WHERE id = ?", wtPath, t.ID)
			}
		}

		// Launch execution in background
		go o.executeTask(ap, t, wtPath)
	}
}

// executeTask runs a task on an agent and handles the result.
func (o *Orchestrator) executeTask(ap *agent.AgentProcess, t *task.Task, wtPath string) {
	ctx, cancel := context.WithCancel(context.Background())

	o.mu.Lock()
	o.running[ap.ID] = cancel
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		delete(o.running, ap.ID)
		o.mu.Unlock()
		cancel()
	}()

	exec := executor.New(o.db, o.log)
	result := exec.Execute(ctx, t, ap.ID, ap.CLITool, wtPath)

	o.handleResult(result, t)
}

// handleResult processes the outcome of an agent execution.
func (o *Orchestrator) handleResult(result *executor.Result, t *task.Task) {
	if result.Status == "failed" {
		o.handleFailure(result, t)
		return
	}

	// Check if this is a verifier result
	if t.Role == string(roles.RoleVerifier) && result.Score > 0 {
		o.handleVerification(result, t)
		return
	}

	// For non-verifier tasks, check if we need verification
	if shouldVerify(t) {
		o.log.Info("orchestrator", "task %s completed, queuing for verification", t.ID)
		if err := queries.UpdateTaskStatus(o.db, t.ID, "verifying"); err != nil {
			o.log.Error("orchestrator", "failed to update task %s to verifying: %v", t.ID, err)
		}
		// Create a verification task
		o.createVerificationTask(t, result)
		return
	}

	// Mark as done
	o.completeTask(t, result)
}

// handleVerification processes a verifier's score result.
func (o *Orchestrator) handleVerification(result *executor.Result, t *task.Task) {
	minScore := o.cfg.Verification.MinScore
	maxIter := o.cfg.Verification.MaxIterations

	if result.Score >= minScore {
		// Approved!
		o.log.Info("orchestrator", "task %s APPROVED with score %d/%d", t.ID, result.Score, 100)
		if err := task.CompleteTask(o.db, t.ID, result.Score, result.Feedback); err != nil {
			o.log.Error("orchestrator", "failed to complete task %s: %v", t.ID, err)
		}
		return
	}

	// Score below threshold — iterate
	o.log.Warn("orchestrator", "task %s scored %d (below %d), iterating", t.ID, result.Score, minScore)

	// Check iteration count
	recoveryCount := t.RecoveryCount + 1
	if recoveryCount >= maxIter {
		o.log.Error("orchestrator", "task %s exceeded max iterations (%d), marking failed", t.ID, maxIter)
		if err := task.FailTask(o.db, t.ID, fmt.Sprintf("exceeded max verification iterations (%d), last score: %d", maxIter, result.Score)); err != nil {
			o.log.Error("orchestrator", "failed to fail task %s: %v", t.ID, err)
		}
		return
	}

	// Re-delegate: reset to pending with feedback
	feedback := fmt.Sprintf("Verification score: %d/%d. Feedback: %s", result.Score, 100, result.Feedback)
	o.db.Exec("UPDATE tasks SET status = 'pending', failure_reason = ?, recovery_count = ? WHERE id = ?",
		feedback, recoveryCount, t.ID)
	o.log.Info("orchestrator", "re-delegated task %s (iteration %d/%d)", t.ID, recoveryCount, maxIter)
}

// handleFailure processes a failed task execution.
func (o *Orchestrator) handleFailure(result *executor.Result, t *task.Task) {
	reason := "unknown failure"
	if result.Error != nil {
		reason = result.Error.Error()
	}

	failureType := classifyFailure(reason)
	o.log.Error("orchestrator", "task %s failed (%s): %s", t.ID, failureType, reason)

	// Record failure
	o.recordFailure(t.ID, result.SessionID, failureType, reason)

	// Attempt recovery
	if err := task.FailTask(o.db, t.ID, reason); err != nil {
		o.log.Error("orchestrator", "failed to record task failure: %v", err)
	}
}

// createVerificationTask creates a new task for verifying completed work.
func (o *Orchestrator) createVerificationTask(originalTask *task.Task, result *executor.Result) {
	verifyTask := &queries.Task{
		ID:          fmt.Sprintf("V-%s", originalTask.ID),
		Title:       fmt.Sprintf("Verify: %s", originalTask.Title),
		Description: sql.NullString{String: fmt.Sprintf("Verify the output of task %s.\n\nOutput:\n%s", originalTask.ID, truncateOutput(result.Output, 2000)), Valid: true},
		Role:        string(roles.RoleVerifier),
		Status:      "pending",
		Priority:    originalTask.Priority, // Same priority as original
	}
	verifyTask.EpicID = sql.NullString{String: originalTask.EpicID, Valid: originalTask.EpicID != ""}

	if err := queries.InsertTask(o.db, verifyTask); err != nil {
		o.log.Error("orchestrator", "failed to create verification task: %v", err)
		// Fall back to completing without verification
		o.completeTask(originalTask, result)
	}
}

// completeTask marks a task as done.
func (o *Orchestrator) completeTask(t *task.Task, result *executor.Result) {
	score := result.Score
	if score == 0 {
		score = 100 // Default score for unverified tasks
	}
	if err := task.CompleteTask(o.db, t.ID, score, result.Feedback); err != nil {
		o.log.Error("orchestrator", "failed to complete task %s: %v", t.ID, err)
		return
	}
	o.log.Info("orchestrator", "task %s completed (score: %d)", t.ID, score)
}

// recordFailure records a failure event in the database.
func (o *Orchestrator) recordFailure(taskID, sessionID, failureType, description string) {
	_, err := o.db.Exec(`
		INSERT INTO failures (id, task_id, session_id, failure_type, description, recovery_action)
		VALUES (?, ?, ?, ?, ?, ?)`,
		fmt.Sprintf("F-%s-%d", taskID, time.Now().UnixNano()),
		taskID,
		sessionID,
		failureType,
		description,
		suggestRecovery(failureType),
	)
	if err != nil {
		o.log.Error("orchestrator", "failed to record failure: %v", err)
	}
}

// checkStaleAgents detects agents that haven't sent a heartbeat recently.
func (o *Orchestrator) checkStaleAgents() {
	agents, err := queries.ListAgents(o.db, "busy")
	if err != nil {
		return
	}

	staleThreshold := time.Duration(o.cfg.Agents.HeartbeatIntervalSec*3) * time.Second

	for _, a := range agents {
		if !a.LastHeartbeat.Valid {
			continue
		}
		lastBeat, err := time.Parse("2006-01-02 15:04:05", a.LastHeartbeat.String)
		if err != nil {
			continue
		}
		if time.Since(lastBeat) > staleThreshold {
			o.log.Warn("orchestrator", "agent %s appears stale (last heartbeat: %s)", a.ID[:8], a.LastHeartbeat.String)

			// Release the agent's current task back to pending
			if a.CurrentTaskID.Valid && a.CurrentTaskID.String != "" {
				if err := task.ReleaseTask(o.db, a.CurrentTaskID.String); err != nil {
					o.log.Error("orchestrator", "failed to release task %s: %v", a.CurrentTaskID.String, err)
				} else {
					o.log.Info("orchestrator", "released task %s from stale agent %s", a.CurrentTaskID.String, a.ID[:8])
				}
			}

			// Mark agent as crashed
			queries.UpdateAgentStatus(o.db, a.ID, "crashed", "", "")
		}
	}
}

// shouldVerify determines if a task needs verification.
func shouldVerify(t *task.Task) bool {
	role := roles.Role(t.Role)
	// Verifier tasks don't need re-verification
	if role == roles.RoleVerifier {
		return false
	}
	// Orchestrator decisions don't need verification
	if role == roles.RoleOrchestrator {
		return false
	}
	// Worker, architect, etc. should be verified
	return true
}

// classifyFailure determines the type of failure from the error message.
func classifyFailure(reason string) string {
	lower := fmt.Sprintf("%s", reason)
	switch {
	case containsAny(lower, "timeout", "deadline exceeded", "context canceled"):
		return "timeout"
	case containsAny(lower, "compile", "build failed", "syntax error"):
		return "compile_error"
	case containsAny(lower, "not found", "no such file", "missing"):
		return "missing_context"
	case containsAny(lower, "crash", "signal", "killed", "segfault"):
		return "tool_crash"
	case containsAny(lower, "score", "below threshold"):
		return "score_below_threshold"
	default:
		return "unknown"
	}
}

// suggestRecovery suggests a recovery action based on failure type.
func suggestRecovery(failureType string) string {
	switch failureType {
	case "timeout":
		return "restart with compressed context"
	case "compile_error":
		return "route to security/debugger agent for fix"
	case "missing_context":
		return "load additional context via librarian, retry"
	case "tool_crash":
		return "switch to backup CLI tool and retry"
	case "score_below_threshold":
		return "re-delegate with specific feedback from verifier"
	default:
		return "retry with same configuration"
	}
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}
