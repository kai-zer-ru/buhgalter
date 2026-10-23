package store

import (
	"context"
	"database/sql"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/merchant"
)

// SyncFromMerchants ensures a stores row exists for each merchant name and backfills
// store_id on expenses that only have merchant_id (for filters and stats by store).
func SyncFromMerchants(ctx context.Context, db *sql.DB, userID string) error {
	merchants, err := merchant.List(ctx, db, userID)
	if err != nil {
		return err
	}
	for _, m := range merchants {
		if _, err := GetOrCreate(ctx, db, userID, m.Name); err != nil {
			return err
		}
	}
	q := queries(db)
	rows, err := q.ListExpenseTransactionsMissingStoreWithMerchant(ctx, userID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range rows {
		storeID, err := GetOrCreate(ctx, db, userID, row.MerchantName)
		if err != nil {
			return err
		}
		if storeID == nil {
			continue
		}
		if err := q.SetTransactionStoreID(ctx, sqlcdb.SetTransactionStoreIDParams{
			StoreID:   storeID,
			UpdatedAt: now,
			ID:        row.ID,
			UserID:    userID,
		}); err != nil {
			return err
		}
	}
	return nil
}
