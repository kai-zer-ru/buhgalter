package transaction

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type ReceiptCreateInput struct {
	AccountID        string
	Amount           int64
	Description      *string
	CategoryID       *string
	SubcategoryID    *string
	TransactionDate  time.Time
	StoreID          *string
	ReceiptImportID  *string
	ReceiptItemIndex *int64
}

// CreateWithReceipt inserts an expense linked to a receipt inside an open transaction.
func CreateWithReceipt(ctx context.Context, db sqlcdb.DBTX, userID string, in ReceiptCreateInput) (string, error) {
	if in.Amount <= 0 {
		return "", ErrInvalidAmount
	}
	kind, err := resolveKindTx(ctx, db, userID, in.TransactionDate)
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(in.TransactionDate)
	if err := queries(db).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID:                id,
		UserID:            userID,
		AccountID:         in.AccountID,
		Type:              "expense",
		Kind:              kind,
		Amount:            in.Amount,
		Description:       in.Description,
		CategoryID:        in.CategoryID,
		SubcategoryID:     in.SubcategoryID,
		TransferGroupID:   nil,
		TransferAccountID: nil,
		StoreID:           in.StoreID,
		ReceiptImportID:   in.ReceiptImportID,
		ReceiptItemIndex:  in.ReceiptItemIndex,
		TransactionDate:   txDate,
		AffectsBalance:    1,
		SubscriptionID:    nil,
		MerchantID:        nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		return "", fmt.Errorf("insert transaction: %w", err)
	}
	return id, nil
}

func resolveKindTx(ctx context.Context, db sqlcdb.DBTX, userID string, txDate time.Time) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		tz = "Europe/Moscow"
	}
	future, err := timeutil.IsFutureInTZ(txDate, timeutil.NowUTC(), tz)
	if err != nil {
		return "", err
	}
	if future {
		return "future", nil
	}
	return "manual", nil
}

// RefreshAccountBalances updates cached balances after batch receipt import.
func RefreshAccountBalances(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) error {
	return refreshAccountBalances(ctx, db, userID, timeutil.NowUTC(), accountIDs...)
}

// ValidateActiveAccount checks account is active before receipt import.
func ValidateActiveAccount(ctx context.Context, db *sql.DB, userID, accountID string) error {
	return validateActiveAccount(ctx, db, userID, accountID)
}
