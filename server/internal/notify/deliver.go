package notify

import (
	"context"
	"database/sql"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// BudgetThresholdChecker is wired from main to avoid an import cycle (budget -> notify -> budget).
var BudgetThresholdChecker func(ctx context.Context, db *sql.DB, userID string) error

// Deliver sends a notification on all enabled channels with deduplication.
func Deliver(ctx context.Context, db *sql.DB, settings sqlcdb.NotificationSetting, userID, triggerType, entityID, dedupDate, text string) {
	w := &Worker{DB: db}
	w.sendByChannels(ctx, sqlcdb.New(db), settings, userID, triggerType, entityID, dedupDate, text)
}

// UserFormatting returns locale, timezone and currency for a user.
func UserFormatting(ctx context.Context, db *sql.DB, userID string) (localeCode, timezone, currencyCode string, err error) {
	return userFormatting(ctx, db, userID)
}

// ResolveExternalURL returns the configured external base URL.
func ResolveExternalURL(ctx context.Context, db *sql.DB) string {
	return resolveExternalURL(ctx, db)
}

// ToTemplateMap indexes custom templates by trigger type.
func ToTemplateMap(items []sqlcdb.NotificationTemplate) map[string]*string {
	return toTemplateMap(items)
}
