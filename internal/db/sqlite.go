package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

// Open opens (or creates) the SQLite database and runs the schema.
func Open(dbPath string) (*sql.DB, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create db directory %s: %w", dir, err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on", dbPath)
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runSchema(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to run schema: %w", err)
	}

	return database, nil
}

// runSchema executes the embedded schema.sql to create all tables and indexes.
// All statements use IF NOT EXISTS so this is safe to run on an existing DB.
func runSchema(database *sql.DB) error {
	if schemaSQL == "" {
		return fmt.Errorf("no schema SQL available")
	}
	if _, err := database.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}
