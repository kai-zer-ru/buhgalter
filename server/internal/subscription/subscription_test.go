package subscription_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/subscription"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func setup(t *testing.T) (*db.Manager, string, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "subuser", "hash", "Sub", false, auth.UserStatusActive)
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
	return mgr, userID, accID
}

func TestCreateNoSubcategoryUntilApply(t *testing.T) {
	mgr, userID, accID := setup(t)
	ctx := context.Background()
	day := int64(1)
	sub, err := subscription.Create(ctx, mgr.DB(), userID, subscription.Input{
		Name: "Netflix", Amount: 99900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		TimeLocal: "08:00", Active: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sub.SubcategoryID != nil {
		t.Fatalf("expected no subcategory before charge, got %v", sub.SubcategoryID)
	}
	var count int
	_ = mgr.DB().QueryRow(`SELECT COUNT(*) FROM subcategories`).Scan(&count)
	if count != 0 {
		t.Fatalf("expected 0 subcategories, got %d", count)
	}

	tz := "Europe/Moscow"
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Hour))
	_, err = sqlcdb.New(mgr.DB()).MarkSubscriptionRan(ctx, sqlcdb.MarkSubscriptionRanParams{
		NextRunAt: past, LastRunAt: nil, SubcategoryID: nil,
		UpdatedAt: timeutil.FormatUTC(timeutil.NowUTC()), ID: sub.ID, UserID: userID,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := subscription.ApplyDue(ctx, mgr.DB(), userID, timeutil.NowUTC(), tz)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("applied %d", n)
	}
	got, err := subscription.List(ctx, mgr.DB(), userID)
	if err != nil || len(got) != 1 || got[0].SubcategoryID == nil {
		t.Fatalf("expected subcategory after apply: %+v err=%v", got, err)
	}
	catID, err := categoryseed.SubscriptionsCategoryID(ctx, mgr.DB(), userID)
	if err != nil {
		t.Fatal(err)
	}
	var txCat, txSub, txSubID string
	err = mgr.DB().QueryRow(`SELECT category_id, subcategory_id, subscription_id FROM transactions WHERE user_id = ?`, userID).
		Scan(&txCat, &txSub, &txSubID)
	if err != nil {
		t.Fatal(err)
	}
	if txCat != catID || txSubID != sub.ID || txSub == "" {
		t.Fatalf("tx cat=%s sub=%s sid=%s", txCat, txSub, txSubID)
	}
}

func TestSameNameSharesSubcategory(t *testing.T) {
	mgr, userID, accID := setup(t)
	ctx := context.Background()
	day := int64(5)
	in := subscription.Input{
		Name: "Яндекс Плюс", Amount: 29900, AccountID: accID, Period: "month",
		DayOfMonth: &day, StartDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		TimeLocal: "08:00", Active: true,
	}
	a, err := subscription.Create(ctx, mgr.DB(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	desc := "мамина"
	in.Description = &desc
	in.Amount = 39900
	b, err := subscription.Create(ctx, mgr.DB(), userID, in)
	if err != nil {
		t.Fatal(err)
	}
	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Hour))
	for _, id := range []string{a.ID, b.ID} {
		_, _ = sqlcdb.New(mgr.DB()).MarkSubscriptionRan(ctx, sqlcdb.MarkSubscriptionRanParams{
			NextRunAt: past, UpdatedAt: past, ID: id, UserID: userID,
		})
	}
	if _, err := subscription.ApplyDue(ctx, mgr.DB(), userID, timeutil.NowUTC(), "Europe/Moscow"); err != nil {
		t.Fatal(err)
	}
	list, err := subscription.List(ctx, mgr.DB(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].SubcategoryID == nil || list[1].SubcategoryID == nil || *list[0].SubcategoryID != *list[1].SubcategoryID {
		t.Fatalf("expected shared subcategory: %+v", list)
	}
}
