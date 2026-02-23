package queries

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Task represents a row in the tasks table.
type Task struct {
	ID             string
	EpicID         sql.NullString
	Title          string
	Description    sql.NullString
	Role           string
	Status         string
	Priority       int
	Dependencies   []string
	WorktreePath   sql.NullString
	SessionID      sql.NullString
	AgentID        sql.NullString
	CLITool        sql.NullString
	ContextRefs    []string
	Score          sql.NullInt64
	ScoreBreakdown sql.NullString
	RecoveryCount  int
	MaxRetries     int
	FailureReason  sql.NullString
	HandoffPath    sql.NullString
	ReportPath     sql.NullString
	CreatedAt      string
	ClaimedAt      sql.NullString
	StartedAt      sql.NullString
	CompletedAt    sql.NullString
	EstimatedHours sql.NullFloat64
	ActualHours    sql.NullFloat64
}

// InsertTask inserts a new task into the database.
func InsertTask(db *sql.DB, t *Task) error {
	depsJSON, _ := json.Marshal(t.Dependencies)
	refsJSON, _ := json.Marshal(t.ContextRefs)

	_, err := db.Exec(`
		INSERT INTO tasks (id, epic_id, title, description, role, status, priority, dependencies,
			worktree_path, session_id, agent_id, cli_tool, context_refs, score, score_breakdown,
			recovery_count, max_retries, failure_reason, handoff_path, report_path,
			estimated_hours, actual_hours)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.EpicID, t.Title, t.Description, t.Role, t.Status, t.Priority,
		string(depsJSON), t.WorktreePath, t.SessionID, t.AgentID, t.CLITool,
		string(refsJSON), t.Score, t.ScoreBreakdown,
		t.RecoveryCount, t.MaxRetries, t.FailureReason, t.HandoffPath, t.ReportPath,
		t.EstimatedHours, t.ActualHours,
	)
	return err
}

// GetTask retrieves a single task by ID.
func GetTask(db *sql.DB, id string) (*Task, error) {
	t := &Task{}
	var depsStr, refsStr sql.NullString

	err := db.QueryRow(`
		SELECT id, epic_id, title, description, role, status, priority, dependencies,
			worktree_path, session_id, agent_id, cli_tool, context_refs, score, score_breakdown,
			recovery_count, max_retries, failure_reason, handoff_path, report_path,
			created_at, claimed_at, started_at, completed_at, estimated_hours, actual_hours
		FROM tasks WHERE id = ?`, id).Scan(
		&t.ID, &t.EpicID, &t.Title, &t.Description, &t.Role, &t.Status, &t.Priority,
		&depsStr, &t.WorktreePath, &t.SessionID, &t.AgentID, &t.CLITool,
		&refsStr, &t.Score, &t.ScoreBreakdown,
		&t.RecoveryCount, &t.MaxRetries, &t.FailureReason, &t.HandoffPath, &t.ReportPath,
		&t.CreatedAt, &t.ClaimedAt, &t.StartedAt, &t.CompletedAt,
		&t.EstimatedHours, &t.ActualHours,
	)
	if err != nil {
		return nil, err
	}

	if depsStr.Valid {
		json.Unmarshal([]byte(depsStr.String), &t.Dependencies)
	}
	if refsStr.Valid {
		json.Unmarshal([]byte(refsStr.String), &t.ContextRefs)
	}

	return t, nil
}

// ListTasks returns all tasks, optionally filtered by status.
func ListTasks(db *sql.DB, statusFilter string) ([]*Task, error) {
	query := `
		SELECT id, epic_id, title, description, role, status, priority, dependencies,
			worktree_path, session_id, agent_id, cli_tool, context_refs, score, score_breakdown,
			recovery_count, max_retries, failure_reason, handoff_path, report_path,
			created_at, claimed_at, started_at, completed_at, estimated_hours, actual_hours
		FROM tasks`

	var rows *sql.Rows
	var err error

	if statusFilter != "" {
		query += " WHERE status = ? ORDER BY priority ASC, created_at ASC"
		rows, err = db.Query(query, statusFilter)
	} else {
		query += " ORDER BY priority ASC, created_at ASC"
		rows, err = db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		t := &Task{}
		var depsStr, refsStr sql.NullString

		if err := rows.Scan(
			&t.ID, &t.EpicID, &t.Title, &t.Description, &t.Role, &t.Status, &t.Priority,
			&depsStr, &t.WorktreePath, &t.SessionID, &t.AgentID, &t.CLITool,
			&refsStr, &t.Score, &t.ScoreBreakdown,
			&t.RecoveryCount, &t.MaxRetries, &t.FailureReason, &t.HandoffPath, &t.ReportPath,
			&t.CreatedAt, &t.ClaimedAt, &t.StartedAt, &t.CompletedAt,
			&t.EstimatedHours, &t.ActualHours,
		); err != nil {
			return nil, err
		}

		if depsStr.Valid {
			json.Unmarshal([]byte(depsStr.String), &t.Dependencies)
		}
		if refsStr.Valid {
			json.Unmarshal([]byte(refsStr.String), &t.ContextRefs)
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

// UpdateTaskStatus updates the status of a task and related timestamp.
func UpdateTaskStatus(db *sql.DB, id string, status string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	var query string
	switch status {
	case "claimed":
		query = fmt.Sprintf("UPDATE tasks SET status = '%s', claimed_at = '%s' WHERE id = ?", status, now)
	case "running":
		query = fmt.Sprintf("UPDATE tasks SET status = '%s', started_at = '%s' WHERE id = ?", status, now)
	case "done":
		query = fmt.Sprintf("UPDATE tasks SET status = '%s', completed_at = '%s' WHERE id = ?", status, now)
	default:
		query = fmt.Sprintf("UPDATE tasks SET status = '%s' WHERE id = ?", status)
	}

	result, err := db.Exec(query, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("task %s not found", id)
	}
	return nil
}

// UpdateTaskScore sets the verification score on a task.
func UpdateTaskScore(db *sql.DB, id string, score int, breakdown string) error {
	_, err := db.Exec("UPDATE tasks SET score = ?, score_breakdown = ? WHERE id = ?", score, breakdown, id)
	return err
}

// IncrementRecoveryCount increments the recovery count for a task.
func IncrementRecoveryCount(db *sql.DB, id string, reason string) error {
	_, err := db.Exec(`
		UPDATE tasks SET
			recovery_count = recovery_count + 1,
			failure_reason = ?,
			status = CASE WHEN recovery_count + 1 >= max_retries THEN 'failed' ELSE 'pending' END
		WHERE id = ?`, reason, id)
	return err
}

// CountTasksByStatus returns counts per status.
func CountTasksByStatus(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
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
