package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/receipt"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/store"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func TestByStore_groupsCatalogMerchants(t *testing.T) {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "statsuser", hash, "Stats User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES ('acc-stats', ?, 'Кошелёк', 'cash', 100000, 'active', datetime('now'), datetime('now'))`, userID)
	if err != nil {
		t.Fatal(err)
	}
	cats, err := category.ListByUser(ctx, sqlDB, userID, "expense")
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) == 0 {
		t.Fatal("expected expense categories")
	}
	cat := cats[0].ID
	past := time.Now().UTC().Add(-2 * time.Hour)

	lenta := "Лента"
	manual, err := transaction.Create(ctx, sqlDB, userID, transaction.CreateInput{
		AccountID: "acc-stats", Type: "expense", Amount: 4000,
		CategoryID: &cat, MerchantName: &lenta, TransactionDate: past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Create(ctx, sqlDB, userID, transaction.CreateInput{
		AccountID: "acc-stats", Type: "expense", Amount: 2000,
		CategoryID: &cat, TransactionDate: past,
	}); err != nil {
		t.Fatal(err)
	}
	point, err := store.GetOrCreate(ctx, sqlDB, userID, "Только точка")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transaction.Create(ctx, sqlDB, userID, transaction.CreateInput{
		AccountID: "acc-stats", Type: "expense", Amount: 1000,
		CategoryID: &cat, StoreID: point, TransactionDate: past,
	}); err != nil {
		t.Fatal(err)
	}

	receiptAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	parsed := fns.ParsedReceipt{
		FiscalDriveNumber: "7288940700123456", FiscalDocumentNumber: "12345", FiscalSign: "9876543210",
		DateTime: receiptAt, TotalSum: 10000, StoreName: "Магнит",
		Items:       []fns.Item{{Name: "Молоко", Quantity: 1, Price: 10000, Sum: 10000}},
		FNSCheckURL: "https://proverkacheka.nalog.ru/?fn=7288940700123456",
	}
	if _, err := receipt.Import(ctx, sqlDB, userID, receipt.ImportInput{
		AccountID: "acc-stats", FiscalDriveNumber: parsed.FiscalDriveNumber,
		FiscalDocumentNumber: parsed.FiscalDocumentNumber, FiscalSign: parsed.FiscalSign,
		Receipt: parsed,
		Items:   []receipt.ImportItemInput{{Index: 0, CategoryID: &cat, Include: true}},
	}); err != nil {
		t.Fatal(err)
	}

	svc := New(sqlDB)
	rows, err := svc.ByStore(ctx, userID, Filters{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, row := range rows {
		got[row.StoreName] = row.Total
		if row.StoreID == "" {
			t.Fatalf("empty store_id key: %+v", row)
		}
	}
	if got["Лента"] != 4000 || got["Магнит"] != 10000 || len(got) != 2 {
		t.Fatalf("expected manual and receipt merchants only, got %+v", rows)
	}

	filtered, err := svc.ByStore(ctx, userID, Filters{MerchantID: *manual.MerchantID})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].StoreName != "Лента" || filtered[0].StoreID != *manual.MerchantID {
		t.Fatalf("merchant filter: %+v", filtered)
	}
	summary, err := svc.Summary(ctx, userID, Filters{MerchantID: *manual.MerchantID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ExpenseTotal != 4000 || summary.TransactionCount != 1 {
		t.Fatalf("summary merchant filter: %+v", summary)
	}
}
