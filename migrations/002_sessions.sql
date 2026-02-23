-- migrations/002_sessions.sql
-- Additional session tracking indexes

CREATE INDEX IF NOT EXISTS idx_sessions_agent  ON sessions(agent_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_failures_task   ON failures(task_id);
