package queries

import (
	"database/sql"
	"time"
)

// Agent represents a row in the agents table.
type Agent struct {
	ID             string
	Role           string
	CLITool        string
	Status         string
	CurrentTaskID  sql.NullString
	CurrentSession sql.NullString
	WorktreePath   sql.NullString
	PID            sql.NullInt64
	LastHeartbeat  sql.NullString
	TotalTasks     int
	AvgScore       float64
	CreatedAt      string
}

// InsertAgent creates a new agent record.
func InsertAgent(db *sql.DB, a *Agent) error {
	_, err := db.Exec(`
		INSERT INTO agents (id, role, cli_tool, status, current_task_id, current_session,
			worktree_path, pid, last_heartbeat, total_tasks, avg_score)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Role, a.CLITool, a.Status, a.CurrentTaskID, a.CurrentSession,
		a.WorktreePath, a.PID, a.LastHeartbeat, a.TotalTasks, a.AvgScore,
	)
	return err
}

// GetAgent retrieves an agent by ID.
func GetAgent(db *sql.DB, id string) (*Agent, error) {
	a := &Agent{}
	err := db.QueryRow(`
		SELECT id, role, cli_tool, status, current_task_id, current_session,
			worktree_path, pid, last_heartbeat, total_tasks, avg_score, created_at
		FROM agents WHERE id = ?`, id).Scan(
		&a.ID, &a.Role, &a.CLITool, &a.Status, &a.CurrentTaskID, &a.CurrentSession,
		&a.WorktreePath, &a.PID, &a.LastHeartbeat, &a.TotalTasks, &a.AvgScore, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ListAgents returns all agents, optionally filtered by status.
func ListAgents(db *sql.DB, statusFilter string) ([]*Agent, error) {
	query := `
		SELECT id, role, cli_tool, status, current_task_id, current_session,
			worktree_path, pid, last_heartbeat, total_tasks, avg_score, created_at
		FROM agents`

	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		query += " WHERE status = ? ORDER BY created_at ASC"
		rows, err = db.Query(query, statusFilter)
	} else {
		query += " ORDER BY created_at ASC"
		rows, err = db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		a := &Agent{}
		if err := rows.Scan(
			&a.ID, &a.Role, &a.CLITool, &a.Status, &a.CurrentTaskID, &a.CurrentSession,
			&a.WorktreePath, &a.PID, &a.LastHeartbeat, &a.TotalTasks, &a.AvgScore, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// UpdateAgentStatus updates agent status and optionally its current task.
func UpdateAgentStatus(db *sql.DB, id string, status string, taskID string, sessionID string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`
		UPDATE agents SET
			status = ?,
			current_task_id = CASE WHEN ? = '' THEN NULL ELSE ? END,
			current_session = CASE WHEN ? = '' THEN NULL ELSE ? END,
			last_heartbeat = ?
		WHERE id = ?`,
		status, taskID, taskID, sessionID, sessionID, now, id)
	return err
}

// UpdateAgentHeartbeat records a heartbeat for an agent.
func UpdateAgentHeartbeat(db *sql.DB, id string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE agents SET last_heartbeat = ? WHERE id = ?", now, id)
	return err
}

// UpdateAgentPID sets the OS process ID for a running agent.
func UpdateAgentPID(db *sql.DB, id string, pid int) error {
	_, err := db.Exec("UPDATE agents SET pid = ? WHERE id = ?", pid, id)
	return err
}

// DeleteAgent removes an agent record.
func DeleteAgent(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM agents WHERE id = ?", id)
	return err
}

// CountAgentsByStatus returns counts per agent status.
func CountAgentsByStatus(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query("SELECT status, COUNT(*) FROM agents GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, nil
}
