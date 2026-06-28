package transaction

import (
	"context"
	"database/sql"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
)

func refreshAccountBalances(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) error {
	return accountbalance.Refresh(ctx, db, userID, accountIDs...)
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
