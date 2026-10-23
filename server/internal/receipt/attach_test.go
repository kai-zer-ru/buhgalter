package receipt

import (
	"context"
	"errors"
	"testing"
	"time"

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
