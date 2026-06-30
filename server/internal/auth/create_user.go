package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
)

func CreateUser(ctx context.Context, db *sql.DB, login, passwordHash, displayName string, isAdmin bool, status UserStatus) (string, error) {
	if !status.Valid() {
		status = UserStatusActive
	}
	id := uuid.NewString()
	admin := 0
	if isAdmin {
		admin = 1
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, login, password_hash, display_name, is_admin, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, login, passwordHash, displayName, admin, string(status),
	)
	if err != nil {
		return "", fmt.Errorf("insert user: %w", err)
	}
	if err := categoryseed.SeedDefaults(ctx, db, id); err != nil {
		return "", fmt.Errorf("seed categories: %w", err)
	}
	return id, nil
}
