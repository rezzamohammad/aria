package queries

import (
	"database/sql"
	"time"
)

// Session represents a row in the sessions table.
type Session struct {
	ID              string
	TaskID          string
	AgentID         string
	CLITool         string
	WorktreePath    sql.NullString
	Status          string
	ContextSnapshot sql.NullString
	ContextTokens   int
	ContextPct      float64
	LogPath         sql.NullString
	StartedAt       string
	EndedAt         sql.NullString
}

// InsertSession creates a new session record.
func InsertSession(db *sql.DB, s *Session) error {
	_, err := db.Exec(`
		INSERT INTO sessions (id, task_id, agent_id, cli_tool, worktree_path, status,
			context_snapshot, context_tokens, context_pct, log_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TaskID, s.AgentID, s.CLITool, s.WorktreePath, s.Status,
		s.ContextSnapshot, s.ContextTokens, s.ContextPct, s.LogPath,
	)
	return err
}

// GetSession retrieves a session by ID.
func GetSession(db *sql.DB, id string) (*Session, error) {
	s := &Session{}
	err := db.QueryRow(`
		SELECT id, task_id, agent_id, cli_tool, worktree_path, status,
			context_snapshot, context_tokens, context_pct, log_path,
			started_at, ended_at
		FROM sessions WHERE id = ?`, id).Scan(
		&s.ID, &s.TaskID, &s.AgentID, &s.CLITool, &s.WorktreePath, &s.Status,
		&s.ContextSnapshot, &s.ContextTokens, &s.ContextPct, &s.LogPath,
		&s.StartedAt, &s.EndedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// ListSessionsByTask returns all sessions for a given task.
func ListSessionsByTask(db *sql.DB, taskID string) ([]*Session, error) {
	rows, err := db.Query(`
		SELECT id, task_id, agent_id, cli_tool, worktree_path, status,
			context_snapshot, context_tokens, context_pct, log_path,
			started_at, ended_at
		FROM sessions WHERE task_id = ? ORDER BY started_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		s := &Session{}
		if err := rows.Scan(
			&s.ID, &s.TaskID, &s.AgentID, &s.CLITool, &s.WorktreePath, &s.Status,
			&s.ContextSnapshot, &s.ContextTokens, &s.ContextPct, &s.LogPath,
			&s.StartedAt, &s.EndedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// EndSession marks a session as completed or failed.
func EndSession(db *sql.DB, id string, status string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec("UPDATE sessions SET status = ?, ended_at = ? WHERE id = ?", status, now, id)
	return err
}

// UpdateSessionContext updates the context usage for a session.
func UpdateSessionContext(db *sql.DB, id string, tokens int, pct float64) error {
	_, err := db.Exec("UPDATE sessions SET context_tokens = ?, context_pct = ? WHERE id = ?", tokens, pct, id)
	return err
}
