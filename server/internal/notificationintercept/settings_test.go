package notificationintercept

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestReplaceSettingsValidatesAccountType(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u1", hash, "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	q := sqlcdb.New(mgr.DB())
	if err := q.InsertAccount(ctx, sqlcdb.InsertAccountParams{
		ID:             "acc-cash",
		UserID:         userID,
		Name:           "Cash",
		Type:           "cash",
		InitialBalance: 0,
		CurrentBalance: 0,
		IsPrimary:      1,
		CreatedAt:      "2020-01-01T00:00:00Z",
		UpdatedAt:      "2020-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	_, err = ReplaceSettings(ctx, mgr.DB(), userID, UpdateInput{
		BankBindings: []BankBinding{{
			PackageName: "ru.sberbankmobile",
			BankID:      "sberbank",
			AccountID:   "acc-cash",
		}},
	})
	if err == nil || err.Error() != "ERR_NOTIFICATION_INTERCEPT_ACCOUNT_TYPE" {
		t.Fatalf("expected account type error, got %v", err)
	}
}

func TestGetReplaceRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u2", hash, "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if err := bank.SeedIfEmpty(ctx, mgr.DB()); err != nil {
		t.Fatal(err)
	}
	_, err = mgr.DB().ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, bank_id, initial_balance, current_balance, status, is_primary, created_at, updated_at)
		VALUES (?, ?, 'Bank', 'bank', 'tinkoff', 0, 0, 'active', 1, datetime('now'), datetime('now'))`,
		"acc-bank", userID)
	if err != nil {
		t.Fatal(err)
	}

	in := UpdateInput{
		BankBindings: []BankBinding{{
			PackageName: "com.idamob.tinkoff.android",
			BankID:      "tinkoff",
			AccountID:   "acc-bank",
		}},
		CardBindings: []CardBinding{{
			BankID:    "tinkoff",
			Last4:     "4242",
			AccountID: "acc-bank",
		}},
	}
	saved, err := ReplaceSettings(ctx, mgr.DB(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.BankBindings) != 1 {
		t.Fatalf("bank bindings: %+v", saved.BankBindings)
	}
	got, err := GetSettings(ctx, mgr.DB(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.BankBindings) != 1 || got.BankBindings[0].AccountID != "acc-bank" {
		t.Fatalf("bank: %+v", got.BankBindings)
	}
	if len(got.CardBindings) != 1 || got.CardBindings[0].Last4 != "4242" {
		t.Fatalf("card: %+v", got.CardBindings)
	}
}
