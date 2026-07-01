package notify

import (
	"context"
	"database/sql"
	"strings"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

// ResolveSecretKey returns the notification encryption key from system_settings only.
func ResolveSecretKey(ctx context.Context, db *sql.DB) (string, error) {
	raw, err := sqlcdb.New(db).GetNotificationSecretKey(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(raw), nil
}

// SecretKeyConfigured reports whether notification tokens can be encrypted.
func SecretKeyConfigured(ctx context.Context, db *sql.DB) bool {
	secret, err := ResolveSecretKey(ctx, db)
	if err != nil || secret == "" {
		return false
	}
	_, err = NewSecretBox(secret)
	return err == nil
}
