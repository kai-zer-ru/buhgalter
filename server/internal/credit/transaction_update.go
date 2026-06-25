package credit

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// GuardTransactionUpdate blocks editing transactions linked to paid credit installments.
func GuardTransactionUpdate(ctx context.Context, db *sql.DB, userID, txID string) error {
	_, err := queries(db).GetCreditPaymentLinkByTransactionID(ctx, sqlcdb.GetCreditPaymentLinkByTransactionIDParams{
		TransactionID: &txID,
		UserID:        userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return ErrCannotEditPayment
}
