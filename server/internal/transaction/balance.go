package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
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
	CreditLimit        *int64  `json:"credit_limit,omitempty"`
	CreditLimitDisplay *string `json:"credit_limit_display,omitempty"`
}

type CreditCardsSummary struct {
	TotalBalance         int64  `json:"total_balance"`
	TotalForecast        int64  `json:"total_forecast"`
	TotalLimit           int64  `json:"total_limit"`
	Count                int    `json:"count"`
	TotalBalanceDisplay  string `json:"total_balance_display"`
	TotalForecastDisplay string `json:"total_forecast_display"`
	TotalLimitDisplay    string `json:"total_limit_display"`
}

type AccountsSummary struct {
	Accounts      []AccountBalance `json:"accounts"`
	TotalBalance  int64            `json:"total_balance"`
	TotalForecast int64            `json:"total_forecast"`
}

type Dashboard struct {
	TotalBalance       int64               `json:"total_balance"`
	TotalForecast      int64               `json:"total_forecast"`
	CreditCardsSummary *CreditCardsSummary `json:"credit_cards_summary,omitempty"`
	Accounts           []AccountBalance    `json:"accounts"`
	RecentTransactions []Transaction       `json:"recent_transactions"`
	DebtsSummary       debt.Summary        `json:"debts_summary"`
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

func accountBalanceFromRow(
	id, name, accType string,
	bankIcon *string,
	balance int64,
	creditLimit *int64,
) AccountBalance {
	ab := AccountBalance{
		ID:             id,
		Name:           name,
		Type:           accType,
		BankIcon:       bankIcon,
		Balance:        balance,
		BalanceDisplay: money.FormatRubles(balance),
		CreditLimit:    creditLimit,
	}
	if creditLimit != nil {
		s := money.FormatRubles(*creditLimit)
		ab.CreditLimitDisplay = &s
	}
	return ab
}

func EnrichAccountBalance(ctx context.Context, db *sql.DB, userID, accountID, accountName, accountType string, bankIcon *string, initialBalance int64, creditLimit *int64) (AccountBalance, error) {
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
	ab := accountBalanceFromRow(accountID, accountName, accountType, bankIcon, bal, creditLimit)
	ab.ForecastBalance = fc.Balance
	ab.ForecastDisplay = money.FormatRubles(fc.Balance)
	ab.HasFutureThisMonth = fc.HasFutureThisMonth
	return ab, nil
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
		ab := accountBalanceFromRow(row.ID, row.Name, row.Type, row.BankIcon, row.CurrentBalance, row.CreditLimit)
		ab.ForecastBalance = row.CurrentBalance
		ab.ForecastDisplay = money.FormatRubles(row.CurrentBalance)
		out = append(out, ab)
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
		if account.IsCreditCard(out[i].Type) {
			continue
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

func creditCardsSummaryFromAccounts(accounts []AccountBalance) *CreditCardsSummary {
	var count int
	var totalBal, totalForecast, totalLimit int64
	for _, acc := range accounts {
		if !account.IsCreditCard(acc.Type) {
			continue
		}
		count++
		totalBal += acc.Balance
		totalForecast += acc.ForecastBalance
		if acc.CreditLimit != nil {
			totalLimit += *acc.CreditLimit
		}
	}
	if count == 0 {
		return nil
	}
	return &CreditCardsSummary{
		TotalBalance:         totalBal,
		TotalForecast:        totalForecast,
		TotalLimit:           totalLimit,
		Count:                count,
		TotalBalanceDisplay:  money.FormatRubles(totalBal),
		TotalForecastDisplay: money.FormatRubles(totalForecast),
		TotalLimitDisplay:    money.FormatRubles(totalLimit),
	}
}

func DashboardForUser(ctx context.Context, db *sql.DB, userID string) (Dashboard, error) {
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
		CreditCardsSummary: creditCardsSummaryFromAccounts(summary.Accounts),
		Accounts:           summary.Accounts,
		RecentTransactions: recent,
		DebtsSummary:       debtsSummary,
	}, nil
}
