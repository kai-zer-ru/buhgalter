package receipt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/merchant"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func TestAttach_sumMismatch(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          receipt.TotalSum + 100,
		CategoryID:      &cat,
		TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID:        txRow.ID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                attachItemsForReceipt(receipt, cat),
	})
	var mismatch *SumMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("expected SumMismatchError, got %v", err)
	}
	if mismatch.TransactionAmount != txRow.Amount || mismatch.ReceiptTotalSum != receipt.TotalSum {
		t.Fatalf("unexpected amounts: %+v", mismatch)
	}

	// Acknowledged attach succeeds.
	result, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID:          txRow.ID,
		FiscalDriveNumber:      receipt.FiscalDriveNumber,
		FiscalDocumentNumber:   receipt.FiscalDocumentNumber,
		FiscalSign:             receipt.FiscalSign,
		Receipt:                receipt,
		Items:                  attachItemsForReceipt(receipt, cat),
		AcknowledgeSumMismatch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.ReceiptImportID == nil {
		t.Fatal("expected receipt_import_id")
	}
}

func TestAttach_rejectsSecondTransaction(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID

	first, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: receipt.TotalSum,
		CategoryID: &cat, TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: first.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: receipt.TotalSum,
		CategoryID: &cat, TransactionDate: receipt.DateTime.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: second.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate on second attach, got %v", err)
	}
}

func TestAttach_setsMerchantWhenMissing(t *testing.T) {
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
	result, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: txRow.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.MerchantName == nil || *result.Transaction.MerchantName != "Магнит" || result.Transaction.StoreID == nil {
		t.Fatalf("expected point merchant and store, got %+v", result.Transaction)
	}
}

func TestAttach_keepsExistingMerchant(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID
	name := "Лента"
	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: receipt.TotalSum,
		CategoryID: &cat, MerchantName: &name, TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: txRow.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.MerchantID == nil || result.Transaction.MerchantName == nil ||
		*result.Transaction.MerchantID != *txRow.MerchantID || *result.Transaction.MerchantName != "Лента" {
		t.Fatalf("merchant overwritten: before %v after %+v", txRow.MerchantID, result.Transaction)
	}
	merchants, err := merchant.List(ctx, env.mgr.DB(), env.userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(merchants) != 1 || merchants[0].Name != "Лента" {
		t.Fatalf("expected not to create the receipt point, got %+v", merchants)
	}
}

func TestAttach_blankStoreNameLeavesMerchant(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	receipt.StoreName = "  "
	cat := env.expenseID
	name := "Лента"
	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID: env.accountID, Type: "expense", Amount: receipt.TotalSum,
		CategoryID: &cat, MerchantName: &name, TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: txRow.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.MerchantID == nil || *result.Transaction.MerchantID != *txRow.MerchantID {
		t.Fatalf("expected merchant unchanged, got %+v", result.Transaction)
	}
	if n := countRows(t, env.mgr.DB(), `SELECT COUNT(*) FROM merchants WHERE user_id = ?`, env.userID); n != 1 {
		t.Fatalf("expected the original merchant only, got %d", n)
	}
}
