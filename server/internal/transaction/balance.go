package transaction

import (
	"context"
	"database/sql"
	"fmt"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type AccountBalance struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Type               string  `json:"type"`
	BankIcon           *string `json:"bank_icon,omitempty"`
	Balance            int64   `json:"balance"`
	BalanceDisplay     string  `json:"balance_display"`
	ForecastBalance    int64   `json:"forecast_balance"`
	ForecastDisplay    string  `json:"forecast_display"`
	HasFutureThisMonth bool    `json:"has_future_this_month"`
}

type AccountsSummary struct {
	Accounts       []AccountBalance `json:"accounts"`
	TotalBalance   int64            `json:"total_balance"`
	TotalForecast  int64            `json:"total_forecast"`
}

type Dashboard struct {
	TotalBalance       int64            `json:"total_balance"`
	TotalForecast      int64            `json:"total_forecast"`
	Accounts           []AccountBalance `json:"accounts"`
	RecentTransactions []Transaction    `json:"recent_transactions"`
	DebtsSummary       debt.Summary     `json:"debts_summary"`
}

// Balance computes current balance for an account (manual transactions only, date <= now UTC).
func Balance(ctx context.Context, db *sql.DB, userID, accountID string, initialBalance int64) (int64, error) {
	now := timeutil.FormatUTC(timeutil.NowUTC())
	q := queries(db)

	income, err := sumInt64(q.SumIncomeManual(ctx, sqlcdb.SumIncomeManualParams{
		UserID: userID, AccountID: accountID, TransactionDate: now,
	}))
	if err != nil {
		return 0, err
	}
	expense, err := sumInt64(q.SumExpenseManual(ctx, sqlcdb.SumExpenseManualParams{
		UserID: userID, AccountID: accountID, TransactionDate: now,
	}))
	if err != nil {
		return 0, err
	}
	transferOut, err := sumInt64(q.SumTransferOutManual(ctx, sqlcdb.SumTransferOutManualParams{
		UserID: userID, AccountID: accountID, TransactionDate: now,
	}))
	if err != nil {
		return 0, err
	}
	transferIn, err := sumInt64(q.SumTransferInManual(ctx, sqlcdb.SumTransferInManualParams{
		UserID: userID, AccountID: accountID, TransactionDate: now,
	}))
	if err != nil {
		return 0, err
	}

	return initialBalance + income - expense + transferIn - transferOut, nil
}

// ForecastBalance adds future transactions in the current month (user timezone) to balance.
func ForecastBalance(ctx context.Context, db *sql.DB, userID, accountID string, balance int64) (int64, bool, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return 0, false, err
	}
	monthStart, monthEnd, err := timeutil.MonthBoundsUTC(tz, timeutil.NowUTC())
	if err != nil {
		return 0, false, err
	}
	q := queries(db)

	hasFuture, err := q.HasFutureInMonth(ctx, sqlcdb.HasFutureInMonthParams{
		UserID: userID, AccountID: accountID,
		TransactionDate: monthStart, TransactionDate_2: monthEnd,
	})
	if err != nil {
		return 0, false, err
	}

	fi, err := sumInt64(q.SumFutureIncomeInRange(ctx, sqlcdb.SumFutureIncomeInRangeParams{
		UserID: userID, AccountID: accountID,
		TransactionDate: monthStart, TransactionDate_2: monthEnd,
	}))
	if err != nil {
		return 0, false, err
	}
	fe, err := sumInt64(q.SumFutureExpenseInRange(ctx, sqlcdb.SumFutureExpenseInRangeParams{
		UserID: userID, AccountID: accountID,
		TransactionDate: monthStart, TransactionDate_2: monthEnd,
	}))
	if err != nil {
		return 0, false, err
	}
	ftOut, err := sumInt64(q.SumFutureTransferOutInRange(ctx, sqlcdb.SumFutureTransferOutInRangeParams{
		UserID: userID, AccountID: accountID,
		TransactionDate: monthStart, TransactionDate_2: monthEnd,
	}))
	if err != nil {
		return 0, false, err
	}
	ftIn, err := sumInt64(q.SumFutureTransferInInRange(ctx, sqlcdb.SumFutureTransferInInRangeParams{
		UserID: userID, AccountID: accountID,
		TransactionDate: monthStart, TransactionDate_2: monthEnd,
	}))
	if err != nil {
		return 0, false, err
	}

	forecast := balance + fi - fe + ftIn - ftOut
	return forecast, hasFuture > 0, nil
}

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

func EnrichAccountBalance(ctx context.Context, db *sql.DB, userID, accountID, accountName, accountType string, bankIcon *string, initialBalance int64) (AccountBalance, error) {
	bal, err := Balance(ctx, db, userID, accountID, initialBalance)
	if err != nil {
		return AccountBalance{}, err
	}
	forecast, hasFuture, err := ForecastBalance(ctx, db, userID, accountID, bal)
	if err != nil {
		return AccountBalance{}, err
	}
	return AccountBalance{
		ID:                 accountID,
		Name:               accountName,
		Type:               accountType,
		BankIcon:           bankIcon,
		Balance:            bal,
		BalanceDisplay:     money.FormatRubles(bal),
		ForecastBalance:    forecast,
		ForecastDisplay:    money.FormatRubles(forecast),
		HasFutureThisMonth: hasFuture,
	}, nil
}

func AccountsSummaryForUser(ctx context.Context, db *sql.DB, userID string) (AccountsSummary, error) {
	rows, err := queries(db).ListAccountsByUserActive(ctx, userID)
	if err != nil {
		return AccountsSummary{}, err
	}
	out := make([]AccountBalance, 0, len(rows))
	var totalBal, totalForecast int64
	for _, row := range rows {
		ab, err := EnrichAccountBalance(ctx, db, userID, row.ID, row.Name, row.Type, row.BankIcon, row.InitialBalance)
		if err != nil {
			return AccountsSummary{}, err
		}
		out = append(out, ab)
		totalBal += ab.Balance
		totalForecast += ab.ForecastBalance
	}
	return AccountsSummary{
		Accounts:      out,
		TotalBalance:  totalBal,
		TotalForecast: totalForecast,
	}, nil
}

func DashboardForUser(ctx context.Context, db *sql.DB, userID string) (Dashboard, error) {
	if err := ActivateDueFutureTransactions(ctx, db, userID); err != nil {
		return Dashboard{}, err
	}
	summary, err := AccountsSummaryForUser(ctx, db, userID)
	if err != nil {
		return Dashboard{}, err
	}
	recent, err := ListRecent(ctx, db, userID, 10)
	if err != nil {
		return Dashboard{}, err
	}
	if recent == nil {
		recent = []Transaction{}
	}
	debtsSummary, err := debt.SummaryForUser(ctx, db, userID)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		TotalBalance:       summary.TotalBalance,
		TotalForecast:      summary.TotalForecast,
		Accounts:           summary.Accounts,
		RecentTransactions: recent,
		DebtsSummary:       debtsSummary,
	}, nil
}
