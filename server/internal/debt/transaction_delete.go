package debt

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

var ErrLinkedTransactionProtected = errors.New("debt linked transaction cannot be deleted after settlement")

// GuardTransactionDelete blocks deleting the opening debt transaction when other linked operations exist.
func GuardTransactionDelete(ctx context.Context, db *sql.DB, userID, txID string) error {
	q := queries(db)
	link, err := q.GetDebtLinkByTransactionID(ctx, txID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if link.Role != "open" {
		return nil
	}
	if _, err := GetByID(ctx, db, userID, link.DebtID); err != nil {
		return err
	}
	n, err := q.CountDebtLinksByDebt(ctx, link.DebtID)
	if err != nil {
		return err
	}
	if n > 1 {
		return ErrLinkedTransactionProtected
	}
	return nil
}

// OnTransactionDelete updates or removes debt state before the transaction row is deleted.
func OnTransactionDelete(ctx context.Context, q *sqlcdb.Queries, userID, txID string) error {
	link, err := q.GetDebtLinkByTransactionID(ctx, txID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	switch link.Role {
	case "open":
		if err := q.DeleteDebtTransactionLink(ctx, txID); err != nil {
			return err
		}
		if _, err := q.DeleteDebt(ctx, sqlcdb.DeleteDebtParams{ID: link.DebtID, UserID: userID}); err != nil {
			return err
		}
		return nil
	case "settle":
		txRow, err := q.GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: txID, UserID: userID})
		if err != nil {
			return err
		}
		debtRow, err := q.GetDebtByID(ctx, sqlcdb.GetDebtByIDParams{ID: link.DebtID, UserID: userID})
		if err != nil {
			return err
		}
		othersSum, err := q.SumSettleTxAmountsByDebtExcluding(ctx, sqlcdb.SumSettleTxAmountsByDebtExcludingParams{
			DebtID:      link.DebtID,
			UserID:      userID,
			ExcludeTxID: txID,
		})
		if err != nil {
			return err
		}
		newAmount := recalcAmountAfterSettleDelete(debtRow.Amount, debtRow.IsSettled, othersSum, txRow.Amount)
		if err := q.DeleteDebtTransactionLink(ctx, txID); err != nil {
			return err
		}
		if debtRow.IsSettled == 1 {
			if _, err := q.ReopenDebt(ctx, sqlcdb.ReopenDebtParams{ID: link.DebtID, UserID: userID}); err != nil {
				return err
			}
		}
		if _, err := q.UpdateDebtAmount(ctx, sqlcdb.UpdateDebtAmountParams{
			Amount: newAmount, ID: link.DebtID, UserID: userID,
		}); err != nil {
			return err
		}
		return nil
	default:
		return nil
	}
}

func recalcAmountAfterSettleDelete(currentAmount, isSettled, othersSettleSum, deletedAmount int64) int64 {
	allSettleSum := othersSettleSum + deletedAmount
	var initialAmount int64
	if isSettled == 1 {
		initialAmount = allSettleSum
	} else {
		initialAmount = currentAmount + allSettleSum
	}
	return initialAmount - othersSettleSum
}
