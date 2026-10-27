package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

const HouseholdLoginPrefix = "household:"

var ErrReservedLogin = errors.New("reserved login prefix")

func IsReservedLogin(login string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(login)), HouseholdLoginPrefix)
}

func CreateUser(ctx context.Context, db sqlcdb.DBTX, login, passwordHash, displayName string, isAdmin bool, status UserStatus) (string, error) {
	if IsReservedLogin(login) {
		return "", ErrReservedLogin
	}
	if !status.Valid() {
		status = UserStatusActive
	}
	id := uuid.NewString()
	admin := int64(0)
	if isAdmin {
		admin = 1
	}
	if err := queries(db).InsertUser(ctx, sqlcdb.InsertUserParams{
		ID:           id,
		Login:        login,
		PasswordHash: passwordHash,
		DisplayName:  optionalString(displayName),
		IsAdmin:      admin,
		Status:       string(status),
	}); err != nil {
		return "", fmt.Errorf("insert user: %w", err)
	}
	if err := categoryseed.SeedDefaults(ctx, db, id); err != nil {
		return "", fmt.Errorf("seed categories: %w", err)
	}
	return id, nil
}
