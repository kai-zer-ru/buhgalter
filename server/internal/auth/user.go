package auth

import (
	"context"
	"database/sql"
	"errors"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

type User struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
	Status      string `json:"status"`
	Language    string `json:"language"`
	Currency    string `json:"currency"`
	Timezone    string `json:"timezone"`
	Theme       string `json:"theme"`
}

func LoadUser(ctx context.Context, db *sql.DB, userID string) (*User, error) {
	row, err := queries(db).GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return userFromIDRow(row), nil
}

func LoadUserByLogin(ctx context.Context, db *sql.DB, login string) (*User, string, error) {
	row, err := queries(db).GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("user not found")
		}
		return nil, "", err
	}
	return userFromLoginRow(row), row.PasswordHash, nil
}

func UpdateUserProfile(ctx context.Context, db sqlcdb.DBTX, userID, displayName, language, currency, timezone, theme string) error {
	return queries(db).UpdateUserProfile(ctx, sqlcdb.UpdateUserProfileParams{
		DisplayName: optionalString(displayName),
		Language:    language,
		Currency:    currency,
		Timezone:    timezone,
		Theme:       theme,
		ID:          userID,
	})
}
