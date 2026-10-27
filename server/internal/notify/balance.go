package notify

import (
	"context"
	"strings"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type balanceLookup struct {
	q      *sqlcdb.Queries
	userID string
	cache  map[string]int64
}

func newBalanceLookup(q *sqlcdb.Queries, userID string) balanceLookup {
	return balanceLookup{
		q:      q,
		userID: userID,
		cache:  make(map[string]int64),
	}
}

func (b balanceLookup) currentBalance(ctx context.Context, accountID string) (int64, bool) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return 0, false
	}
	if value, ok := b.cache[accountID]; ok {
		return value, true
	}
	acc, err := b.q.GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, OwnerIds: []string{b.userID}})
	if err != nil {
		return 0, false
	}
	b.cache[accountID] = acc.CurrentBalance
	return acc.CurrentBalance, true
}

// formatWithBalanceShortfall appends the shortfall suffix when the account cannot cover expenseAmount.
// shortfallData is non-nil only when the suffix was applied; it carries shortfall-specific
// placeholders ({amount}, {account_url}) for the balance_shortfall keyboard zone so merge does not
// reuse the main trigger's amount.
func formatWithBalanceShortfall(
	ctx context.Context,
	lookup balanceLookup,
	enabled bool,
	localeCode, currencyCode string,
	customMap map[string]*string,
	mainText string,
	accountID string,
	expenseAmount int64,
	externalURL string,
) (text string, shortfallData FormatData, err error) {
	if !enabled || expenseAmount <= 0 {
		return mainText, nil, nil
	}
	balance, ok := lookup.currentBalance(ctx, accountID)
	if !ok || balance >= expenseAmount {
		return mainText, nil, nil
	}
	shortfall := expenseAmount - balance
	amountDisplay := FormatAmountDisplay(shortfall, currencyCode)
	accountURL := accountURLPlaceholderValue(externalURL, localeCode, accountID)
	data := FormatData{
		"amount":      amountDisplay,
		"account_url": accountURL,
	}
	suffix, err := Format(TriggerBalanceShortfall, localeCode, customMap[TriggerBalanceShortfall], data)
	if err != nil {
		return mainText, nil, err
	}
	return mainText + ". " + suffix, data, nil
}
