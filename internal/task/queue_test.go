package task

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	// Create schema
	schema := `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			epic_id TEXT,
			title TEXT NOT NULL,
			description TEXT,
			role TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER NOT NULL DEFAULT 50,
			dependencies TEXT,
			worktree_path TEXT,
			session_id TEXT,
			agent_id TEXT,
			cli_tool TEXT,
			context_refs TEXT,
			score INTEGER,
			score_breakdown TEXT,
			recovery_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			failure_reason TEXT,
			handoff_path TEXT,
			report_path TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			claimed_at TEXT,
			started_at TEXT,
			completed_at TEXT,
			estimated_hours REAL,
			actual_hours REAL
		);
		CREATE INDEX idx_tasks_status_priority ON tasks(status, priority);
		CREATE INDEX idx_tasks_role_status ON tasks(role, status);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(dir)
	}

	return db, cleanup
}

func insertTestTask(t *testing.T, db *sql.DB, id, title, role string, priority int, deps string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO tasks (id, title, role, status, priority, dependencies, max_retries)
		VALUES (?, ?, ?, 'pending', ?, ?, 3)
	`, id, title, role, priority, deps)
	if err != nil {
		t.Fatalf("failed to insert test task: %v", err)
	}
}

func TestClaimNextTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "First task", "worker", 10, "[]")
	insertTestTask(t, db, "T-002", "Second task", "worker", 50, "[]")

	// Claim should return highest priority (lowest number)
	task, err := ClaimNextTask(db, "agent-1", "worker")
	if err != nil {
		t.Fatalf("ClaimNextTask error: %v", err)
	}
	if task == nil {
		t.Fatal("expected a task, got nil")
	}
	if task.ID != "T-001" {
		t.Errorf("expected T-001, got %s", task.ID)
	}
	if task.Status != StatusClaimed {
		t.Errorf("expected status claimed, got %s", task.Status)
	}
}

func TestClaimNextTaskRoleFilter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Worker task", "worker", 10, "[]")
	insertTestTask(t, db, "T-002", "Verifier task", "verifier", 10, "[]")

	// Should only get verifier tasks
	task, err := ClaimNextTask(db, "agent-1", "verifier")
	if err != nil {
		t.Fatalf("ClaimNextTask error: %v", err)
	}
	if task == nil {
		t.Fatal("expected a task, got nil")
	}
	if task.ID != "T-002" {
		t.Errorf("expected T-002, got %s", task.ID)
	}
}

func TestClaimNextTaskNoAvailable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// No tasks at all
	task, err := ClaimNextTask(db, "agent-1", "worker")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task != nil {
		t.Errorf("expected nil task, got %+v", task)
	}
}

func TestClaimNextTaskDependencies(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Dep task", "worker", 10, "[]")
	insertTestTask(t, db, "T-002", "Blocked task", "worker", 5, `["T-001"]`)

	// T-002 has higher priority but is blocked by T-001
	task, err := ClaimNextTask(db, "agent-1", "worker")
	if err != nil {
		t.Fatalf("ClaimNextTask error: %v", err)
	}
	if task == nil {
		t.Fatal("expected a task")
	}
	if task.ID != "T-001" {
		t.Errorf("expected T-001 (unblocked), got %s", task.ID)
	}
}

func TestReleaseTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Test", "worker", 10, "[]")
	_, _ = ClaimNextTask(db, "agent-1", "worker")

	err := ReleaseTask(db, "T-001")
	if err != nil {
		t.Fatalf("ReleaseTask error: %v", err)
	}

	// Should be claimable again
	task, err := ClaimNextTask(db, "agent-2", "worker")
	if err != nil {
		t.Fatalf("ClaimNextTask error: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to be available after release")
	}
	if task.ID != "T-001" {
		t.Errorf("expected T-001, got %s", task.ID)
	}
}

func TestCompleteTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Test", "worker", 10, "[]")

	err := CompleteTask(db, "T-001", 95, `{"technical":48,"content":33,"aesthetic":14}`)
	if err != nil {
		t.Fatalf("CompleteTask error: %v", err)
	}

	// Verify status
	var status string
	var score int
	db.QueryRow("SELECT status, score FROM tasks WHERE id = ?", "T-001").Scan(&status, &score)
	if status != "done" {
		t.Errorf("expected status 'done', got %q", status)
	}
	if score != 95 {
		t.Errorf("expected score 95, got %d", score)
	}
}

func TestFailTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Test", "worker", 10, "[]")

	// First failure should go back to pending (recovery_count=1, max_retries=3)
	err := FailTask(db, "T-001", "compile error")
	if err != nil {
		t.Fatalf("FailTask error: %v", err)
	}

	var status string
	var count int
	db.QueryRow("SELECT status, recovery_count FROM tasks WHERE id = ?", "T-001").Scan(&status, &count)
	if status != "pending" {
		t.Errorf("expected status 'pending' after first failure, got %q", status)
	}
	if count != 1 {
		t.Errorf("expected recovery_count 1, got %d", count)
	}
}

func TestFailTaskMaxRetries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Test", "worker", 10, "[]")

	// Fail 3 times (max_retries=3)
	FailTask(db, "T-001", "error 1")
	FailTask(db, "T-001", "error 2")
	FailTask(db, "T-001", "error 3")

	var status string
	db.QueryRow("SELECT status FROM tasks WHERE id = ?", "T-001").Scan(&status)
	if status != "failed" {
		t.Errorf("expected status 'failed' after max retries, got %q", status)
	}
}

func TestBlockTask(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Test", "worker", 10, "[]")

	err := BlockTask(db, "T-001", "waiting for T-000")
	if err != nil {
		t.Fatalf("BlockTask error: %v", err)
	}

	var status string
	db.QueryRow("SELECT status FROM tasks WHERE id = ?", "T-001").Scan(&status)
	if status != "blocked" {
		t.Errorf("expected status 'blocked', got %q", status)
	}
}

func TestGetQueueStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	insertTestTask(t, db, "T-001", "Pending1", "worker", 10, "[]")
	insertTestTask(t, db, "T-002", "Pending2", "worker", 20, "[]")
	insertTestTask(t, db, "T-003", "Running", "worker", 30, "[]")
	db.Exec("UPDATE tasks SET status = 'running' WHERE id = 'T-003'")
	insertTestTask(t, db, "T-004", "Done", "worker", 40, "[]")
	db.Exec("UPDATE tasks SET status = 'done' WHERE id = 'T-004'")

	stats, err := GetQueueStats(db)
	if err != nil {
		t.Fatalf("GetQueueStats error: %v", err)
	}

	if stats.Total != 4 {
		t.Errorf("expected total 4, got %d", stats.Total)
	}
	if stats.Pending != 2 {
		t.Errorf("expected pending 2, got %d", stats.Pending)
	}
	if stats.Running != 1 {
		t.Errorf("expected running 1, got %d", stats.Running)
	}
	if stats.Done != 1 {
		t.Errorf("expected done 1, got %d", stats.Done)
	}
}
