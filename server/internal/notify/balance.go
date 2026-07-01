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
	acc, err := b.q.GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{
		ID:     accountID,
		UserID: b.userID,
	})
	if err != nil {
		return 0, false
	}
	b.cache[accountID] = acc.CurrentBalance
	return acc.CurrentBalance, true
}

func formatWithBalanceShortfall(
	ctx context.Context,
	lookup balanceLookup,
	enabled bool,
	localeCode, currencyCode string,
	customMap map[string]*string,
	mainText string,
	accountID string,
	expenseAmount int64,
) (string, error) {
	if !enabled || expenseAmount <= 0 {
		return mainText, nil
	}
	balance, ok := lookup.currentBalance(ctx, accountID)
	if !ok || balance >= expenseAmount {
		return mainText, nil
	}
	shortfall := expenseAmount - balance
	suffix, err := Format(TriggerBalanceShortfall, localeCode, customMap[TriggerBalanceShortfall], FormatData{
		"amount": FormatAmountDisplay(shortfall, currencyCode),
	})
	if err != nil {
		return mainText, err
	}
	return mainText + ". " + suffix, nil
}
