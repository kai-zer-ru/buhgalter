package transaction

import (
	"context"
	"database/sql"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/balancehooks"
)

// AfterBalanceRefresh is deprecated; use balancehooks.AfterRefresh from main.
var AfterBalanceRefresh = balancehooks.NotifyRefresh

func refreshAccountBalances(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) error {
	if err := accountbalance.Refresh(ctx, db, userID, accountIDs...); err != nil {
		return err
	}
	balancehooks.NotifyRefresh(ctx, db, userID, accountIDs...)
	return nil
}

func uniqueAccountIDs(ids ...string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
