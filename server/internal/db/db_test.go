package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
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
	// 063–066 run again after the replayed receipt chain; undo their objects first.
	// 064's index covers budgets.currency, so it must go before the column.
	if _, err := mgr.DB().Exec(`DROP INDEX idx_budgets_active_unique`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"household_invites", "household_members", "households"} {
		if _, err := mgr.DB().Exec(`DROP TABLE ` + table); err != nil {
			t.Fatal(err)
		}
	}
	for _, stmt := range []string{
		`ALTER TABLE users DROP COLUMN kind`,
		`ALTER TABLE transactions DROP COLUMN created_by_user_id`,
		`ALTER TABLE accounts DROP COLUMN original_owner_user_id`,
	} {
		if _, err := mgr.DB().Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"accounts", "budgets", "debts"} {
		if _, err := mgr.DB().Exec(`ALTER TABLE ` + table + ` DROP COLUMN currency`); err != nil {
			t.Fatal(err)
		}
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
	if version != 67 {
		t.Fatalf("goose version %d, want 67", version)
	}
	for _, table := range []string{
		"merchant_aliases",
		"receipt_scan_jobs",
		"stores",
		"receipt_imports",
		"receipt_fiscal_previews",
		"receipt_import_item_categories",
		"households",
		"household_members",
		"household_invites",
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

// Migration 063 backfills accounts/budgets from the owner's profile currency and
// debts from the currency of their transaction's account (else the owner's profile).
func TestMigration063AccountCurrencyBackfill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v62.db")
	sqlDB, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}
	if err := goose.UpTo(sqlDB, "migrations", 62); err != nil {
		t.Fatal(err)
	}

	// u-usd owns a debt that points at a transaction on u-eur's account: the debt
	// must take the account currency (EUR), not the debt owner's profile (USD).
	if _, err := sqlDB.Exec(`
		INSERT INTO users (id, login, password_hash, currency) VALUES
			('u-usd', 'usd', 'x', 'USD'),
			('u-eur', 'eur', 'x', 'EUR');
		INSERT INTO accounts (id, user_id, name, type) VALUES
			('a-usd', 'u-usd', 'Cash', 'cash'),
			('a-eur', 'u-eur', 'Cash', 'cash');
		INSERT INTO transactions (id, user_id, account_id, type, amount, transaction_date) VALUES
			('t-eur', 'u-eur', 'a-eur', 'expense', 100, '2026-01-01');
		INSERT INTO budgets (id, user_id, name, scope, amount, month) VALUES
			('b-usd', 'u-usd', 'All', 'all_expense', 1000, '2026-01');
		INSERT INTO debtors (id, user_id, name) VALUES ('d', 'u-usd', 'Bob');
		INSERT INTO debts (id, user_id, debtor_id, direction, amount, debt_date, due_date, transaction_id) VALUES
			('debt-tx', 'u-usd', 'd', 'lent', 100, '2026-01-01', '2026-02-01', 't-eur'),
			('debt-plain', 'u-usd', 'd', 'lent', 100, '2026-01-01', '2026-02-01', NULL);
	`); err != nil {
		t.Fatal(err)
	}

	if err := migrate(sqlDB); err != nil {
		t.Fatal(err)
	}
	assertMigrationChain(t, sqlDB)

	for _, tc := range []struct{ query, id, want string }{
		{`SELECT currency FROM accounts WHERE id = ?`, "a-usd", "USD"},
		{`SELECT currency FROM accounts WHERE id = ?`, "a-eur", "EUR"},
		{`SELECT currency FROM budgets WHERE id = ?`, "b-usd", "USD"},
		{`SELECT currency FROM debts WHERE id = ?`, "debt-tx", "EUR"},
		{`SELECT currency FROM debts WHERE id = ?`, "debt-plain", "USD"},
	} {
		var got string
		if err := sqlDB.QueryRow(tc.query, tc.id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("%s for %s = %q, want %q", tc.query, tc.id, got, tc.want)
		}
	}
}

func TestMigration064BudgetUniqueIncludesCurrency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	sqlDB := mgr.DB()

	if _, err := sqlDB.Exec(`
		INSERT INTO users (id, login, password_hash, currency) VALUES ('u', 'u', 'x', 'RUB');
		INSERT INTO budgets (id, user_id, name, scope, amount, month, currency, is_active) VALUES
			('b-rub', 'u', 'All RUB', 'all_expense', 1000, '2026-01', 'RUB', 1),
			('b-usd', 'u', 'All USD', 'all_expense', 1000, '2026-01', 'USD', 1);
	`); err != nil {
		t.Fatalf("two currencies should coexist: %v", err)
	}
	_, err = sqlDB.Exec(`
		INSERT INTO budgets (id, user_id, name, scope, amount, month, currency, is_active)
		VALUES ('b-usd-2', 'u', 'Dup', 'all_expense', 1, '2026-01', 'USD', 1)`)
	if err == nil {
		t.Fatal("expected unique conflict on same currency")
	}
}
