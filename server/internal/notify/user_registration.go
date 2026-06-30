package notify

import (
	"context"
	"database/sql"
	"strings"
	"time"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func NotifyAdminsOnUserRegistration(ctx context.Context, db *sql.DB, userID, login, displayName, registeredAt string) error {
	q := sqlcdb.New(db)
	secret, err := ResolveSecretKey(ctx, db)
	if err != nil {
		return nil
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		return nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id FROM users WHERE is_admin = 1`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var adminID string
		if err := rows.Scan(&adminID); err != nil {
			return err
		}
		if err := notifyAdminUserRegistration(ctx, db, q, box, adminID, userID, login, displayName, registeredAt); err != nil {
			return err
		}
	}
	return rows.Err()
}

func notifyAdminUserRegistration(ctx context.Context, db *sql.DB, q *sqlcdb.Queries, box *SecretBox, adminID, userID, login, displayName, registeredAt string) error {
	if err := q.EnsureNotificationSettings(ctx, adminID); err != nil {
		return nil
	}
	settings, err := q.GetNotificationSettings(ctx, adminID)
	if err != nil {
		return nil
	}
	if settings.TriggerUserRegistration != 1 {
		return nil
	}
	localeCode, timezone, currencyCode, err := userFormatting(ctx, db, adminID)
	if err != nil {
		return nil
	}
	templates, err := q.ListNotificationTemplates(ctx, adminID)
	if err != nil {
		return nil
	}
	customMap := toTemplateMap(templates)
	display := strings.TrimSpace(displayName)
	if display == "" {
		display = login
	}
	modURL := moderationURL(ctx, db, userID, localeCode)
	text, err := Format(TriggerUserRegistration, localeCode, customMap[TriggerUserRegistration], FormatData{
		"login":          login,
		"display_name":   display,
		"registered_at":  timeutil.FormatDisplayDateTimeShortInTimezone(registeredAt, timezone),
		"moderation_url": modURL,
		"amount":         FormatAmountDisplay(0, currencyCode),
	})
	if err != nil {
		return nil
	}
	now := time.Now()
	loc, err := time.LoadLocation(defaultTZ(timezone))
	if err != nil {
		loc = time.UTC
	}
	dateKey := now.In(loc).Format("2006-01-02")
	for _, channel := range activeChannels(settings) {
		exists, err := DedupExists(ctx, q, adminID, TriggerUserRegistration, channel, userID, dateKey)
		if err != nil || exists {
			continue
		}
		notifier, recipient, err := buildNotifier(settings, channel, box)
		if err != nil {
			_ = appendLog(ctx, q, adminID, TriggerUserRegistration, channel, &userID, &dateKey, "error", text)
			continue
		}
		if err := notifier.Send(ctx, recipient, text); err != nil {
			_ = appendLog(ctx, q, adminID, TriggerUserRegistration, channel, &userID, &dateKey, "error", text)
			continue
		}
		_ = appendLog(ctx, q, adminID, TriggerUserRegistration, channel, &userID, &dateKey, "sent", text)
	}
	return nil
}

func moderationURL(ctx context.Context, db *sql.DB, userID, localeCode string) string {
	externalURL, err := settingscache.ExternalURL(ctx, db)
	if err != nil || !externalURL.Valid || strings.TrimSpace(externalURL.String) == "" {
		if normalizeLocale(localeCode) == "en" {
			return "configure external URL in admin settings"
		}
		return "настройте внешний URL в админке"
	}
	base := strings.TrimRight(strings.TrimSpace(externalURL.String), "/")
	return base + "/admin/users?moderate=" + userID
}
