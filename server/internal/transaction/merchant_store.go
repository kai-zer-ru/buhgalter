package transaction

import (
	"context"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// resolveStoreIDForExpense returns the store id the caller already has.
// A merchant on a manual expense does not create a stores row.
func resolveStoreIDForExpense(
	_ context.Context,
	_ sqlcdb.DBTX,
	_, _ string,
	_ *string,
	storeID *string,
	_ *string,
) (*string, error) {
	return storeID, nil
}
