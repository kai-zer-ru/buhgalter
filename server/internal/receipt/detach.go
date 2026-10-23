package receipt

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/transaction"
)

// ErrNotAttached means the transaction has no linked receipt.
var ErrNotAttached = errors.New("receipt not attached")

type DetachResult struct {
	Transaction transaction.Transaction `json:"transaction"`
}

func Detach(ctx context.Context, db *sql.DB, userID, transactionID string) (DetachResult, error) {
	txID := strings.TrimSpace(transactionID)
	if txID == "" {
		return DetachResult{}, transaction.ErrNotFound
	}
	txBefore, err := transaction.GetByID(ctx, db, userID, txID)
	if errors.Is(err, transaction.ErrNotFound) {
		return DetachResult{}, ErrNotFound
	} else if err != nil {
		return DetachResult{}, err
	}
	var receiptIDToMaybeClean *string
	if txBefore.ReceiptImportID != nil && strings.TrimSpace(*txBefore.ReceiptImportID) != "" {
		rid := strings.TrimSpace(*txBefore.ReceiptImportID)
		receiptIDToMaybeClean = &rid
	}

	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).DetachReceiptFromTransaction(ctx, sqlcdb.DetachReceiptFromTransactionParams{
		UpdatedAt: now,
		ID:        txID,
		UserID:    userID,
	})
	if err != nil {
		return DetachResult{}, err
	}
	if n == 0 {
		return DetachResult{}, ErrNotAttached
	}

	if receiptIDToMaybeClean != nil {
		left, err := queries(db).CountReceiptLinkedTransactions(ctx, sqlcdb.CountReceiptLinkedTransactionsParams{
			UserID:          userID,
			ReceiptImportID: receiptIDToMaybeClean,
		})
		if err != nil {
			return DetachResult{}, err
		}
		if left == 0 {
			if err := queries(db).DeleteReceiptImportItemCategories(ctx, sqlcdb.DeleteReceiptImportItemCategoriesParams{
				UserID:          userID,
				ReceiptImportID: *receiptIDToMaybeClean,
			}); err != nil {
				return DetachResult{}, err
			}
		}
	}

	txRow, err := transaction.GetByID(ctx, db, userID, txID)
	if err != nil {
		return DetachResult{}, err
	}
	return DetachResult{Transaction: txRow}, nil
}
