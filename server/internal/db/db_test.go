package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigration033CreditCard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	var ddl string
	if err := mgr.DB().QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='accounts'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"credit_card", "credit_limit", "payment_account_id"} {
		if !strings.Contains(ddl, part) {
			t.Fatalf("accounts schema missing %q: %s", part, ddl)
		}
	}
}

func TestRecoverInterruptedAccountRebuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.db")
	dsn := "file:" + path + "?_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	if _, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE accounts_new (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			bank_id TEXT,
			initial_balance INTEGER NOT NULL DEFAULT 0,
			current_balance INTEGER NOT NULL DEFAULT 0,
			credit_limit INTEGER,
			payment_account_id TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		INSERT INTO accounts_new (id, user_id, name, type, created_at, updated_at)
		VALUES ('acc-1', 'user-1', 'Test', 'cash', datetime('now'), datetime('now'));
	`); err != nil {
		t.Fatal(err)
	}

	if err := recoverInterruptedAccountRebuild(sqlDB); err != nil {
		t.Fatal(err)
	}

	var name string
	if err := sqlDB.QueryRowContext(ctx, `SELECT name FROM accounts WHERE id='acc-1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Test" {
		t.Fatalf("name %q", name)
	}
	_ = os.Remove(path)
}
