package credit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func init() {
	db.RegisterOpenHook(func(ctx context.Context, sqlDB *sql.DB) error {
		if err := RepairShortSchedules(ctx, sqlDB); err != nil {
			return fmt.Errorf("repair credit schedules: %w", err)
		}
		return nil
	})
}
