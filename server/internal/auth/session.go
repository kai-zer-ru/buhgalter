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
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
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

	if err := queries(db).InsertSession(ctx, sqlcdb.InsertSessionParams{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		IpAddress: optionalString(ip),
		UserAgent: optionalString(userAgent),
	}); err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return rawToken, nil
}

func LookupSession(ctx context.Context, db *sql.DB, rawToken string) (*Session, error) {
	if rawToken == "" {
		return nil, errors.New("empty token")
	}

	hash := HashToken(rawToken)
	row, err := queries(db).GetSessionByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	s, err := sessionFromRow(row.ID, row.UserID, row.LastActivity, row.ExpiresAt)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if now.After(s.ExpiresAt) || now.Sub(s.LastActivity) > IdleTimeout {
		_ = DeleteSessionByID(ctx, db, s.ID)
		return nil, errors.New("session expired")
	}

	_ = queries(db).TouchSession(ctx, sqlcdb.TouchSessionParams{
		ExpiresAt: now.Add(IdleTimeout).Format(time.RFC3339),
		ID:        s.ID,
	})

	return s, nil
}

// LookupSessionWithUser loads session and user in one query.
func LookupSessionWithUser(ctx context.Context, db *sql.DB, rawToken string) (*Session, *User, error) {
	if rawToken == "" {
		return nil, nil, errors.New("empty token")
	}

	hash := HashToken(rawToken)
	row, err := queries(db).GetSessionWithUser(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errors.New("session not found")
		}
		return nil, nil, err
	}

	s, err := sessionFromRow(row.ID, row.UserID, row.LastActivity, row.ExpiresAt)
	if err != nil {
		return nil, nil, err
	}
	u := userFromSessionRow(row)

	now := time.Now().UTC()
	if now.After(s.ExpiresAt) || now.Sub(s.LastActivity) > IdleTimeout {
		_ = DeleteSessionByID(ctx, db, s.ID)
		return nil, nil, errors.New("session expired")
	}

	_ = queries(db).TouchSession(ctx, sqlcdb.TouchSessionParams{
		ExpiresAt: now.Add(IdleTimeout).Format(time.RFC3339),
		ID:        s.ID,
	})

	return s, u, nil
}

func DeleteSessionByToken(ctx context.Context, db *sql.DB, rawToken string) error {
	hash := HashToken(rawToken)
	return queries(db).DeleteSessionByTokenHash(ctx, hash)
}

func DeleteSessionByID(ctx context.Context, db *sql.DB, id string) error {
	return queries(db).DeleteSessionByID(ctx, id)
}

func DeleteSessionsByUserID(ctx context.Context, db *sql.DB, userID string) error {
	return queries(db).DeleteSessionsByUserID(ctx, userID)
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
	row, err := queries(db).GetAPITokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("api token not found")
		}
		return "", err
	}
	if apiTokenExpired(row.ExpiresAt) {
		return "", errors.New("api token expired")
	}
	_ = queries(db).TouchAPITokenByID(ctx, row.ID)
	return row.UserID, nil
}

func verifyAPIToken(ctx context.Context, db *sql.DB, rawToken string) bool {
	hash := HashToken(rawToken)
	expiresAt, err := queries(db).GetAPITokenExpiresAt(ctx, hash)
	if err != nil {
		return false
	}
	if apiTokenExpired(expiresAt) {
		return false
	}
	_ = queries(db).TouchAPITokenByHash(ctx, hash)
	return true
}

func sessionFromRow(id, userID, lastActivity, expiresAt string) (*Session, error) {
	var s Session
	s.ID = id
	s.UserID = userID
	var err error
	s.LastActivity, err = parseTime(lastActivity)
	if err != nil {
		return nil, err
	}
	s.ExpiresAt, err = parseTime(expiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", value)
	}
	return t, err
}

func apiTokenExpired(expiresAt *string) bool {
	if expiresAt == nil || *expiresAt == "" {
		return false
	}
	t, err := parseTime(*expiresAt)
	if err != nil {
		return false
	}
	return time.Now().UTC().After(t)
}
