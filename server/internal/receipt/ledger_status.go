package receipt

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

func applyLedgerFlags(ctx context.Context, db *sql.DB, userID string, out *fns.ParsedReceipt) {
	if out == nil {
		return
	}
	fn, fd, fp := NormalizeFiscalKey(out.FiscalDriveNumber, out.FiscalDocumentNumber, out.FiscalSign, *out)
	if fn == "" || fd == "" || fp == "" {
		return
	}
	importID, err := queries(db).GetReceiptImportByFiscalKey(ctx, sqlcdb.GetReceiptImportByFiscalKeyParams{
		UserID:               userID,
		FiscalDriveNumber:    fn,
		FiscalDocumentNumber: fd,
		FiscalSign:           fp,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		return
	}
	out.AlreadyImported = true
	rid := importID
	count, err := queries(db).CountReceiptLinkedTransactions(ctx, sqlcdb.CountReceiptLinkedTransactionsParams{
		UserID:          userID,
		ReceiptImportID: &rid,
	})
	if err != nil {
		return
	}
	out.AlreadyInLedger = count > 0
}
