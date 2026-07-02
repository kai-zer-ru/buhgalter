package account

import (
	"context"
	"database/sql"
)

// AfterAutoTopupConfigured is wired from main to evaluate auto-topup after settings change.
var AfterAutoTopupConfigured func(ctx context.Context, db *sql.DB, userID, accountID string)
