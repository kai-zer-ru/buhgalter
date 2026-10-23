package receipt

import (
	"context"
	"errors"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func TestDetach_clearsReceiptLink(t *testing.T) {
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
	attached, err := Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID: txRow.ID, FiscalDriveNumber: receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber, FiscalSign: receipt.FiscalSign,
		Receipt: receipt, Items: attachItemsForReceipt(receipt, cat),
	})
	if err != nil {
		t.Fatal(err)
	}
	if attached.Transaction.ReceiptImportID == nil {
		t.Fatal("expected attach")
	}

	result, err := Detach(ctx, env.mgr.DB(), env.userID, txRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Transaction.ReceiptImportID != nil && *result.Transaction.ReceiptImportID != "" {
		t.Fatalf("expected cleared receipt_import_id, got %v", result.Transaction.ReceiptImportID)
	}

	_, err = Detach(ctx, env.mgr.DB(), env.userID, txRow.ID)
	if !errors.Is(err, ErrNotAttached) {
		t.Fatalf("expected ErrNotAttached, got %v", err)
	}
}
