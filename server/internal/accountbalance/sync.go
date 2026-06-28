package accountbalance

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func sumInt64(v interface{}, err error) (int64, error) {
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case float64:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("unexpected sum type %T", v)
	}
}

type Forecast struct {
	Balance            int64
	HasFutureThisMonth bool
}

func queries(db *sql.DB) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func computeDeltas(ctx context.Context, db *sql.DB, userID, cutoff string) (income, expense, transferOut, transferIn map[string]int64, err error) {
	q := queries(db)
	income = make(map[string]int64)
	expense = make(map[string]int64)
	transferOut = make(map[string]int64)
	transferIn = make(map[string]int64)

	rows, err := q.SumIncomeManualByUser(ctx, sqlcdb.SumIncomeManualByUserParams{
		UserID: userID, TransactionDate: cutoff,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, row := range rows {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		income[row.AccountID] = total
	}

	rows2, err := q.SumExpenseManualByUser(ctx, sqlcdb.SumExpenseManualByUserParams{
		UserID: userID, TransactionDate: cutoff,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, row := range rows2 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		expense[row.AccountID] = total
	}

	rows3, err := q.SumTransferOutManualByUser(ctx, sqlcdb.SumTransferOutManualByUserParams{
		UserID: userID, TransactionDate: cutoff,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, row := range rows3 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		transferOut[row.AccountID] = total
	}

	rows4, err := q.SumTransferInManualByUser(ctx, sqlcdb.SumTransferInManualByUserParams{
		UserID: userID, TransactionDate: cutoff,
	})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for _, row := range rows4 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		transferIn[row.AccountID] = total
	}
	return income, expense, transferOut, transferIn, nil
}

// ComputeAll returns current balances for every account of the user.
func ComputeAll(ctx context.Context, db *sql.DB, userID string) (map[string]int64, error) {
	cutoff := timeutil.FormatUTC(timeutil.NowUTC())
	accounts, err := queries(db).ListAllAccountIDsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	income, expense, transferOut, transferIn, err := computeDeltas(ctx, db, userID, cutoff)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(accounts))
	for _, acc := range accounts {
		out[acc.ID] = acc.InitialBalance +
			income[acc.ID] - expense[acc.ID] +
			transferIn[acc.ID] - transferOut[acc.ID]
	}
	return out, nil
}

// Refresh updates stored current_balance for the given accounts (or all if none specified).
func Refresh(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) error {
	computed, err := ComputeAll(ctx, db, userID)
	if err != nil {
		return err
	}
	target := accountIDs
	if len(target) == 0 {
		target = make([]string, 0, len(computed))
		for id := range computed {
			target = append(target, id)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	q := queries(db)
	for _, id := range target {
		bal, ok := computed[id]
		if !ok {
			continue
		}
		if err := q.UpdateAccountCurrentBalance(ctx, sqlcdb.UpdateAccountCurrentBalanceParams{
			CurrentBalance: bal,
			UpdatedAt:      now,
			ID:             id,
			UserID:         userID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// BackfillAll recomputes current_balance for every account in the database.
func BackfillAll(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT user_id FROM accounts`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, userID := range userIDs {
		if err := Refresh(ctx, db, userID); err != nil {
			return err
		}
	}
	return nil
}

// ForecastsByUser returns month forecast per account in one batch.
func ForecastsByUser(ctx context.Context, db *sql.DB, userID, tz string, balances map[string]int64) (map[string]Forecast, error) {
	monthStart, monthEnd, err := timeutil.MonthBoundsUTC(tz, timeutil.NowUTC())
	if err != nil {
		return nil, err
	}
	q := queries(db)
	out := make(map[string]Forecast, len(balances))
	for id, bal := range balances {
		out[id] = Forecast{Balance: bal, HasFutureThisMonth: false}
	}

	futureAccounts, err := q.AccountsWithFutureInMonth(ctx, sqlcdb.AccountsWithFutureInMonthParams{
		UserID: userID, TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return nil, err
	}
	futureSet := make(map[string]struct{}, len(futureAccounts))
	for _, id := range futureAccounts {
		futureSet[id] = struct{}{}
	}

	fi, fe, ftOut, ftIn := make(map[string]int64), make(map[string]int64), make(map[string]int64), make(map[string]int64)

	rows, err := q.SumFutureIncomeByUser(ctx, sqlcdb.SumFutureIncomeByUserParams{
		UserID: userID, TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, err
		}
		fi[row.AccountID] = total
	}

	rows2, err := q.SumFutureExpenseByUser(ctx, sqlcdb.SumFutureExpenseByUserParams{
		UserID: userID, TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows2 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, err
		}
		fe[row.AccountID] = total
	}

	rows3, err := q.SumFutureTransferOutByUser(ctx, sqlcdb.SumFutureTransferOutByUserParams{
		UserID: userID, TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows3 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, err
		}
		ftOut[row.AccountID] = total
	}

	rows4, err := q.SumFutureTransferInByUser(ctx, sqlcdb.SumFutureTransferInByUserParams{
		UserID: userID, TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows4 {
		total, err := sumInt64(row.Total, nil)
		if err != nil {
			return nil, err
		}
		ftIn[row.AccountID] = total
	}

	for id, base := range balances {
		forecast := base + fi[id] - fe[id] + ftIn[id] - ftOut[id]
		_, hasFuture := futureSet[id]
		out[id] = Forecast{Balance: forecast, HasFutureThisMonth: hasFuture}
	}
	return out, nil
}
