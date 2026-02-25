package executor

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/aria-cli/aria/internal/handoff"
	"github.com/aria-cli/aria/internal/logger"
	"github.com/aria-cli/aria/internal/task"
	"github.com/google/uuid"
)

// Result represents the outcome of an agent execution.
type Result struct {
	TaskID    string
	SessionID string
	Status    string // "completed", "failed"
	Output    string
	Score     int
	Feedback  string
	Error     error
}

// Executor manages the lifecycle of a single agent process execution.
type Executor struct {
	db     *sql.DB
	log    *logger.Logger
	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

// New creates a new Executor.
func New(db *sql.DB, log *logger.Logger) *Executor {
	return &Executor{
		db:  db,
		log: log,
	}
}

// Execute runs a CLI tool for a given task and agent.
// It creates a session, generates the handoff, launches the process,
// captures output, and returns the result.
func (e *Executor) Execute(ctx context.Context, t *task.Task, agentID string, cliTool string, worktreePath string) *Result {
	sessionID := uuid.New().String()

	// Create session record
	sess := &queries.Session{
		ID:      sessionID,
		TaskID:  t.ID,
		AgentID: agentID,
		CLITool: cliTool,
		Status:  "active",
	}
	if worktreePath != "" {
		sess.WorktreePath = sql.NullString{String: worktreePath, Valid: true}
	}
	if err := queries.InsertSession(e.db, sess); err != nil {
		e.log.Error(agentID[:8], "failed to create session: %v", err)
		return &Result{TaskID: t.ID, SessionID: sessionID, Status: "failed", Error: err}
	}

	// Update task status to running
	if err := queries.UpdateTaskStatus(e.db, t.ID, "running"); err != nil {
		e.log.Error(agentID[:8], "failed to update task status: %v", err)
	}

	// Update agent status to busy
	if err := queries.UpdateAgentStatus(e.db, agentID, "busy", t.ID, sessionID); err != nil {
		e.log.Error(agentID[:8], "failed to update agent status: %v", err)
	}

	e.log.Info(agentID[:8], "started session %s for task %s using %s", sessionID[:8], t.ID, cliTool)

	// Generate handoff envelope
	handoffContent := handoff.GenerateHandoff(t)

	// Write handoff to file
	handoffPath := e.writeHandoff(t, handoffContent, worktreePath)
	if handoffPath != "" {
		t.HandoffPath = handoffPath
	}

	// Execute the CLI tool
	result := e.runCLITool(ctx, t, agentID, cliTool, handoffContent, worktreePath, sessionID)

	// End session
	sessStatus := "completed"
	if result.Status == "failed" {
		sessStatus = "failed"
	}
	if err := queries.EndSession(e.db, sessionID, sessStatus); err != nil {
		e.log.Error(agentID[:8], "failed to end session: %v", err)
	}

	// Update agent back to idle
	if err := queries.UpdateAgentStatus(e.db, agentID, "idle", "", ""); err != nil {
		e.log.Error(agentID[:8], "failed to update agent status: %v", err)
	}

	return result
}

// Kill terminates the running process if any.
func (e *Executor) Kill() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
}

// runCLITool executes the actual CLI tool process.
func (e *Executor) runCLITool(ctx context.Context, t *task.Task, agentID string, cliTool string, handoffContent string, worktreePath string, sessionID string) *Result {
	execCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancel = cancel
	e.mu.Unlock()
	defer cancel()

	result := &Result{
		TaskID:    t.ID,
		SessionID: sessionID,
	}

	// Build the command based on the CLI tool
	args := buildCLIArgs(cliTool, t, handoffContent)
	if len(args) == 0 {
		result.Status = "failed"
		result.Error = fmt.Errorf("unsupported CLI tool: %s", cliTool)
		e.log.Error(agentID[:8], "unsupported CLI tool: %s", cliTool)
		return result
	}

	cmd := exec.CommandContext(execCtx, args[0], args[1:]...)

	// Set working directory
	if worktreePath != "" {
		cmd.Dir = worktreePath
	}

	// Set environment
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ARIA_TASK_ID=%s", t.ID),
		fmt.Sprintf("ARIA_SESSION_ID=%s", sessionID),
		fmt.Sprintf("ARIA_AGENT_ID=%s", agentID),
		fmt.Sprintf("ARIA_ROLE=%s", t.Role),
	)

	// Create pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Errorf("failed to create stdout pipe: %w", err)
		return result
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Errorf("failed to create stderr pipe: %w", err)
		return result
	}

	// Pipe handoff content to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Errorf("failed to create stdin pipe: %w", err)
		return result
	}

	e.mu.Lock()
	e.cmd = cmd
	e.mu.Unlock()

	// Start the process
	if err := cmd.Start(); err != nil {
		result.Status = "failed"
		result.Error = fmt.Errorf("failed to start %s: %w", cliTool, err)
		e.log.Error(agentID[:8], "failed to start %s: %v", cliTool, err)
		return result
	}

	// Update agent PID
	if cmd.Process != nil {
		queries.UpdateAgentPID(e.db, agentID, cmd.Process.Pid)
	}

	e.log.Info(agentID[:8], "launched %s (pid: %d) for task %s", cliTool, cmd.Process.Pid, t.ID)

	// Write handoff to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, handoffContent)
	}()

	// Capture output
	var outputBuf strings.Builder
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		e.streamOutput(stdout, agentID, &outputBuf)
	}()
	go func() {
		defer wg.Done()
		e.streamOutput(stderr, agentID, &outputBuf)
	}()

	wg.Wait()

	// Wait for process to finish
	err = cmd.Wait()
	result.Output = outputBuf.String()

	if err != nil {
		result.Status = "failed"
		result.Error = err
		e.log.Error(agentID[:8], "process exited with error: %v", err)
	} else {
		result.Status = "completed"
		e.log.Info(agentID[:8], "process completed for task %s", t.ID)
	}

	// Try to parse the output for a score/report
	if report, parseErr := handoff.ParseReport(result.Output); parseErr == nil {
		if report.Score != "" {
			fmt.Sscanf(report.Score, "%d", &result.Score)
		}
		if report.Body != "" {
			result.Feedback = report.Body
		}
	}

	return result
}

// streamOutput reads from a reader line by line and logs it.
func (e *Executor) streamOutput(r io.Reader, agentID string, buf *strings.Builder) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		e.log.Debug(agentID[:8], "%s", line)
	}
}

// writeHandoff writes the handoff envelope to disk.
func (e *Executor) writeHandoff(t *task.Task, content string, worktreePath string) string {
	dir := ".aria/handoffs"
	if worktreePath != "" {
		dir = filepath.Join(worktreePath, ".aria", "handoffs")
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.log.Warn("system", "failed to create handoff dir: %v", err)
		return ""
	}

	filename := fmt.Sprintf("%s_%s_handoff.md", t.ID, time.Now().Format("20060102_150405"))
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.log.Warn("system", "failed to write handoff: %v", err)
		return ""
	}

	return path
}

// buildCLIArgs constructs the command-line arguments for the given CLI tool.
func buildCLIArgs(cliTool string, t *task.Task, handoffContent string) []string {
	systemPrompt := roles.SystemPrompt(roles.Role(t.Role))

	switch cliTool {
	case "claude-code":
		// Claude Code: claude --print --system-prompt "..." "task description"
		return []string{
			"claude",
			"--print",
			"--system-prompt", systemPrompt,
			fmt.Sprintf("Execute the following task:\n\n%s", handoffContent),
		}
	case "codex":
		// Codex: codex --approval-mode full-auto "task description"
		return []string{
			"codex",
			"--approval-mode", "full-auto",
			fmt.Sprintf("Execute the following task:\n\n%s", handoffContent),
		}
	case "gemini-cli":
		// Gemini CLI: gemini -p "task description"
		return []string{
			"gemini",
			"-p",
			fmt.Sprintf("%s\n\n%s", systemPrompt, handoffContent),
		}
	case "crush":
		// Crush: crush "task description"
		return []string{
			"crush",
			fmt.Sprintf("%s\n\n%s", systemPrompt, handoffContent),
		}
	case "kiro":
		return []string{
			"kiro",
			"--task", fmt.Sprintf("%s\n\n%s", systemPrompt, handoffContent),
		}
	case "droids":
		return []string{
			"droids",
			"run",
			"--prompt", fmt.Sprintf("%s\n\n%s", systemPrompt, handoffContent),
		}
	default:
		// Generic fallback: try to run the tool with the handoff as argument
		return []string{
			cliTool,
			fmt.Sprintf("%s\n\n%s", systemPrompt, handoffContent),
		}
	}
}
