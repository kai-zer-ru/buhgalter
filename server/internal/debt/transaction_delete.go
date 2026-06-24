package debt

import (
	"context"
	"database/sql"
	"errors"
)

var ErrLinkedTransactionProtected = errors.New("debt linked transaction cannot be deleted after settlement")

// GuardTransactionDelete blocks deleting debt-linked transactions once any settlement exists.
// Otherwise removing the opening transaction skews the account balance.
func GuardTransactionDelete(ctx context.Context, db *sql.DB, userID, txID string) error {
	q := queries(db)
	link, err := q.GetDebtLinkByTransactionID(ctx, txID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := GetByID(ctx, db, userID, link.DebtID); err != nil {
		return err
	}
	n, err := q.CountSettleLinksByDebt(ctx, link.DebtID)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrLinkedTransactionProtected
	}
	return nil
}
