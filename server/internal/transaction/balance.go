package transaction

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/debt"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/scope"
)

type AccountBalance struct {
	ID                        string  `json:"id"`
	Name                      string  `json:"name"`
	Type                      string  `json:"type"`
	Scope                     string  `json:"scope,omitempty"`
	Currency                  string  `json:"currency"`
	BankIcon                  *string `json:"bank_icon,omitempty"`
	Balance                   int64   `json:"balance"`
	BalanceDisplay            string  `json:"balance_display"`
	ForecastBalance           int64   `json:"forecast_balance"`
	ForecastDisplay           string  `json:"forecast_display"`
	HasFutureThisMonth        bool    `json:"has_future_this_month"`
	HasPlannedThisMonth       bool    `json:"has_planned_this_month"`
	HasSubscriptionsThisMonth bool    `json:"has_subscriptions_this_month"`
	IsPrimary                 bool    `json:"is_primary"`
	CreditLimit               *int64  `json:"credit_limit,omitempty"`
	CreditLimitDisplay        *string `json:"credit_limit_display,omitempty"`
	AutoTopupEnabled          bool    `json:"auto_topup_enabled"`
	AutoTopupThreshold        *int64  `json:"auto_topup_threshold,omitempty"`
	AutoTopupThresholdDisplay *string `json:"auto_topup_threshold_display,omitempty"`
	AutoTopupTarget           *int64  `json:"auto_topup_target,omitempty"`
	AutoTopupTargetDisplay    *string `json:"auto_topup_target_display,omitempty"`
	AutoTopupSourceAccountID  *string `json:"auto_topup_source_account_id,omitempty"`
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

type CreditCardsByCurrency struct {
	CreditCardsSummary
	Currency string `json:"currency"`
}

type TotalsByCurrency struct {
	Currency             string `json:"currency"`
	TotalBalance         int64  `json:"total_balance"`
	TotalForecast        int64  `json:"total_forecast"`
	TotalBalanceDisplay  string `json:"total_balance_display"`
	TotalForecastDisplay string `json:"total_forecast_display"`
}

type AccountsSummary struct {
	Accounts              []AccountBalance        `json:"accounts"`
	TotalBalance          int64                   `json:"total_balance"`
	TotalForecast         int64                   `json:"total_forecast"`
	TotalsByCurrency      []TotalsByCurrency      `json:"totals_by_currency"`
	CreditCardsSummary    *CreditCardsSummary     `json:"credit_cards_summary,omitempty"`
	CreditCardsByCurrency []CreditCardsByCurrency `json:"credit_cards_by_currency"`
}

type Dashboard struct {
	TotalBalance          int64                   `json:"total_balance"`
	TotalForecast         int64                   `json:"total_forecast"`
	TotalsByCurrency      []TotalsByCurrency      `json:"totals_by_currency"`
	InboxCount            int64                   `json:"inbox_count"`
	CreditCardsSummary    *CreditCardsSummary     `json:"credit_cards_summary,omitempty"`
	CreditCardsByCurrency []CreditCardsByCurrency `json:"credit_cards_by_currency"`
	Accounts              []AccountBalance        `json:"accounts"`
	RecentTransactions    []Transaction           `json:"recent_transactions"`
	DebtsSummary          debt.Summary            `json:"debts_summary"`
}

// Balance returns stored current balance for an account.
func Balance(ctx context.Context, db *sql.DB, userID, accountID string, initialBalance int64) (int64, error) {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, OwnerIds: []string{userID}})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return initialBalance, nil
		}
		return 0, err
	}
	_ = initialBalance
	return row.CurrentBalance, nil
}

// ForecastBalance adds future transactions and pending scheduled charges
// (subscriptions + recurring) in the current month (user timezone) to balance.
func ForecastBalance(ctx context.Context, db *sql.DB, userID, accountID string, balance int64) (int64, bool, error) {
	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return 0, false, err
	}
	forecasts, err := accountbalance.ForecastsByUser(ctx, db, userID, tz, map[string]int64{accountID: balance})
	if err != nil {
		return 0, false, err
	}
	fc := forecasts[accountID]
	return fc.Balance, fc.HasFutureThisMonth, nil
}

func accountBalanceFromRow(
	id, name, accType, currency string,
	bankIcon *string,
	balance int64,
	isPrimary int64,
	creditLimit *int64,
	autoTopupEnabled int64,
	autoTopupThreshold, autoTopupTarget *int64,
	autoTopupSourceAccountID *string,
) AccountBalance {
	ab := AccountBalance{
		ID:                       id,
		Name:                     name,
		Type:                     accType,
		Currency:                 money.ProfileOrDefault(currency),
		BankIcon:                 bankIcon,
		Balance:                  balance,
		BalanceDisplay:           money.FormatRubles(balance),
		IsPrimary:                isPrimary != 0,
		CreditLimit:              creditLimit,
		AutoTopupEnabled:         autoTopupEnabled != 0,
		AutoTopupThreshold:       autoTopupThreshold,
		AutoTopupTarget:          autoTopupTarget,
		AutoTopupSourceAccountID: autoTopupSourceAccountID,
	}
	if creditLimit != nil {
		s := money.FormatRubles(*creditLimit)
		ab.CreditLimitDisplay = &s
	}
	if autoTopupThreshold != nil {
		s := money.FormatRubles(*autoTopupThreshold)
		ab.AutoTopupThresholdDisplay = &s
	}
	if autoTopupTarget != nil {
		s := money.FormatRubles(*autoTopupTarget)
		ab.AutoTopupTargetDisplay = &s
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
	accRow, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, OwnerIds: []string{userID}})
	if err != nil {
		return AccountBalance{}, err
	}
	ab := accountBalanceFromRow(
		accountID, accountName, accountType, accRow.Currency, bankIcon, bal, accRow.IsPrimary, creditLimit,
		accRow.AutoTopupEnabled, accRow.AutoTopupThreshold, accRow.AutoTopupTarget, accRow.AutoTopupSourceAccountID,
	)
	if accRow.UserID != userID {
		ab.Scope = scope.Household
	} else {
		ab.Scope = scope.Personal
	}
	ab.ForecastBalance = fc.Balance
	ab.ForecastDisplay = money.FormatRubles(fc.Balance)
	ab.HasFutureThisMonth = fc.HasFutureThisMonth
	ab.HasPlannedThisMonth = fc.HasPlannedThisMonth
	ab.HasSubscriptionsThisMonth = fc.HasSubscriptionsThisMonth
	return ab, nil
}

func AccountsSummaryForUser(ctx context.Context, db *sql.DB, userID string) (AccountsSummary, error) {
	return AccountsSummaryForOwners(ctx, db, userID, []string{userID})
}

func AccountsSummaryForOwners(ctx context.Context, db *sql.DB, tzUserID string, ownerIDs []string) (AccountsSummary, error) {
	if len(ownerIDs) == 0 {
		return AccountsSummary{Accounts: []AccountBalance{}}, nil
	}
	rows, err := queries(db).ListAccountsByUserActive(ctx, ownerIDs)
	if err != nil {
		return AccountsSummary{}, err
	}
	balances := make(map[string]int64, len(rows))
	out := make([]AccountBalance, 0, len(rows))
	for _, row := range rows {
		balances[row.ID] = row.CurrentBalance
		ab := accountBalanceFromRow(
			row.ID, row.Name, row.Type, row.Currency, row.BankIcon, row.CurrentBalance, row.IsPrimary, row.CreditLimit,
			row.AutoTopupEnabled, row.AutoTopupThreshold, row.AutoTopupTarget, row.AutoTopupSourceAccountID,
		)
		if row.UserID != tzUserID {
			ab.Scope = scope.Household
		} else {
			ab.Scope = scope.Personal
		}
		ab.ForecastBalance = row.CurrentBalance
		ab.ForecastDisplay = money.FormatRubles(row.CurrentBalance)
		out = append(out, ab)
	}
	tz, err := userTimezone(ctx, db, tzUserID)
	if err != nil {
		return AccountsSummary{}, err
	}
	forecasts := map[string]accountbalance.Forecast{}
	for _, oid := range ownerIDs {
		part, err := accountbalance.ForecastsByUser(ctx, db, oid, tz, balances)
		if err != nil {
			return AccountsSummary{}, err
		}
		for id, fc := range part {
			forecasts[id] = fc
		}
	}
	for i := range out {
		if fc, ok := forecasts[out[i].ID]; ok {
			out[i].ForecastBalance = fc.Balance
			out[i].ForecastDisplay = money.FormatRubles(fc.Balance)
			out[i].HasFutureThisMonth = fc.HasFutureThisMonth
			out[i].HasPlannedThisMonth = fc.HasPlannedThisMonth
			out[i].HasSubscriptionsThisMonth = fc.HasSubscriptionsThisMonth
		}
	}
	profile, err := account.ProfileCurrency(ctx, db, tzUserID)
	if err != nil {
		return AccountsSummary{}, err
	}
	totals, totalBal, totalForecast := totalsByCurrency(out, profile)
	cardsByCur, cardsProfile := creditCardsByCurrency(out, profile)
	if cardsByCur == nil {
		cardsByCur = []CreditCardsByCurrency{}
	}
	return AccountsSummary{
		Accounts:              out,
		TotalBalance:          totalBal,
		TotalForecast:         totalForecast,
		TotalsByCurrency:      totals,
		CreditCardsSummary:    cardsProfile,
		CreditCardsByCurrency: cardsByCur,
	}, nil
}

func totalsByCurrency(accounts []AccountBalance, profile string) ([]TotalsByCurrency, int64, int64) {
	type acc struct{ bal, forecast int64 }
	by := map[string]acc{}
	for _, a := range accounts {
		if account.IsCreditCard(a.Type) {
			continue
		}
		cur := money.ProfileOrDefault(a.Currency)
		v := by[cur]
		v.bal += a.Balance
		v.forecast += a.ForecastBalance
		by[cur] = v
	}
	codes := make([]string, 0, len(by))
	for c := range by {
		codes = append(codes, c)
	}
	order := money.OrderPresentWithProfileFirst(profile, codes)
	out := make([]TotalsByCurrency, 0, len(order))
	for _, c := range order {
		v := by[c]
		out = append(out, TotalsByCurrency{
			Currency:             c,
			TotalBalance:         v.bal,
			TotalForecast:        v.forecast,
			TotalBalanceDisplay:  money.FormatRubles(v.bal),
			TotalForecastDisplay: money.FormatRubles(v.forecast),
		})
	}
	prof := by[profile]
	return out, prof.bal, prof.forecast
}

func creditCardsSummaryForCurrency(accounts []AccountBalance, currency string) *CreditCardsSummary {
	var count int
	var totalBal, totalForecast, totalLimit int64
	for _, acc := range accounts {
		if !account.IsCreditCard(acc.Type) {
			continue
		}
		if currency != "" && money.ProfileOrDefault(acc.Currency) != currency {
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

func creditCardsByCurrency(accounts []AccountBalance, profile string) ([]CreditCardsByCurrency, *CreditCardsSummary) {
	seen := map[string]struct{}{}
	codes := make([]string, 0)
	for _, acc := range accounts {
		if !account.IsCreditCard(acc.Type) {
			continue
		}
		c := money.ProfileOrDefault(acc.Currency)
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		codes = append(codes, c)
	}
	order := money.OrderPresentWithProfileFirst(profile, codes)
	out := make([]CreditCardsByCurrency, 0, len(order))
	for _, c := range order {
		sum := creditCardsSummaryForCurrency(accounts, c)
		if sum == nil {
			continue
		}
		out = append(out, CreditCardsByCurrency{CreditCardsSummary: *sum, Currency: c})
	}
	return out, creditCardsSummaryForCurrency(accounts, profile)
}

func DashboardForUser(ctx context.Context, db *sql.DB, userID string) (Dashboard, error) {
	return DashboardForOwners(ctx, db, userID, []string{userID})
}

func DashboardForOwners(ctx context.Context, db *sql.DB, tzUserID string, ownerIDs []string) (Dashboard, error) {
	summary, err := AccountsSummaryForOwners(ctx, db, tzUserID, ownerIDs)
	if err != nil {
		return Dashboard{}, err
	}
	recent, err := ListRecentOwned(ctx, db, ownerIDs, tzUserID, 10)
	if err != nil {
		return Dashboard{}, err
	}
	if recent == nil {
		recent = []Transaction{}
	}
	debtsSummary, err := debt.SummaryForOwners(ctx, db, tzUserID, ownerIDs)
	if err != nil {
		return Dashboard{}, err
	}
	inboxCount, err := CountInboxOwned(ctx, db, ownerIDs)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		TotalBalance:          summary.TotalBalance,
		TotalForecast:         summary.TotalForecast,
		TotalsByCurrency:      summary.TotalsByCurrency,
		InboxCount:            inboxCount,
		CreditCardsSummary:    summary.CreditCardsSummary,
		CreditCardsByCurrency: summary.CreditCardsByCurrency,
		Accounts:              summary.Accounts,
		RecentTransactions:    recent,
		DebtsSummary:          debtsSummary,
	}, nil
}
