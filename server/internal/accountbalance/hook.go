package accountbalance

import (
	"context"
	"database/sql"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func init() {
	db.RegisterOpenHook(backfillHook)
}

func backfillHook(ctx context.Context, sqlDB *sql.DB) error {
	var count int
	err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pragma_table_info('accounts') WHERE name = 'current_balance'`).Scan(&count)
	if err != nil || count == 0 {
		return err
	}
	return BackfillAll(ctx, sqlDB)
}
