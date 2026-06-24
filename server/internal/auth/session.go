package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	SessionCookieName = "session"
	SessionTokenBytes = 32
	IdleTimeout       = 10 * 24 * time.Hour
)

type Session struct {
	ID           string
	UserID       string
	LastActivity time.Time
	ExpiresAt    time.Time
}

func GenerateToken() (raw string, hash string, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = HashToken(raw)
	return raw, hash, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func CreateSession(ctx context.Context, db *sql.DB, userID, ip, userAgent string) (rawToken string, err error) {
	rawToken, tokenHash, err := GenerateToken()
	if err != nil {
		return "", err
	}

	id := uuid.NewString()
	expiresAt := time.Now().UTC().Add(IdleTimeout)

	_, err = db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, last_activity, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, datetime('now'), ?, ?, ?)`,
		id, userID, tokenHash, expiresAt.UTC().Format(time.RFC3339), nullStr(ip), nullStr(userAgent),
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return rawToken, nil
}

func LookupSession(ctx context.Context, db *sql.DB, rawToken string) (*Session, error) {
	if rawToken == "" {
		return nil, errors.New("empty token")
	}

	hash := HashToken(rawToken)
	var s Session
	var lastActivity, expiresAt string
	err := db.QueryRowContext(ctx, `
		SELECT id, user_id, last_activity, expires_at
		FROM sessions WHERE token_hash = ?`, hash,
	).Scan(&s.ID, &s.UserID, &lastActivity, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	s.LastActivity, err = time.Parse(time.RFC3339, lastActivity)
	if err != nil {
		s.LastActivity, _ = time.Parse("2006-01-02 15:04:05", lastActivity)
	}
	s.ExpiresAt, err = time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		s.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAt)
	}

	now := time.Now().UTC()
	if now.After(s.ExpiresAt) || now.Sub(s.LastActivity) > IdleTimeout {
		_ = DeleteSessionByID(ctx, db, s.ID)
		return nil, errors.New("session expired")
	}

	_, _ = db.ExecContext(ctx, `
		UPDATE sessions
		SET last_activity = datetime('now'),
		    expires_at = ?
		WHERE id = ?`,
		now.Add(IdleTimeout).Format(time.RFC3339), s.ID,
	)

	return &s, nil
}

func DeleteSessionByToken(ctx context.Context, db *sql.DB, rawToken string) error {
	hash := HashToken(rawToken)
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hash)
	return err
}

func DeleteSessionByID(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func VerifyToken(ctx context.Context, db *sql.DB, rawToken string) bool {
	if _, err := LookupSession(ctx, db, rawToken); err == nil {
		return true
	}
	return verifyAPIToken(ctx, db, rawToken)
}

func LookupAPIToken(ctx context.Context, db *sql.DB, rawToken string) (userID string, err error) {
	if rawToken == "" {
		return "", errors.New("empty token")
	}
	hash := HashToken(rawToken)
	var tokenID string
	var expiresAt sql.NullString
	err = db.QueryRowContext(ctx, `
		SELECT id, user_id, expires_at FROM api_tokens WHERE token_hash = ?`, hash,
	).Scan(&tokenID, &userID, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("api token not found")
		}
		return "", err
	}
	if expiresAt.Valid && expiresAt.String != "" {
		t, parseErr := time.Parse(time.RFC3339, expiresAt.String)
		if parseErr != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", expiresAt.String)
		}
		if time.Now().UTC().After(t) {
			return "", errors.New("api token expired")
		}
	}
	_, _ = db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = datetime('now') WHERE id = ?`, tokenID)
	return userID, nil
}

func verifyAPIToken(ctx context.Context, db *sql.DB, rawToken string) bool {
	hash := HashToken(rawToken)
	var expiresAt sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT expires_at FROM api_tokens WHERE token_hash = ?`, hash,
	).Scan(&expiresAt)
	if err != nil {
		return false
	}
	if expiresAt.Valid && expiresAt.String != "" {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil {
			t, _ = time.Parse("2006-01-02 15:04:05", expiresAt.String)
		}
		if time.Now().UTC().After(t) {
			return false
		}
	}
	_, _ = db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = datetime('now') WHERE token_hash = ?`, hash)
	return true
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
