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

// v1.9.2 owns goose version 56 (merchant_aliases). Receipt migrations are 057–061.
// Re-applying them must succeed when those objects already exist, and 062 must
// recreate merchant_aliases if version 56 was recorded without that table.
func TestReceiptMigrationsAfterMerchantAliasVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	assertMigrationChain(t, mgr.DB())

	if _, err := mgr.DB().Exec(`DROP TABLE merchant_aliases`); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.DB().Exec(`DELETE FROM goose_db_version WHERE version_id >= 57`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(mgr.DB()); err != nil {
		t.Fatal(err)
	}
	assertMigrationChain(t, mgr.DB())
}

func assertMigrationChain(t *testing.T, sqlDB *sql.DB) {
	t.Helper()
	var version int
	if err := sqlDB.QueryRow(`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 62 {
		t.Fatalf("goose version %d, want 62", version)
	}
	for _, table := range []string{
		"merchant_aliases",
		"receipt_scan_jobs",
		"stores",
		"receipt_imports",
		"receipt_fiscal_previews",
		"receipt_import_item_categories",
	} {
		var n int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("table %s count %d", table, n)
		}
	}
	var cols int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('system_settings') WHERE name = 'receipt_api_token_encrypted'`).Scan(&cols); err != nil {
		t.Fatal(err)
	}
	if cols != 1 {
		t.Fatalf("receipt_api_token_encrypted columns %d", cols)
	}
	var idx int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_tx_store'`).Scan(&idx); err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("idx_tx_store count %d", idx)
	}
}
