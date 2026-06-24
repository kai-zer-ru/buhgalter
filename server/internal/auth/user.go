package auth

import (
	"context"
	"database/sql"
	"errors"
)

type User struct {
	ID          string `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"display_name"`
	IsAdmin     bool   `json:"is_admin"`
	Language    string `json:"language"`
	Currency    string `json:"currency"`
	Timezone    string `json:"timezone"`
	Theme       string `json:"theme"`
}

func LoadUser(ctx context.Context, db *sql.DB, userID string) (*User, error) {
	var u User
	var isAdmin int
	err := db.QueryRowContext(ctx, `
		SELECT id, login, COALESCE(display_name, ''), is_admin, language, currency, timezone, theme
		FROM users WHERE id = ?`, userID,
	).Scan(&u.ID, &u.Login, &u.DisplayName, &isAdmin, &u.Language, &u.Currency, &u.Timezone, &u.Theme)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	return &u, nil
}

func LoadUserByLogin(ctx context.Context, db *sql.DB, login string) (*User, string, error) {
	var u User
	var isAdmin int
	var passwordHash string
	err := db.QueryRowContext(ctx, `
		SELECT id, login, COALESCE(display_name, ''), is_admin, language, currency, timezone, theme, password_hash
		FROM users WHERE login = ?`, login,
	).Scan(&u.ID, &u.Login, &u.DisplayName, &isAdmin, &u.Language, &u.Currency, &u.Timezone, &u.Theme, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("user not found")
		}
		return nil, "", err
	}
	u.IsAdmin = isAdmin == 1
	return &u, passwordHash, nil
}
