package credit

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// OnTransactionDelete unlinks a credit payment and rolls back paid_amount before the transaction row is removed.
func OnTransactionDelete(ctx context.Context, q *sqlcdb.Queries, userID, txID string) error {
	link, err := q.GetCreditPaymentLinkByTransactionID(ctx, sqlcdb.GetCreditPaymentLinkByTransactionIDParams{
		TransactionID: &txID,
		UserID:        userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	if link.PaymentKind == "early" {
		if _, err := q.DeleteCreditPaymentByID(ctx, sqlcdb.DeleteCreditPaymentByIDParams{
			ID:       link.PaymentID,
			CreditID: link.CreditID,
		}); err != nil {
			return err
		}
	} else {
		n, err := q.RevertCreditPaymentLink(ctx, sqlcdb.RevertCreditPaymentLinkParams{
			ID:            link.PaymentID,
			CreditID:      link.CreditID,
			TransactionID: &txID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}

	return adjustPaidAfterPaymentRemoved(ctx, q, userID, link.CreditID, link.PaidAmount, link.PaymentAmount, link.CreditStatus, nowStr)
}

// RemovePayment deletes a row from the credit schedule (and linked transaction if any).
func RemovePayment(ctx context.Context, db *sql.DB, userID, creditID, paymentID string) (Credit, error) {
	row, err := queries(db).GetCreditPaymentForUser(ctx, sqlcdb.GetCreditPaymentForUserParams{
		ID: paymentID, CreditID: creditID, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Credit{}, ErrNotFound
	}
	if err != nil {
		return Credit{}, err
	}
	if row.Kind == "retroactive" {
		return Credit{}, ErrCannotRemoveRetroactive
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	txID := row.TransactionID

	if row.IsApplied == 1 {
		if err := adjustPaidAfterPaymentRemoved(ctx, q, userID, creditID, row.PaidAmount, row.Amount, row.CreditStatus, nowStr); err != nil {
			return Credit{}, err
		}
	}

	if _, err := q.DeleteCreditPaymentByID(ctx, sqlcdb.DeleteCreditPaymentByIDParams{
		ID: paymentID, CreditID: creditID,
	}); err != nil {
		return Credit{}, err
	}

	if txID != nil {
		if err := q.ClearDebtTransactionLink(ctx, sqlcdb.ClearDebtTransactionLinkParams{
			TransactionID: txID, UserID: userID,
		}); err != nil {
			return Credit{}, err
		}
		if err := q.DeleteDebtTransactionLink(ctx, *txID); err != nil {
			return Credit{}, err
		}
		if _, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{ID: *txID, UserID: userID}); err != nil {
			return Credit{}, err
		}
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	return GetByID(ctx, db, userID, creditID, true)
}

func adjustPaidAfterPaymentRemoved(
	ctx context.Context,
	q *sqlcdb.Queries,
	userID, creditID string,
	paidAmount, paymentAmount int64,
	creditStatus, nowStr string,
) error {
	newPaid := paidAmount - paymentAmount
	if newPaid < 0 {
		newPaid = 0
	}
	if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
		PaidAmount: newPaid,
		UpdatedAt:  nowStr,
		ID:         creditID,
		UserID:     userID,
	}); err != nil {
		return err
	}
	if creditStatus == "closed" {
		_, err := q.ReopenCredit(ctx, sqlcdb.ReopenCreditParams{
			UpdatedAt: nowStr,
			ID:        creditID,
			UserID:    userID,
		})
		return err
	}
	return nil
}
