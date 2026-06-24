package notify

import (
	"context"
	"database/sql"
	"strings"
)

// ResolveSecretKey returns the notification encryption key from system_settings only.
func ResolveSecretKey(ctx context.Context, db *sql.DB) (string, error) {
	var raw string
	if err := db.QueryRowContext(ctx, `SELECT notification_secret_key FROM system_settings WHERE id = 1`).Scan(&raw); err != nil {
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
