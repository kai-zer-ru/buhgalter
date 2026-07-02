package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
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
	if err := queries(db).UpsertPasswordResetRequest(ctx, sqlcdb.UpsertPasswordResetRequestParams{
		ID:        requestID,
		UserID:    user.ID,
		CreatedAt: now,
	}); err != nil {
		return err
	}
	displayName := user.DisplayName
	_ = notify.NotifyAdminsOnPasswordReset(ctx, db, user.ID, user.Login, displayName, now, requestID)
	return nil
}

func ListPendingPasswordResetRequests(ctx context.Context, db *sql.DB) ([]PasswordResetRequest, error) {
	rows, err := queries(db).ListPendingPasswordResetRequests(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]PasswordResetRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, PasswordResetRequest{
			ID:          row.ID,
			UserID:      row.UserID,
			Login:       row.Login,
			DisplayName: row.DisplayName,
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil
}

func DismissPasswordResetRequest(ctx context.Context, db *sql.DB, requestID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).DismissPasswordResetRequest(ctx, sqlcdb.DismissPasswordResetRequestParams{
		DismissedAt: &now,
		ID:          requestID,
	})
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
	return queries(db).DismissPasswordResetRequestsForUser(ctx, sqlcdb.DismissPasswordResetRequestsForUserParams{
		DismissedAt: &now,
		UserID:      userID,
	})
}

func SetUserPassword(ctx context.Context, db *sql.DB, userID, passwordHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	n, err := queries(db).UpdateUserPassword(ctx, sqlcdb.UpdateUserPasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    now,
		ID:           userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("user not found")
	}
	return nil
}
