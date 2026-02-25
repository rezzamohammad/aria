package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/aria-cli/aria/internal/config"
	"github.com/aria-cli/aria/internal/db/queries"
	"github.com/aria-cli/aria/internal/logger"
	"github.com/aria-cli/aria/internal/task"
	"github.com/aria-cli/aria/internal/tui/styles"
	"github.com/aria-cli/aria/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Panel represents which panel is currently focused.
type Panel int

const (
	PanelDashboard Panel = iota
	PanelTaskDetail
	PanelAgents
	PanelLog
)

// logMsg is sent when a new log entry arrives from the logger.
type logMsg logger.Entry

// Model is the root Bubble Tea model for the ARIA TUI.
type Model struct {
	db     *sql.DB
	cfg    *config.Config
	log    *logger.Logger
	width  int
	height int

	// State
	activePanel   Panel
	tasks         []*task.Task
	agents        []*queries.Agent
	sessions      map[string]*queries.Session
	logEntries    []views.LogEntry
	selectedTask  int
	selectedAgent int
	queueStats    *task.QueueStats

	// Views
	dashboardView  *views.DashboardView
	taskDetailView *views.TaskDetailView
	agentPanelView *views.AgentPanelView
	logPanelView   *views.LogPanelView

	// Refresh
	lastRefresh time.Time
	quitting    bool

	// Program reference for sending messages from logger
	program *tea.Program
}

// tickMsg is sent periodically to refresh data.
type tickMsg time.Time

// NewModel creates the root TUI model.
func NewModel(db *sql.DB, cfg *config.Config, log *logger.Logger) Model {
	return Model{
		db:          db,
		cfg:         cfg,
		log:         log,
		activePanel: PanelDashboard,
		sessions:    make(map[string]*queries.Session),
		logEntries:  []views.LogEntry{},
	}
}

// SetProgram sets the tea.Program reference for async message sending.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p

	// Subscribe to logger events
	if m.log != nil {
		m.log.Subscribe(func(entry logger.Entry) {
			if p != nil {
				p.Send(logMsg(entry))
			}
		})
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("ARIA — Autonomous Routing & Intelligence Agent"),
		tickCmd(),
	)
}

// tickCmd returns a command that ticks every 2 seconds.
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Update handles messages and user input.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dashboardView = views.NewDashboardView(m.width, m.height/2)
		m.taskDetailView = views.NewTaskDetailView(m.width/2, m.height/2)
		m.agentPanelView = views.NewAgentPanelView(m.width)
		m.logPanelView = views.NewLogPanelView(m.width, 6)
		m.refreshData()
		return m, nil

	case tickMsg:
		m.refreshData()
		return m, tickCmd()

	case logMsg:
		entry := logger.Entry(msg)
		m.logEntries = append(m.logEntries, views.LogEntry{
			Timestamp: entry.Timestamp,
			Level:     string(entry.Level),
			Source:    entry.Source,
			Message:   entry.Message,
		})
		// Keep max 100 entries
		if len(m.logEntries) > 100 {
			m.logEntries = m.logEntries[len(m.logEntries)-100:]
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "tab":
			m.activePanel = (m.activePanel + 1) % 4
			return m, nil

		case "shift+tab":
			m.activePanel = (m.activePanel - 1 + 4) % 4
			return m, nil

		case "j", "down":
			if m.activePanel == PanelDashboard && len(m.tasks) > 0 {
				m.selectedTask = (m.selectedTask + 1) % len(m.tasks)
			}
			return m, nil

		case "k", "up":
			if m.activePanel == PanelDashboard && len(m.tasks) > 0 {
				m.selectedTask = (m.selectedTask - 1 + len(m.tasks)) % len(m.tasks)
			}
			return m, nil

		case "enter":
			if m.activePanel == PanelDashboard {
				m.activePanel = PanelTaskDetail
			}
			return m, nil

		case "esc":
			m.activePanel = PanelDashboard
			return m, nil

		case "r":
			m.refreshData()
			m.addLog("info", "system", "Manual refresh triggered")
			return m, nil
		}
	}

	return m, nil
}

// View renders the full TUI.
func (m Model) View() string {
	if m.quitting {
		return "ARIA shutting down...\n"
	}

	if m.width == 0 {
		return "Loading ARIA..."
	}

	var sections []string

	// Dashboard / kanban board
	if m.dashboardView != nil {
		projectName := m.cfg.Project.Name
		dashboard := m.dashboardView.Render(m.tasks, projectName)
		if m.activePanel == PanelDashboard {
			dashboard = styles.ActivePanelStyle.Render(dashboard)
		}
		sections = append(sections, dashboard)
	}

	// Task detail (shown when selected)
	if m.activePanel == PanelTaskDetail && m.taskDetailView != nil {
		var selectedTask *task.Task
		if m.selectedTask >= 0 && m.selectedTask < len(m.tasks) {
			selectedTask = m.tasks[m.selectedTask]
		}
		detail := m.taskDetailView.Render(selectedTask)
		sections = append(sections, detail)
	}

	// Agent panel
	if m.agentPanelView != nil {
		agentPanel := m.agentPanelView.Render(m.agents, m.sessions)
		if m.activePanel == PanelAgents {
			agentPanel = styles.ActivePanelStyle.Render(agentPanel)
		}
		sections = append(sections, agentPanel)
	}

	// Log panel
	if m.logPanelView != nil {
		logPanel := m.logPanelView.Render(m.logEntries)
		if m.activePanel == PanelLog {
			logPanel = styles.ActivePanelStyle.Render(logPanel)
		}
		sections = append(sections, logPanel)
	}

	// Help bar
	help := styles.HelpStyle.Render("[tab] switch panel  [enter] task detail  [r] retry  [q] quit")
	sections = append(sections, help)

	// Stats bar
	if m.queueStats != nil {
		stats := styles.StatusBarStyle.Render(fmt.Sprintf(
			"Total: %d | Pending: %d | Running: %d | Done: %d | Failed: %d | Blocked: %d",
			m.queueStats.Total, m.queueStats.Pending, m.queueStats.Running,
			m.queueStats.Done, m.queueStats.Failed, m.queueStats.Blocked))
		sections = append(sections, stats)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// refreshData reloads all data from the database.
func (m *Model) refreshData() {
	m.lastRefresh = time.Now()

	// Load tasks
	if tasks, err := loadTasks(m.db); err == nil {
		m.tasks = tasks
	}

	// Load agents
	if agents, err := queries.ListAgents(m.db, ""); err == nil {
		m.agents = agents
	}

	// Load queue stats
	if stats, err := task.GetQueueStats(m.db); err == nil {
		m.queueStats = stats
	}

	// Load active sessions for agents
	for _, a := range m.agents {
		if a.CurrentSession.Valid {
			if sess, err := queries.GetSession(m.db, a.CurrentSession.String); err == nil {
				m.sessions[a.CurrentSession.String] = sess
			}
		}
	}
}

// addLog appends a log entry.
func (m *Model) addLog(level, source, message string) {
	m.logEntries = append(m.logEntries, views.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
	})

	// Keep max 100 entries
	if len(m.logEntries) > 100 {
		m.logEntries = m.logEntries[len(m.logEntries)-100:]
	}
}

// loadTasks retrieves all tasks and converts them to task.Task objects.
func loadTasks(db *sql.DB) ([]*task.Task, error) {
	rows, err := db.Query(`
		SELECT id, epic_id, title, description, role, status, priority, dependencies,
			worktree_path, session_id, agent_id, cli_tool, context_refs, score, score_breakdown,
			recovery_count, max_retries, failure_reason, handoff_path, report_path,
			created_at, claimed_at, started_at, completed_at, estimated_hours, actual_hours
		FROM tasks ORDER BY priority ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*task.Task
	for rows.Next() {
		t := &task.Task{}
		var epicID, desc, wt, sid, aid, tool sql.NullString
		var ctxRefs, scoreBD, failR, handoff, report, deps sql.NullString
		var score sql.NullInt64
		var claimedAt, startedAt, completedAt sql.NullString
		var estHours, actHours sql.NullFloat64

		if err := rows.Scan(
			&t.ID, &epicID, &t.Title, &desc, &t.Role, &t.Status, &t.Priority,
			&deps, &wt, &sid, &aid, &tool,
			&ctxRefs, &score, &scoreBD,
			&t.RecoveryCount, &t.MaxRetries, &failR, &handoff, &report,
			&t.CreatedAt, &claimedAt, &startedAt, &completedAt,
			&estHours, &actHours,
		); err != nil {
			return nil, err
		}

		if epicID.Valid {
			t.EpicID = epicID.String
		}
		if desc.Valid {
			t.Description = desc.String
		}
		if wt.Valid {
			t.WorktreePath = wt.String
		}
		if sid.Valid {
			t.SessionID = sid.String
		}
		if aid.Valid {
			t.AgentID = aid.String
		}
		if tool.Valid {
			t.CLITool = tool.String
		}
		if score.Valid {
			t.Score = int(score.Int64)
		}
		if scoreBD.Valid {
			t.ScoreBreakdown = scoreBD.String
		}
		if failR.Valid {
			t.FailureReason = failR.String
		}
		if handoff.Valid {
			t.HandoffPath = handoff.String
		}
		if report.Valid {
			t.ReportPath = report.String
		}
		if claimedAt.Valid {
			t.ClaimedAt = claimedAt.String
		}
		if startedAt.Valid {
			t.StartedAt = startedAt.String
		}
		if completedAt.Valid {
			t.CompletedAt = completedAt.String
		}
		if estHours.Valid {
			t.EstimatedHours = estHours.Float64
		}
		if actHours.Valid {
			t.ActualHours = actHours.Float64
		}

		// Parse JSON arrays
		if deps.Valid && deps.String != "" {
			parseJSONArray(deps.String, &t.Dependencies)
		}
		if ctxRefs.Valid && ctxRefs.String != "" {
			parseJSONArray(ctxRefs.String, &t.ContextRefs)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

func parseJSONArray(s string, target *[]string) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" || s == "[]" {
		return
	}
	// Simple JSON array parser for string arrays
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"")
		if p != "" {
			*target = append(*target, p)
		}
	}
}
