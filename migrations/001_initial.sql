-- migrations/001_initial.sql

CREATE TABLE IF NOT EXISTS tasks (
    id              TEXT PRIMARY KEY,
    epic_id         TEXT,
    title           TEXT NOT NULL,
    description     TEXT,
    role            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    priority        INTEGER NOT NULL DEFAULT 50,
    dependencies    TEXT,
    worktree_path   TEXT,
    session_id      TEXT,
    agent_id        TEXT,
    cli_tool        TEXT,
    context_refs    TEXT,
    score           INTEGER,
    score_breakdown TEXT,
    recovery_count  INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    failure_reason  TEXT,
    handoff_path    TEXT,
    report_path     TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    claimed_at      TEXT,
    started_at      TEXT,
    completed_at    TEXT,
    estimated_hours REAL,
    actual_hours    REAL
);

CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    task_id         TEXT NOT NULL,
    agent_id        TEXT NOT NULL,
    cli_tool        TEXT NOT NULL,
    worktree_path   TEXT,
    status          TEXT NOT NULL DEFAULT 'active',
    context_snapshot TEXT,
    context_tokens  INTEGER DEFAULT 0,
    context_pct     REAL DEFAULT 0.0,
    log_path        TEXT,
    started_at      TEXT NOT NULL DEFAULT (datetime('now')),
    ended_at        TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS agents (
    id              TEXT PRIMARY KEY,
    role            TEXT NOT NULL,
    cli_tool        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'idle',
    current_task_id TEXT,
    current_session TEXT,
    worktree_path   TEXT,
    pid             INTEGER,
    last_heartbeat  TEXT,
    total_tasks     INTEGER DEFAULT 0,
    avg_score       REAL DEFAULT 0.0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (current_task_id) REFERENCES tasks(id)
);

CREATE TABLE IF NOT EXISTS failures (
    id              TEXT PRIMARY KEY,
    task_id         TEXT NOT NULL,
    session_id      TEXT,
    failure_type    TEXT NOT NULL,
    description     TEXT,
    recovery_action TEXT,
    resolved        INTEGER DEFAULT 0,
    occurred_at     TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority);
CREATE INDEX IF NOT EXISTS idx_tasks_role_status     ON tasks(role, status);
CREATE INDEX IF NOT EXISTS idx_sessions_task         ON sessions(task_id);
CREATE INDEX IF NOT EXISTS idx_agents_status         ON agents(status);
