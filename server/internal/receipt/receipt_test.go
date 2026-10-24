package receipt

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"database/sql"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
	"github.com/kai-zer-ru/buhgalter/internal/store"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

type receiptSeed struct {
	mgr       *db.Manager
	userID    string
	accountID string
	expenseID string
}

func seedReceiptEnv(t *testing.T) receiptSeed {
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
	userID, err := auth.CreateUser(ctx, sqlDB, "receiptuser", hash, "Receipt User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	accountID := "acc-receipt"
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Кошелёк', 'cash', 100000, 'active', datetime('now'), datetime('now'))`,
		accountID, userID)
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
	return receiptSeed{mgr: mgr, userID: userID, accountID: accountID, expenseID: cats[0].ID}
}

func sampleParsedReceipt() fns.ParsedReceipt {
	dt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	return fns.ParsedReceipt{
		FiscalDriveNumber:    "7288940700123456",
		FiscalDocumentNumber: "12345",
		FiscalSign:           "9876543210",
		DateTime:             dt,
		TotalSum:             15000,
		StoreName:            "Магнит",
		UserINN:              "7701234567",
		Items: []fns.Item{
			{Name: "Молоко", Quantity: 1, Price: 10000, Sum: 10000},
			{Name: "Хлеб", Quantity: 1, Price: 5000, Sum: 5000},
		},
		FNSCheckURL: "https://proverkacheka.nalog.ru/?fn=7288940700123456",
	}
}

func TestImport_createsStoreAndTransactions(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID

	result, err := Import(ctx, env.mgr.DB(), env.userID, ImportInput{
		AccountID:            env.accountID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items: []ImportItemInput{
			{Index: 0, CategoryID: &cat, Include: true},
			{Index: 1, CategoryID: &cat, Include: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(result.Transactions))
	}
	for _, tx := range result.Transactions {
		if tx.ReceiptImportID == nil || *tx.ReceiptImportID != result.ReceiptImportID {
			t.Fatalf("missing receipt_import_id on tx %s", tx.ID)
		}
		if tx.StoreID == nil || tx.StoreName == nil || *tx.StoreName != "Магнит" {
			t.Fatalf("expected store on tx %+v", tx)
		}
		if tx.MerchantID == nil || tx.MerchantName == nil || *tx.MerchantName != "Магнит" {
			t.Fatalf("expected merchant on tx %+v", tx)
		}
		if tx.Description == nil || !strings.Contains(*tx.Description, "Чек №12345") {
			t.Fatalf("unexpected description: %v", tx.Description)
		}
	}

	stores, err := store.List(ctx, env.mgr.DB(), env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 1 || stores[0].Name != "Магнит" {
		t.Fatalf("unexpected stores: %+v", stores)
	}
	if result.Transactions[0].MerchantID == nil || result.Transactions[1].MerchantID == nil ||
		*result.Transactions[0].MerchantID != *result.Transactions[1].MerchantID {
		t.Fatalf("expected one merchant id on included rows: %+v", result.Transactions)
	}
	merchants, err := merchant.List(ctx, env.mgr.DB(), env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(merchants) != 1 || merchants[0].Name != "Магнит" || merchants[0].Icon != "default" {
		t.Fatalf("expected one default merchant Магнит, got %+v", merchants)
	}

	detail, err := GetByID(ctx, env.mgr.DB(), env.userID, result.ReceiptImportID, result.Transactions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Items) != 2 || detail.FNSCheckURL == "" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.HighlightItemIndex == nil || *detail.HighlightItemIndex != 0 {
		t.Fatalf("expected highlight index 0, got %v", detail.HighlightItemIndex)
	}
	if detail.Items[0].CategoryName == "" {
		t.Fatalf("expected category_name on linked item, got %+v", detail.Items[0])
	}
}

func TestImport_duplicateRejected(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	in := ImportInput{
		AccountID:            env.accountID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                []ImportItemInput{{Index: 0, CategoryID: &cat, Include: true}},
	}
	if _, err := Import(ctx, env.mgr.DB(), env.userID, in); err != nil {
		t.Fatal(err)
	}
	if _, err := Import(ctx, env.mgr.DB(), env.userID, in); err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestImport_duplicateAfterAttach(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: receipt.TotalSum,
		CategoryID: &cat, TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: txRow.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	}); err != nil {
		t.Fatal(err)
	}
	_, err = Import(ctx, env.mgr.DB(), env.userID, ImportInput{
		AccountID: env.accountID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt,
		Items:   []ImportItemInput{{Index: 0, CategoryID: &cat, Include: true}, {Index: 1, CategoryID: &cat, Include: true}},
	})
	if err != ErrDuplicate {
		t.Fatalf("expected ErrDuplicate after attach, got %v", err)
	}
}

func TestAttach_linksReceiptToExpense(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	amount := receipt.TotalSum
	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          amount,
		CategoryID:      &cat,
		TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID:        txRow.ID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.ReceiptImportID == nil || *result.Transaction.ReceiptImportID != result.ReceiptImportID {
		t.Fatalf("expected receipt_import_id on transaction, got %+v", result.Transaction)
	}
	if result.Transaction.MerchantID == nil || result.Transaction.MerchantName == nil || *result.Transaction.MerchantName != "Магнит" {
		t.Fatalf("expected merchant from receipt point, got %+v", result.Transaction)
	}
}

func TestImport_reusesMerchantByNameOrAlias(t *testing.T) {
	for _, storeName := range []string{"пятёрочка", "Магнит"} {
		t.Run(storeName, func(t *testing.T) {
			env := seedReceiptEnv(t)
			ctx := context.Background()
			if _, err := merchant.CreateWithAliases(ctx, env.mgr.DB(), env.userID, "Пятёрочка", "shopping", []string{"Магнит"}); err != nil {
				t.Fatal(err)
			}
			receipt := sampleParsedReceipt()
			receipt.StoreName = storeName
			cat := env.expenseID
			result, err := Import(ctx, env.mgr.DB(), env.userID, ImportInput{
				AccountID:            env.accountID,
				FiscalDriveNumber:    receipt.FiscalDriveNumber,
				FiscalDocumentNumber: receipt.FiscalDocumentNumber,
				FiscalSign:           receipt.FiscalSign,
				Receipt:              receipt,
				Items: []ImportItemInput{
					{Index: 0, CategoryID: &cat, Include: true},
					{Index: 1, CategoryID: &cat, Include: false},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Transactions) != 1 {
				t.Fatalf("expected 1 included row, got %d", len(result.Transactions))
			}
			tx := result.Transactions[0]
			if tx.MerchantName == nil || *tx.MerchantName != "Пятёрочка" {
				t.Fatalf("expected existing merchant, got %+v", tx)
			}
			merchants, err := merchant.List(ctx, env.mgr.DB(), env.userID)
			if err != nil {
				t.Fatal(err)
			}
			if len(merchants) != 1 || merchants[0].Icon != "shopping" {
				t.Fatalf("expected the original merchant only, got %+v", merchants)
			}
		})
	}
}

func TestImport_blankStoreNameSkipsMerchant(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	receipt.StoreName = "   "
	cat := env.expenseID
	result, err := Import(ctx, env.mgr.DB(), env.userID, ImportInput{
		AccountID:            env.accountID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                []ImportItemInput{{Index: 0, CategoryID: &cat, Include: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	tx := result.Transactions[0]
	if tx.MerchantID != nil || tx.StoreID != nil {
		t.Fatalf("expected empty merchant and store, got %+v", tx)
	}
	if n := countRows(t, env.mgr.DB(), `SELECT COUNT(*) FROM merchants WHERE user_id = ?`, env.userID); n != 0 {
		t.Fatalf("expected no merchant, got %d", n)
	}
	if n := countRows(t, env.mgr.DB(), `SELECT COUNT(*) FROM stores WHERE user_id = ?`, env.userID); n != 0 {
		t.Fatalf("expected no store, got %d", n)
	}
}

func TestImport_merchantResolveErrorRollsBack(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	if _, err := env.mgr.DB().ExecContext(ctx, `
		CREATE TRIGGER block_merchant_insert BEFORE INSERT ON merchants
		BEGIN
			SELECT RAISE(ABORT, 'block merchant');
		END`); err != nil {
		t.Fatal(err)
	}
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	_, err := Import(ctx, env.mgr.DB(), env.userID, ImportInput{
		AccountID:            env.accountID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                []ImportItemInput{{Index: 0, CategoryID: &cat, Include: true}},
	})
	if err == nil {
		t.Fatal("expected resolve error")
	}
	for _, table := range []string{"merchants", "stores", "receipt_imports", "transactions"} {
		if n := countRows(t, env.mgr.DB(), `SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, env.userID); n != 0 {
			t.Fatalf("expected rollback of %s, got %d rows", table, n)
		}
	}
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
