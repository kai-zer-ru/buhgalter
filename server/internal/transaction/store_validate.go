package transaction

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kai-zer-ru/buhgalter/internal/store"
)

var ErrStoreExpenseOnly = errors.New("store only allowed for expense")

func validateStoreForType(txType string, storeID *string) error {
	if storeID == nil || *storeID == "" {
		return nil
	}
	if txType != "expense" {
		return ErrStoreExpenseOnly
	}
	return nil
}

func validateStoreOwner(ctx context.Context, db *sql.DB, userID, storeID string) error {
	if err := store.ValidateForUser(ctx, db, userID, storeID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return ErrInvalidAccount
		}
		return err
	}
	return nil
}
