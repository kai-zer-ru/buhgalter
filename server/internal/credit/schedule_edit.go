package credit

import (
	"context"
	"database/sql"
	"errors"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type ScheduleAmountUpdate struct {
	PaymentID string
	Amount    int64
}

var ErrCannotEditPayment = errors.New("cannot edit payment")

// UpdateScheduleAmounts changes amounts on pending scheduled payments only.
func UpdateScheduleAmounts(ctx context.Context, db *sql.DB, userID, creditID string, updates []ScheduleAmountUpdate) (Credit, error) {
	if len(updates) == 0 {
		return Credit{}, ErrInvalidAmount
	}

	existing, err := GetByID(ctx, db, userID, creditID, false)
	if err != nil {
		return Credit{}, err
	}
	if existing.Status == "closed" {
		return Credit{}, ErrAlreadyClosed
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	for _, u := range updates {
		if u.Amount <= 0 {
			return Credit{}, ErrInvalidAmount
		}
		row, err := q.GetCreditPaymentByID(ctx, sqlcdb.GetCreditPaymentByIDParams{
			ID: u.PaymentID, CreditID: creditID,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return Credit{}, ErrNotFound
		}
		if err != nil {
			return Credit{}, err
		}
		if row.IsApplied == 1 || row.Kind != "scheduled" {
			return Credit{}, ErrCannotEditPayment
		}
		if row.Amount == u.Amount {
			continue
		}

		n, err := q.UpdateScheduledCreditPaymentAmount(ctx, sqlcdb.UpdateScheduledCreditPaymentAmountParams{
			Amount: u.Amount, ID: u.PaymentID, CreditID: creditID,
		})
		if err != nil {
			return Credit{}, err
		}
		if n == 0 {
			return Credit{}, ErrCannotEditPayment
		}

		if row.TransactionID != nil {
			if _, err := q.UpdateFutureTransactionAmount(ctx, sqlcdb.UpdateFutureTransactionAmountParams{
				Amount: u.Amount, UpdatedAt: nowStr, ID: *row.TransactionID, UserID: userID,
			}); err != nil {
				return Credit{}, err
			}
		}
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	return GetByID(ctx, db, userID, creditID, true)
}
