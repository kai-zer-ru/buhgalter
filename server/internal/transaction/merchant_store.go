package transaction

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/store"
)

func resolveStoreIDForExpense(
	ctx context.Context,
	db sqlcdb.DBTX,
	userID, txType string,
	merchantID *string,
	storeID *string,
	receiptImportID *string,
) (*string, error) {
	if txType != "expense" {
		return storeID, nil
	}
	if receiptImportID != nil && strings.TrimSpace(*receiptImportID) != "" {
		return storeID, nil
	}
	if storeID != nil && strings.TrimSpace(*storeID) != "" {
		return storeID, nil
	}
	if merchantID == nil || strings.TrimSpace(*merchantID) == "" {
		return storeID, nil
	}
	row, err := queries(db).GetMerchantByID(ctx, sqlcdb.GetMerchantByIDParams{ID: *merchantID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return storeID, nil
	}
	if err != nil {
		return nil, err
	}
	return store.GetOrCreate(ctx, db, userID, row.Name)
}
