package transaction

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestInboxListAndCount(t *testing.T) {
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "inbox.db"))
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
	userID, err := auth.CreateUser(ctx, sqlDB, "inboxuser", hash, "Inbox", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	accID := "acc-inbox-1"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Cash', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		accID, userID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Create(ctx, sqlDB, userID, CreateInput{
		AccountID:       accID,
		Type:            "expense",
		Amount:          1000,
		Description:     strPtr("uncategorized e2e"),
		TransactionDate: mustParseTxDate(t, "2026-01-15 12:00:00"),
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := CountInbox(ctx, sqlDB, userID)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}

	res, err := ListInbox(ctx, sqlDB, userID, 1, 20, "date_desc")
	if err != nil || res.Meta.Total != 1 || len(res.Data) != 1 {
		t.Fatalf("list=%+v err=%v", res, err)
	}
	if res.Data[0].CategoryID != nil {
		t.Fatalf("expected null category, got %v", res.Data[0].CategoryID)
	}
	if res.Data[0].Currency != "RUB" {
		t.Fatalf("inbox currency %q, want RUB", res.Data[0].Currency)
	}
}

func TestInboxCarriesAccountCurrency(t *testing.T) {
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "inbox-usd.db"))
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
	userID, err := auth.CreateUser(ctx, sqlDB, "inboxusd", hash, "Inbox USD", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	accID := "acc-inbox-usd"
	if _, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, currency, created_at, updated_at)
		VALUES (?, ?, 'USD', 'cash', 0, 'active', 'USD', datetime('now'), datetime('now'))`,
		accID, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, sqlDB, userID, CreateInput{
		AccountID:       accID,
		Type:            "expense",
		Amount:          2500,
		Description:     strPtr("usd inbox"),
		TransactionDate: mustParseTxDate(t, "2026-01-15 12:00:00"),
	}); err != nil {
		t.Fatal(err)
	}
	res, err := ListInbox(ctx, sqlDB, userID, 1, 20, "date_desc")
	if err != nil || len(res.Data) != 1 {
		t.Fatalf("list=%+v err=%v", res, err)
	}
	if res.Data[0].Currency != "USD" {
		t.Fatalf("inbox currency %q, want USD", res.Data[0].Currency)
	}
}

func mustParseTxDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := timeutil.ParseUTC(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
