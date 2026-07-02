package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestMigration033WithProductionDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prod-dsn.db")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		t.Fatal(err)
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(db, "migrations", 32); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	// Restore pre-033 accounts schema (simulate production DB at goose v32).
	db2, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db2.Exec(`
		PRAGMA foreign_keys=OFF;
		DROP TABLE IF EXISTS accounts;
		CREATE TABLE accounts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			type TEXT NOT NULL CHECK (type IN ('cash', 'bank')),
			bank_id TEXT REFERENCES banks(id),
			initial_balance INTEGER NOT NULL DEFAULT 0,
			current_balance INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance)
		SELECT 'acc-1', id, 'Cash', 'cash', 0, 0 FROM users LIMIT 1;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		t.Fatal(err)
	}
	if err := db2.Close(); err != nil {
		t.Fatal(err)
	}

	// Re-open like production — runs pending 033+
	mgr2, err := NewManager(path)
	if err != nil {
		t.Fatalf("re-open migrate 033: %v", err)
	}
	t.Cleanup(func() { _ = mgr2.Close() })

	var ddl string
	if err := mgr2.DB().QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='accounts'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ddl, "credit_card") {
		t.Fatalf("expected credit_card in schema: %s", ddl)
	}
}
