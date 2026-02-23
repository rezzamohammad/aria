package task

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ClaimNextTask atomically claims the highest-priority pending task for a given role
// where all dependencies are satisfied (status = 'done').
// Returns nil, nil if no task is available.
func ClaimNextTask(db *sql.DB, agentID string, role string) (*Task, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find highest priority pending task for this role where all deps are done
	row := tx.QueryRow(`
		SELECT t.id FROM tasks t
		WHERE t.role = ?
		  AND t.status = 'pending'
		  AND t.recovery_count < t.max_retries
		  AND NOT EXISTS (
		      SELECT 1 FROM tasks dep
		      WHERE dep.id IN (SELECT value FROM json_each(t.dependencies))
		        AND dep.status != 'done'
		  )
		ORDER BY t.priority ASC, t.created_at ASC
		LIMIT 1
	`, role)

	var taskID string
	if err := row.Scan(&taskID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No task available
		}
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}

	// Atomic claim
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	result, err := tx.Exec(`
		UPDATE tasks SET
			status     = 'claimed',
			agent_id   = ?,
			claimed_at = ?
		WHERE id = ? AND status = 'pending'
	`, agentID, now, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim task: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, nil // Race condition: someone else claimed it
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit claim: %w", err)
	}

	return getTask(db, taskID)
}

// ReleaseTask returns a claimed task back to pending (e.g., agent crashed).
func ReleaseTask(db *sql.DB, taskID string) error {
	_, err := db.Exec(`
		UPDATE tasks SET
			status   = 'pending',
			agent_id = NULL,
			claimed_at = NULL
		WHERE id = ? AND status IN ('claimed', 'running')
	`, taskID)
	return err
}

// CompleteTask marks a task as done with an optional score.
func CompleteTask(db *sql.DB, taskID string, score int, scoreBreakdown string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`
		UPDATE tasks SET
			status         = 'done',
			score          = ?,
			score_breakdown = ?,
			completed_at   = ?
		WHERE id = ?
	`, score, scoreBreakdown, now, taskID)
	return err
}

// FailTask marks a task as failed and increments its recovery count.
func FailTask(db *sql.DB, taskID string, reason string) error {
	_, err := db.Exec(`
		UPDATE tasks SET
			recovery_count = recovery_count + 1,
			failure_reason = ?,
			status = CASE
				WHEN recovery_count + 1 >= max_retries THEN 'failed'
				ELSE 'pending'
			END,
			agent_id = NULL
		WHERE id = ?
	`, reason, taskID)
	return err
}

// BlockTask marks a task as blocked.
func BlockTask(db *sql.DB, taskID string, reason string) error {
	_, err := db.Exec(`
		UPDATE tasks SET status = 'blocked', failure_reason = ?
		WHERE id = ?
	`, reason, taskID)
	return err
}

// QueueStats returns summary statistics about the task queue.
type QueueStats struct {
	Total    int
	Pending  int
	Running  int
	Done     int
	Failed   int
	Blocked  int
}

// GetQueueStats calculates current queue statistics.
func GetQueueStats(db *sql.DB) (*QueueStats, error) {
	rows, err := db.Query("SELECT status, COUNT(*) FROM tasks GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &QueueStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.Total += count
		switch Status(status) {
		case StatusPending:
			stats.Pending = count
		case StatusClaimed, StatusRunning, StatusVerifying:
			stats.Running += count
		case StatusDone:
			stats.Done = count
		case StatusFailed:
			stats.Failed = count
		case StatusBlocked:
			stats.Blocked = count
		}
	}
	return stats, nil
}

// getTask is an internal helper to retrieve a full task by ID.
func getTask(db *sql.DB, id string) (*Task, error) {
	t := &Task{}
	var depsStr, epicID, desc, wt, sid, aid, tool sql.NullString
	var ctxRefs, scoreBD, failR, handoff, report sql.NullString
	var score sql.NullInt64
	var claimedAt, startedAt, completedAt sql.NullString
	var estHours, actHours sql.NullFloat64

	err := db.QueryRow(`
		SELECT id, epic_id, title, description, role, status, priority, dependencies,
			worktree_path, session_id, agent_id, cli_tool, context_refs, score, score_breakdown,
			recovery_count, max_retries, failure_reason, handoff_path, report_path,
			created_at, claimed_at, started_at, completed_at, estimated_hours, actual_hours
		FROM tasks WHERE id = ?`, id).Scan(
		&t.ID, &epicID, &t.Title, &desc, &t.Role, &t.Status, &t.Priority,
		&depsStr, &wt, &sid, &aid, &tool,
		&ctxRefs, &score, &scoreBD,
		&t.RecoveryCount, &t.MaxRetries, &failR, &handoff, &report,
		&t.CreatedAt, &claimedAt, &startedAt, &completedAt,
		&estHours, &actHours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", id, err)
	}

	// Map nullable fields
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

	if depsStr.Valid && depsStr.String != "" {
		json.Unmarshal([]byte(depsStr.String), &t.Dependencies)
	}
	if ctxRefs.Valid && ctxRefs.String != "" {
		json.Unmarshal([]byte(ctxRefs.String), &t.ContextRefs)
	}

	return t, nil
}
