package accountbalance

import (
	"context"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestComputeAndRefreshBalance(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	mgr, err := db.NewManager(dir + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	sqlDB := mgr.DB()

	userID := "user-1"
	accountID := "acc-1"
	_, err = sqlDB.ExecContext(ctx, `INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'test', 'hash', 'UTC')`, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Test', 'cash', 100000, 100000, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO transactions (id, user_id, account_id, type, kind, amount, transaction_date, created_at, updated_at)
		VALUES ('tx-1', ?, ?, 'income', 'manual', 5000, datetime('now'), datetime('now'), datetime('now'))`,
		userID, accountID)
	if err != nil {
		t.Fatal(err)
	}

	if err := Refresh(ctx, sqlDB, userID, accountID); err != nil {
		t.Fatal(err)
	}

	var stored int64
	err = sqlDB.QueryRowContext(ctx, `SELECT current_balance FROM accounts WHERE id = ?`, accountID).Scan(&stored)
	if err != nil {
		t.Fatal(err)
	}
	if stored != 105000 {
		t.Fatalf("expected stored balance 105000, got %d", stored)
	}
}
