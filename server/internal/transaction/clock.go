package transaction

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

var (
	ErrInvalidClock          = errors.New("invalid clock")
	ErrClockNotCreditPayment = errors.New("clock edit is only for a credit payment")
)

// parseClock accepts HH:MM or HH:MM:SS. Seconds are ignored.
func parseClock(value string) (hour, minute int, err error) {
	value = strings.TrimSpace(value)
	var parsed time.Time
	var parseErr error
	for _, layout := range []string{"15:04", "15:04:05"} {
		parsed, parseErr = time.Parse(layout, value)
		if parseErr == nil {
			return parsed.Hour(), parsed.Minute(), nil
		}
	}
	return 0, 0, ErrInvalidClock
}

// UpdateCreditPaymentClock writes hour and minute into transaction_date in the
// user's timezone. The calendar day, amount, accounts, and credit debit time stay.
func UpdateCreditPaymentClock(ctx context.Context, db *sql.DB, userID, id string, hour, minute int) (Transaction, error) {
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return Transaction{}, ErrInvalidClock
	}
	existing, err := GetByID(ctx, db, userID, id)
	if err != nil {
		return Transaction{}, err
	}
	_, err = queries(db).GetCreditPaymentLinkByTransactionID(ctx, sqlcdb.GetCreditPaymentLinkByTransactionIDParams{
		TransactionID: &id,
		UserID:        userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, ErrClockNotCreditPayment
	}
	if err != nil {
		return Transaction{}, err
	}

	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Transaction{}, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return Transaction{}, err
	}
	current, err := timeutil.ParseFlexibleUTC(existing.TransactionDate)
	if err != nil {
		return Transaction{}, err
	}
	local := current.In(loc)
	next := time.Date(local.Year(), local.Month(), local.Day(), hour, minute, 0, 0, loc)

	// kind stays manual: a system «Кредиты» row cannot become future.
	// The balance refresh still counts this applied payment later the same local day.
	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Transaction{}, err
	}
	defer func() { _ = dbTx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := queries(dbTx).UpdateTransactionDate(ctx, sqlcdb.UpdateTransactionDateParams{
		TransactionDate: timeutil.FormatUTC(next),
		UpdatedAt:       now,
		ID:              id,
		UserID:          userID,
	}); err != nil {
		return Transaction{}, err
	}
	if err := refreshBalancesOnTx(ctx, dbTx, userID, existing.AccountID); err != nil {
		return Transaction{}, err
	}
	if err := dbTx.Commit(); err != nil {
		return Transaction{}, err
	}
	notifyBalancesAfterCommit(ctx, db, userID, next, existing.AccountID)
	return GetByID(ctx, db, userID, id)
}
