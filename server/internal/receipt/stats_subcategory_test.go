package receipt

import (
	"context"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/category"
	"github.com/kai-zer-ru/buhgalter/internal/stats"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

func TestAttach_subcategoryAppearsInStats(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID

	sub, err := category.CreateSubcategory(ctx, env.mgr.DB(), env.userID, cat, "Гигиена", "soap")
	if err != nil {
		t.Fatal(err)
	}

	txRow, err := transaction.Create(ctx, env.mgr.DB(), env.userID, transaction.CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          receipt.TotalSum,
		CategoryID:      &cat,
		TransactionDate: receipt.DateTime,
	})
	if err != nil {
		t.Fatal(err)
	}

	items := attachItemsForReceipt(receipt, cat)
	items[0].SubcategoryID = &sub.ID

	_, err = Attach(ctx, env.mgr.DB(), env.userID, AttachInput{
		TransactionID:        txRow.ID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items:                items,
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := stats.New(env.mgr.DB())
	out, err := svc.BySubcategory(ctx, env.userID, stats.Filters{})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, row := range out {
		if row.SubcategoryID == sub.ID {
			found = true
			if row.Total != 10_000 {
				t.Fatalf("expected hygiene line 10000, got %d", row.Total)
			}
			if row.Count != 1 {
				t.Fatalf("expected count 1, got %d", row.Count)
			}
		}
	}
	if !found {
		t.Fatal("subcategory Гигиена missing from stats after attach")
	}
}

func TestImport_subcategoryAppearsInStats(t *testing.T) {
	env := seedReceiptEnv(t)
	ctx := context.Background()
	receipt := sampleParsedReceipt()
	cat := env.expenseID

	sub, err := category.CreateSubcategory(ctx, env.mgr.DB(), env.userID, cat, "Гигиена", "soap")
	if err != nil {
		t.Fatal(err)
	}

	_, err = Import(ctx, env.mgr.DB(), env.userID, ImportInput{
		AccountID:            env.accountID,
		FiscalDriveNumber:    receipt.FiscalDriveNumber,
		FiscalDocumentNumber: receipt.FiscalDocumentNumber,
		FiscalSign:           receipt.FiscalSign,
		Receipt:              receipt,
		Items: []ImportItemInput{
			{Index: 0, CategoryID: &cat, SubcategoryID: &sub.ID, Include: true},
			{Index: 1, CategoryID: &cat, Include: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := stats.New(env.mgr.DB())
	out, err := svc.BySubcategory(ctx, env.userID, stats.Filters{})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, row := range out {
		if row.SubcategoryID == sub.ID {
			found = true
			if row.Total != 10_000 {
				t.Fatalf("expected hygiene line 10000, got %d", row.Total)
			}
		}
	}
	if !found {
		t.Fatal("subcategory Гигиена missing from stats after import")
	}
}
