package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
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

// Balance returns stored current balance for an account.
func Balance(ctx context.Context, db *sql.DB, userID, accountID string, initialBalance int64) (int64, error) {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return initialBalance, nil
		}
		return 0, err
	}
	_ = initialBalance
	return row.CurrentBalance, nil
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
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return AccountBalance{}, err
	}
	forecasts, err := accountbalance.ForecastsByUser(ctx, db, userID, tz, map[string]int64{accountID: bal})
	if err != nil {
		return AccountBalance{}, err
	}
	fc := forecasts[accountID]
	return AccountBalance{
		ID:                 accountID,
		Name:               accountName,
		Type:               accountType,
		BankIcon:           bankIcon,
		Balance:            bal,
		BalanceDisplay:     money.FormatRubles(bal),
		ForecastBalance:    fc.Balance,
		ForecastDisplay:    money.FormatRubles(fc.Balance),
		HasFutureThisMonth: fc.HasFutureThisMonth,
	}, nil
}

func AccountsSummaryForUser(ctx context.Context, db *sql.DB, userID string) (AccountsSummary, error) {
	rows, err := queries(db).ListAccountsByUserActive(ctx, userID)
	if err != nil {
		return AccountsSummary{}, err
	}
	balances := make(map[string]int64, len(rows))
	out := make([]AccountBalance, 0, len(rows))
	for _, row := range rows {
		balances[row.ID] = row.CurrentBalance
		out = append(out, AccountBalance{
			ID:              row.ID,
			Name:            row.Name,
			Type:            row.Type,
			BankIcon:        row.BankIcon,
			Balance:         row.CurrentBalance,
			BalanceDisplay:  money.FormatRubles(row.CurrentBalance),
			ForecastBalance: row.CurrentBalance,
			ForecastDisplay: money.FormatRubles(row.CurrentBalance),
		})
	}
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return AccountsSummary{}, err
	}
	forecasts, err := accountbalance.ForecastsByUser(ctx, db, userID, tz, balances)
	if err != nil {
		return AccountsSummary{}, err
	}
	var totalBal, totalForecast int64
	for i := range out {
		if fc, ok := forecasts[out[i].ID]; ok {
			out[i].ForecastBalance = fc.Balance
			out[i].ForecastDisplay = money.FormatRubles(fc.Balance)
			out[i].HasFutureThisMonth = fc.HasFutureThisMonth
		}
		totalBal += out[i].Balance
		totalForecast += out[i].ForecastBalance
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
