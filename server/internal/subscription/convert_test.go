package subscription_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/recurring"
	"github.com/kai-zer-ru/buhgalter/internal/subscription"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func setupConvert(t *testing.T) (*db.Manager, string, string, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "cuser", "hash", "C", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	accID := "acc-1"
	now := timeutil.FormatUTC(timeutil.NowUTC())
	if _, err := mgr.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, current_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 0, 'active', ?, ?)`, accID, userID, now, now); err != nil {
		t.Fatal(err)
	}
	catID := "cat-1"
	if _, err := mgr.DB().ExecContext(ctx, `
		INSERT INTO categories (id, user_id, name, type, icon, sort_order, is_primary, is_system, created_at)
		VALUES (?, ?, 'Misc', 'expense', 'default', 0, 1, 0, ?)`, catID, userID, now); err != nil {
		t.Fatal(err)
	}
	return mgr, userID, accID, catID
}

func TestConvertPreservesExplicitTimeLocal(t *testing.T) {
	mgr, userID, accID, catID := setupConvert(t)
	ctx := context.Background()
	day := int64(15)
	op, err := recurring.Create(ctx, mgr.DB(), userID, recurring.Input{
		Type: "expense", Amount: 10000, AccountID: accID, CategoryID: catID,
		Period: "month", DayOfMonth: &day,
		StartDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		TimeLocal: "14:30", Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := subscription.ConvertFromRecurring(ctx, mgr.DB(), userID, op.ID, "", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Subscription.TimeLocal != "14:30" {
		t.Fatalf("got time_local %q want 14:30", res.Subscription.TimeLocal)
	}
}

func TestConvertPreservesMidnightTimeLocal(t *testing.T) {
	mgr, userID, accID, catID := setupConvert(t)
	ctx := context.Background()
	day := int64(10)
	op, err := recurring.Create(ctx, mgr.DB(), userID, recurring.Input{
		Type: "expense", Amount: 29900, AccountID: accID, CategoryID: catID,
		Period: "month", DayOfMonth: &day,
		StartDate: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		TimeLocal: "00:00", Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := subscription.ConvertFromRecurring(ctx, mgr.DB(), userID, op.ID, "Яндекс Плюс", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Subscription.TimeLocal != "00:00" {
		t.Fatalf("got time_local %q want 00:00", res.Subscription.TimeLocal)
	}
}
