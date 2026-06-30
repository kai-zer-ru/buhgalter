package recurring

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func seedRecurringEnv(t *testing.T) (context.Context, *db.Handle, string, string, string) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	sqlDB := mgr.DB()

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, sqlDB, "recuser", hash, "Recurring", false)
	if err != nil {
		t.Fatal(err)
	}

	accountID := "acc-rec"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Кошелёк', 'cash', 100000, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
	if err != nil {
		t.Fatal(err)
	}
	if err := accountbalance.Refresh(ctx, sqlDB, userID, accountID); err != nil {
		t.Fatal(err)
	}

	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) == 0 {
		t.Fatal("expected expense categories")
	}

	return ctx, db.NewHandle(mgr), userID, accountID, cats[0].ID
}

func TestApplyDueUpdatesAccountBalance(t *testing.T) {
	ctx, handle, userID, accountID, categoryID := seedRecurringEnv(t)
	sqlDB := handle.DB()

	startDate := timeutil.NowUTC().AddDate(0, -1, 0)
	day := int64(startDate.Day())
	op, err := Create(ctx, sqlDB, userID, Input{
		Type:       "expense",
		Amount:     5_000,
		AccountID:  accountID,
		CategoryID: categoryID,
		Period:     "month",
		DayOfMonth: &day,
		StartDate:  startDate,
		TimeLocal:  "08:00",
		Active:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Hour))
	_, err = sqlDB.ExecContext(ctx, `UPDATE recurring_operations SET next_run_at = ? WHERE id = ?`, past, op.ID)
	if err != nil {
		t.Fatal(err)
	}

	applied, err := ApplyDue(ctx, sqlDB, userID, timeutil.NowUTC(), "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied operation, got %d", applied)
	}

	var balance int64
	err = sqlDB.QueryRowContext(ctx, `SELECT current_balance FROM accounts WHERE id = ?`, accountID).Scan(&balance)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 95_000 {
		t.Fatalf("expected balance 95000 after recurring expense 5000, got %d", balance)
	}
}
