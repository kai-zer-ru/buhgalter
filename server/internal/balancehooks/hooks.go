package balancehooks

import (
	"context"
	"database/sql"
)

// AfterRefresh is wired from main to evaluate auto-topup after balance changes.
var AfterRefresh func(ctx context.Context, db *sql.DB, userID string, accountIDs ...string)

// NotifyRefresh calls AfterRefresh when configured.
func NotifyRefresh(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) {
	if AfterRefresh != nil {
		AfterRefresh(ctx, db, userID, accountIDs...)
	}
}

// NotifyAll calls AfterRefresh for every enabled auto-topup beneficiary.
var NotifyAll func(ctx context.Context, db *sql.DB, userID string)

func NotifyAllRefresh(ctx context.Context, db *sql.DB, userID string) {
	if NotifyAll != nil {
		NotifyAll(ctx, db, userID)
	}
}
