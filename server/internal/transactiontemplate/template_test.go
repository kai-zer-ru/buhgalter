package transactiontemplate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/tag"
)

func seed(t *testing.T) (context.Context, *db.Handle, string, string, string, string) {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "tpluser", hash, "Tpl", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	acc1 := "acc-1"
	acc2 := "acc-2"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Наличные', 'cash', 0, 'active', datetime('now'), datetime('now')),
		       (?, ?, 'Карта', 'bank', 0, 'active', datetime('now'), datetime('now'))`,
		acc1, userID, acc2, userID)
	if err != nil {
		t.Fatal(err)
	}

	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil || len(cats) == 0 {
		t.Fatalf("expense cats: %v", err)
	}
	return ctx, db.NewHandle(mgr), userID, acc1, acc2, cats[0].ID
}

func TestTemplateExpenseCRUDAndReorder(t *testing.T) {
	ctx, handle, userID, acc1, _, catID := seed(t)
	sqlDB := handle.DB()

	amount := int64(25000)
	desc := "обед"
	a, err := Create(ctx, sqlDB, userID, Input{
		Name:        "Обед",
		Type:        "expense",
		AccountID:   &acc1,
		CategoryID:  &catID,
		Amount:      &amount,
		Description: &desc,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Обед" || a.Amount == nil || *a.Amount != amount {
		t.Fatalf("created=%+v", a)
	}

	b, err := Create(ctx, sqlDB, userID, Input{
		Name:       "Бензин",
		Type:       "expense",
		AccountID:  &acc1,
		CategoryID: &catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Amount != nil {
		t.Fatalf("expected null amount, got %v", *b.Amount)
	}

	list, err := List(ctx, sqlDB, userID)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	if list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("order by create: %+v", list)
	}

	if err := Reorder(ctx, sqlDB, userID, []string{b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	list, _ = List(ctx, sqlDB, userID)
	if list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("after reorder: %+v", list)
	}

	updated, err := Update(ctx, sqlDB, userID, a.ID, Input{
		Name:       "Обед офис",
		Type:       "expense",
		AccountID:  &acc1,
		CategoryID: &catID,
		Amount:     &amount,
	})
	if err != nil || updated.Name != "Обед офис" {
		t.Fatalf("update=%+v err=%v", updated, err)
	}

	if err := Delete(ctx, sqlDB, userID, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, sqlDB, userID, a.ID); err != ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestTemplateTransferAndTags(t *testing.T) {
	ctx, handle, userID, acc1, acc2, _ := seed(t)
	sqlDB := handle.DB()

	_, err := Create(ctx, sqlDB, userID, Input{
		Name:      "Bad",
		Type:      "transfer",
		AccountID: &acc1,
	})
	if err != ErrInvalidAccount {
		t.Fatalf("want invalid account, got %v", err)
	}

	tg, err := tag.Create(ctx, sqlDB, userID, "еда")
	if err != nil {
		t.Fatal(err)
	}
	m, err := merchant.Create(ctx, sqlDB, userID, "Столовая", "default")
	if err != nil {
		t.Fatal(err)
	}

	amount := int64(100000)
	tpl, err := Create(ctx, sqlDB, userID, Input{
		Name:        "Маме",
		Type:        "transfer",
		AccountID:   &acc1,
		ToAccountID: &acc2,
		Amount:      &amount,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.ToAccountID == nil || *tpl.ToAccountID != acc2 {
		t.Fatalf("transfer=%+v", tpl)
	}

	cats, _ := category.ListByUser(ctx, sqlDB, userID, "expense")
	catID := cats[0].ID
	withTags, err := Create(ctx, sqlDB, userID, Input{
		Name:       "Столовая",
		Type:       "expense",
		AccountID:  &acc1,
		CategoryID: &catID,
		MerchantID: &m.ID,
		TagIDs:     []string{tg.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(withTags.Tags) != 1 || withTags.Tags[0].ID != tg.ID {
		t.Fatalf("tags=%+v", withTags.Tags)
	}
	if withTags.MerchantID == nil || *withTags.MerchantID != m.ID {
		t.Fatalf("merchant=%+v", withTags)
	}
}

func TestTemplateRejectsSystemCategory(t *testing.T) {
	ctx, handle, userID, acc1, _, _ := seed(t)
	sqlDB := handle.DB()
	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil {
		t.Fatal(err)
	}
	var systemID string
	for _, c := range cats {
		if c.IsSystem {
			systemID = c.ID
			break
		}
	}
	if systemID == "" {
		t.Skip("no system expense category")
	}
	_, err = Create(ctx, sqlDB, userID, Input{
		Name:       "Sys",
		Type:       "expense",
		AccountID:  &acc1,
		CategoryID: &systemID,
	})
	if err != ErrInvalidCategory {
		t.Fatalf("want invalid category, got %v", err)
	}
}
