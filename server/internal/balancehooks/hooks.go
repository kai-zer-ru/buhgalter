package balancehooks

import (
	"context"
	"database/sql"
	"time"
)

// AfterRefresh is wired from main to evaluate auto-topup after balance changes.
// asOf is the trigger operation's transaction_date; zero means "now".
var AfterRefresh func(ctx context.Context, db *sql.DB, userID string, asOf time.Time, accountIDs ...string)

// NotifyRefresh calls AfterRefresh when configured.
func NotifyRefresh(ctx context.Context, db *sql.DB, userID string, asOf time.Time, accountIDs ...string) {
	if AfterRefresh != nil {
		AfterRefresh(ctx, db, userID, asOf, accountIDs...)
	}
}

// NotifyAll calls AfterRefresh for every enabled auto-topup beneficiary.
var NotifyAll func(ctx context.Context, db *sql.DB, userID string, asOf time.Time)

func NotifyAllRefresh(ctx context.Context, db *sql.DB, userID string, asOf time.Time) {
	if NotifyAll != nil {
		NotifyAll(ctx, db, userID, asOf)
	}
}
