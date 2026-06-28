package credit

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type repairCreditRow struct {
	id, userID, issueDate, paymentInterval string
	principal, monthlyPayment              int64
	termMonths                             int
	interestRate                           float64
	addedRetroactively                     bool
}

// RepairShortSchedules appends missing scheduled payments for credits created with the
// old schedule generator that stopped early when principal was covered.
func RepairShortSchedules(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT id, user_id, principal_amount, issue_date, term_months, interest_rate,
		       payment_interval, monthly_payment, added_retroactively
		FROM credits
		WHERE status = 'active' AND payment_interval != 'manual'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var credits []repairCreditRow
	for rows.Next() {
		var r repairCreditRow
		var addedRetro int64
		if err := rows.Scan(
			&r.id, &r.userID, &r.principal, &r.issueDate, &r.termMonths, &r.interestRate,
			&r.paymentInterval, &r.monthlyPayment, &addedRetro,
		); err != nil {
			return err
		}
		r.addedRetroactively = addedRetro == 1
		credits = append(credits, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, c := range credits {
		if err := repairCreditSchedule(ctx, db, c); err != nil {
			return err
		}
	}
	return nil
}

func repairCreditSchedule(ctx context.Context, db *sql.DB, c repairCreditRow) error {
	payments, err := queries(db).ListCreditPayments(ctx, c.id)
	if err != nil {
		return err
	}
	if len(payments) >= c.termMonths {
		return nil
	}

	issueDate, err := timeutil.ParseUTC(c.issueDate)
	if err != nil {
		return err
	}
	entries, err := GenerateAutoSchedule(
		c.principal, c.termMonths, c.monthlyPayment,
		PaymentInterval(c.paymentInterval), issueDate, c.interestRate,
	)
	if err != nil {
		return err
	}
	if len(payments) >= len(entries) {
		return nil
	}

	tz, err := userTimezone(ctx, db, c.userID)
	if err != nil {
		return err
	}
	todayStart, err := timeutil.TodayStartUTC(tz, timeutil.NowUTC())
	if err != nil {
		return err
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := queries(tx)
	var retroPaid int64
	for _, e := range entries[len(payments):] {
		payDate, err := timeutil.ParseUTC(e.PaymentDate)
		if err != nil {
			return err
		}
		kind := "scheduled"
		applied := int64(0)
		excludeStats := int64(0)
		if c.addedRetroactively && payDate.Before(todayStart) {
			kind = "retroactive"
			applied = 1
			excludeStats = 1
			retroPaid += e.Amount
		}
		if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
			ID: uuid.NewString(), CreditID: c.id, TransactionID: nil,
			Amount: e.Amount, PaymentDate: e.PaymentDate, Kind: kind,
			IsApplied: applied, ExcludeFromStats: excludeStats, CreatedAt: nowStr,
		}); err != nil {
			return err
		}
	}

	if retroPaid > 0 {
		creditRow, err := q.GetCreditByID(ctx, sqlcdb.GetCreditByIDParams{ID: c.id, UserID: c.userID})
		if err != nil {
			return err
		}
		newPaid := creditRow.PaidAmount + retroPaid
		if newPaid > c.principal {
			newPaid = c.principal
		}
		if newPaid != creditRow.PaidAmount {
			if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
				PaidAmount: newPaid, UpdatedAt: nowStr, ID: c.id, UserID: c.userID,
			}); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
