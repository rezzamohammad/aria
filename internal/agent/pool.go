package agent

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/aria-cli/aria/internal/agent/roles"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/google/uuid"
)

// AgentProcess represents a running agent process.
type AgentProcess struct {
	ID      string
	Role    roles.Role
	CLITool string
	Cmd     *exec.Cmd
	Done    chan struct{}
}

// Pool manages the lifecycle of agent processes.
type Pool struct {
	mu      sync.Mutex
	db      *sql.DB
	router  *Router
	agents  map[string]*AgentProcess
	maxSize int
	stopCh  chan struct{}
}

// NewPool creates a new agent pool.
func NewPool(db *sql.DB, router *Router, maxSize int) *Pool {
	if maxSize <= 0 {
		maxSize = 20
	}
	return &Pool{
		db:      db,
		router:  router,
		agents:  make(map[string]*AgentProcess),
		maxSize: maxSize,
		stopCh:  make(chan struct{}),
	}
}

// SpawnAgent creates and registers a new agent for the given role.
// It does NOT start the actual CLI process yet — that happens when a task is claimed.
func (p *Pool) SpawnAgent(role roles.Role, toolOverride string) (*AgentProcess, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.agents) >= p.maxSize {
		return nil, fmt.Errorf("agent pool at capacity (%d/%d)", len(p.agents), p.maxSize)
	}

	agentID := uuid.New().String()
	cliTool := toolOverride
	if cliTool == "" {
		cliTool = p.router.ResolveToolOrDefault(role)
	}

	// Register in database
	agent := &queries.Agent{
		ID:      agentID,
		Role:    string(role),
		CLITool: cliTool,
		Status:  "idle",
	}
	if err := queries.InsertAgent(p.db, agent); err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	ap := &AgentProcess{
		ID:      agentID,
		Role:    role,
		CLITool: cliTool,
		Done:    make(chan struct{}),
	}

	p.agents[agentID] = ap
	return ap, nil
}

// KillAgent stops and removes an agent.
func (p *Pool) KillAgent(agentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ap, exists := p.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found in pool", agentID)
	}

	// Kill process if running
	if ap.Cmd != nil && ap.Cmd.Process != nil {
		ap.Cmd.Process.Signal(os.Interrupt)
		// Give it a moment then force kill
		go func() {
			time.Sleep(5 * time.Second)
			if ap.Cmd.Process != nil {
				ap.Cmd.Process.Kill()
			}
		}()
	}

	close(ap.Done)
	delete(p.agents, agentID)

	// Update database
	if err := queries.UpdateAgentStatus(p.db, agentID, "offline", "", ""); err != nil {
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// ListAgents returns all agents in the pool.
func (p *Pool) ListAgents() []*AgentProcess {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]*AgentProcess, 0, len(p.agents))
	for _, ap := range p.agents {
		result = append(result, ap)
	}
	return result
}

// GetAgent returns a specific agent from the pool.
func (p *Pool) GetAgent(agentID string) (*AgentProcess, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ap, ok := p.agents[agentID]
	return ap, ok
}

// ActiveCount returns the number of agents currently in the pool.
func (p *Pool) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.agents)
}

// IdleAgents returns agents that are not currently working on a task.
// An agent is idle if it has no running process (Cmd == nil) and its
// DB status is "idle" (not crashed/offline).
func (p *Pool) IdleAgents() []*AgentProcess {
	p.mu.Lock()
	defer p.mu.Unlock()

	var idle []*AgentProcess
	for _, ap := range p.agents {
		if ap.Cmd != nil {
			continue
		}
		// Double-check DB status
		dbAgent, err := queries.GetAgent(p.db, ap.ID)
		if err != nil {
			// If we can't read DB, still consider it idle based on Cmd
			idle = append(idle, ap)
			continue
		}
		if dbAgent.Status == "idle" {
			idle = append(idle, ap)
		}
	}
	return idle
}

// Stop shuts down all agents in the pool.
func (p *Pool) Stop() {
	close(p.stopCh)

	p.mu.Lock()
	ids := make([]string, 0, len(p.agents))
	for id := range p.agents {
		ids = append(ids, id)
	}
	p.mu.Unlock()

	for _, id := range ids {
		p.KillAgent(id)
	}
}

// Heartbeat updates the heartbeat timestamp for all active agents.
func (p *Pool) Heartbeat() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id := range p.agents {
		queries.UpdateAgentHeartbeat(p.db, id)
	}
}
