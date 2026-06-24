package credit

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

// repairMissingSchedule rebuilds credit_payments when the graph was lost (e.g. after a failed close).
func repairMissingSchedule(ctx context.Context, db *sql.DB, userID string, f creditFields) error {
	if f.status != "active" {
		return nil
	}
	if PaymentInterval(f.paymentInterval) == IntervalManual {
		return nil
	}
	count, err := queries(db).CountCreditPayments(ctx, f.id)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	issueDate, err := timeutil.ParseUTC(f.issueDate)
	if err != nil {
		return err
	}
	entries, err := GenerateAutoSchedule(
		f.principal, f.termMonths, f.monthlyPayment,
		PaymentInterval(f.paymentInterval), issueDate,
	)
	if err != nil {
		return err
	}

	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return err
	}
	todayStart, err := timeutil.TodayStartUTC(tz, timeutil.NowUTC())
	if err != nil {
		return err
	}
	addedRetro := f.addedRetro == 1

	nowStr := time.Now().UTC().Format(time.RFC3339)
	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	remainingPaid := f.paidAmount

	for _, e := range entries {
		payDate, err := timeutil.ParseUTC(e.PaymentDate)
		if err != nil {
			return err
		}
		kind := "scheduled"
		applied := int64(0)
		excludeStats := int64(0)

		if addedRetro && payDate.Before(todayStart) {
			kind = "retroactive"
			applied = 1
			excludeStats = 1
			if remainingPaid >= e.Amount {
				remainingPaid -= e.Amount
			}
		} else if remainingPaid >= e.Amount {
			applied = 1
			remainingPaid -= e.Amount
		}

		if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
			ID: uuid.NewString(), CreditID: f.id, TransactionID: nil,
			Amount: e.Amount, PaymentDate: e.PaymentDate, Kind: kind,
			IsApplied: applied, ExcludeFromStats: excludeStats, CreatedAt: nowStr,
		}); err != nil {
			return err
		}
	}

	return dbTx.Commit()
}

func computeFallbackNextPayment(f creditFields) (*string, *int64, error) {
	if f.status != "active" {
		return nil, nil, nil
	}
	if RemainingAmount(f.principal, f.paidAmount) <= 0 {
		return nil, nil, nil
	}
	interval := PaymentInterval(f.paymentInterval)
	if interval == IntervalManual || f.monthlyPayment <= 0 {
		return nil, nil, nil
	}

	issueDate, err := timeutil.ParseUTC(f.issueDate)
	if err != nil {
		return nil, nil, err
	}

	appliedCount := 0
	if f.monthlyPayment > 0 {
		appliedCount = int(f.paidAmount / f.monthlyPayment)
	}

	date := issueDate
	for i := 0; i <= appliedCount; i++ {
		date = nextPaymentDate(date, interval)
	}
	d := timeutil.FormatUTC(date)
	a := f.monthlyPayment
	return &d, &a, nil
}
