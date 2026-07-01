package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
)

type PasswordResetRequest struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

// RequestPasswordReset records a password reset request for an existing user.
// Returns nil when the login is unknown (caller should still respond with success).
func RequestPasswordReset(ctx context.Context, db *sql.DB, login string) error {
	login = strings.TrimSpace(login)
	if login == "" {
		return errors.New("login required")
	}

	user, _, err := LoadUserByLogin(ctx, db, login)
	if err != nil {
		if err.Error() == "user not found" {
			return nil
		}
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	requestID := uuid.NewString()
	_, err = db.ExecContext(ctx, `
		INSERT INTO password_reset_requests (id, user_id, created_at, dismissed_at)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(user_id) DO UPDATE SET
			created_at = excluded.created_at,
			dismissed_at = NULL`,
		requestID, user.ID, now,
	)
	if err != nil {
		return err
	}
	displayName := user.DisplayName
	_ = notify.NotifyAdminsOnPasswordReset(ctx, db, user.ID, user.Login, displayName, now, requestID)
	return nil
}

func ListPendingPasswordResetRequests(ctx context.Context, db *sql.DB) ([]PasswordResetRequest, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.user_id, u.login, COALESCE(u.display_name, ''), r.created_at
		FROM password_reset_requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.dismissed_at IS NULL
		ORDER BY r.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PasswordResetRequest, 0)
	for rows.Next() {
		var item PasswordResetRequest
		if err := rows.Scan(&item.ID, &item.UserID, &item.Login, &item.DisplayName, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func DismissPasswordResetRequest(ctx context.Context, db *sql.DB, requestID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.ExecContext(ctx, `
		UPDATE password_reset_requests
		SET dismissed_at = ?
		WHERE id = ? AND dismissed_at IS NULL`,
		now, requestID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("request not found")
	}
	return nil
}

func DismissPasswordResetRequestsForUser(ctx context.Context, db *sql.DB, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `
		UPDATE password_reset_requests
		SET dismissed_at = ?
		WHERE user_id = ? AND dismissed_at IS NULL`,
		now, userID,
	)
	return err
}

func SetUserPassword(ctx context.Context, db *sql.DB, userID, passwordHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, now, userID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("user not found")
	}
	return nil
}
