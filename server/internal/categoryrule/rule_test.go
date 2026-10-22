package categoryrule

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func seed(t *testing.T) (context.Context, *db.Handle, string, string, string, string, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
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
	userID, err := auth.CreateUser(ctx, sqlDB, "rulesuser", hash, "Rules", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	cashID := "acc-cash"
	cardID := "acc-card"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Наличные', 'cash', 0, 'active', datetime('now'), datetime('now')),
		       (?, ?, 'Карта', 'bank', 0, 'active', datetime('now'), datetime('now'))`,
		cashID, userID, cardID, userID)
	if err != nil {
		t.Fatal(err)
	}

	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil || len(cats) == 0 {
		t.Fatalf("expense cats: %v", err)
	}
	catID := cats[0].ID
	sub, err := category.CreateSubcategory(ctx, sqlDB, userID, catID, "Автобус", "")
	if err != nil {
		t.Fatal(err)
	}
	return ctx, db.NewHandle(mgr), userID, cashID, cardID, catID, sub.ID
}

func TestCategoryAccountCRUDReorderAndMatch(t *testing.T) {
	ctx, handle, userID, cashID, cardID, catID, subID := seed(t)
	sqlDB := handle.DB()

	rule, err := Create(ctx, sqlDB, userID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		SubcategoryID:    &subID,
		DefaultAccountID: &cashID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.Kind != KindCategoryAccount || rule.DefaultAccountID == nil || *rule.DefaultAccountID != cashID {
		t.Fatalf("created=%+v", rule)
	}

	got, err := Get(ctx, sqlDB, userID, rule.ID)
	if err != nil || got.ID != rule.ID {
		t.Fatalf("get: %v %+v", err, got)
	}

	broad, err := Create(ctx, sqlDB, userID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		DefaultAccountID: &cardID,
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := List(ctx, sqlDB, userID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}

	if err := Reorder(ctx, sqlDB, userID, []string{broad.ID, rule.ID}); err != nil {
		t.Fatal(err)
	}
	list, _ = List(ctx, sqlDB, userID)
	if list[0].ID != broad.ID {
		t.Fatalf("order: %+v", list)
	}

	acc, err := SuggestDefaultAccount(ctx, sqlDB, userID, catID, &subID)
	if err != nil || acc == nil || *acc != cardID {
		t.Fatalf("specific sub should lose to broad rule first: acc=%v err=%v", acc, err)
	}

	acc, err = SuggestDefaultAccount(ctx, sqlDB, userID, catID, nil)
	if err != nil || acc == nil || *acc != cardID {
		t.Fatalf("nil sub: acc=%v err=%v", acc, err)
	}

	if err := Reorder(ctx, sqlDB, userID, []string{rule.ID, broad.ID}); err != nil {
		t.Fatal(err)
	}
	acc, _ = SuggestDefaultAccount(ctx, sqlDB, userID, catID, &subID)
	if acc == nil || *acc != cashID {
		t.Fatalf("after reorder specific wins: %v", acc)
	}

	updated, err := Update(ctx, sqlDB, userID, rule.ID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		SubcategoryID:    &subID,
		DefaultAccountID: &cardID,
	})
	if err != nil || updated.DefaultAccountID == nil || *updated.DefaultAccountID != cardID {
		t.Fatalf("update: %+v err=%v", updated, err)
	}

	if err := Delete(ctx, sqlDB, userID, rule.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, sqlDB, userID, rule.ID); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestTextCategoryRejectsSystemCategory(t *testing.T) {
	ctx, handle, userID, _, _, catID, _ := seed(t)
	sqlDB := handle.DB()

	var systemID string
	err := sqlDB.QueryRowContext(ctx, `
		SELECT id FROM categories WHERE user_id = ? AND is_system = 1 LIMIT 1`, userID).Scan(&systemID)
	if err != nil {
		t.Fatal(err)
	}

	pat := "кофе"
	_, err = Create(ctx, sqlDB, userID, Input{
		Kind:       KindTextCategory,
		CategoryID: systemID,
		Pattern:    &pat,
	})
	if err != ErrSystemCategory {
		t.Fatalf("expected system err, got %v", err)
	}

	_, err = Create(ctx, sqlDB, userID, Input{
		Kind:       KindTextCategory,
		CategoryID: catID,
		Pattern:    &pat,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCategoryAccountValidation(t *testing.T) {
	ctx, handle, userID, cashID, _, catID, _ := seed(t)
	sqlDB := handle.DB()

	_, err := Create(ctx, sqlDB, userID, Input{
		Kind:       KindCategoryAccount,
		CategoryID: catID,
	})
	if err != ErrInvalidAccount {
		t.Fatalf("expected account err, got %v", err)
	}

	pat := "x"
	_, err = Create(ctx, sqlDB, userID, Input{
		Kind:             KindCategoryAccount,
		CategoryID:       catID,
		DefaultAccountID: &cashID,
		Pattern:          &pat,
	})
	if err != ErrInvalidPattern {
		t.Fatalf("expected pattern err, got %v", err)
	}
}
