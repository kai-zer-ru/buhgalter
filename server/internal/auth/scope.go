package auth

import (
	"context"
	"database/sql"
	"errors"
)

func withScope(ctx context.Context, db *sql.DB, info AuthInfo) AuthInfo {
	info.Scope = LoadScope(ctx, db, info.User.ID)
	return info
}

func LoadScope(ctx context.Context, db *sql.DB, userID string) Scope {
	scope := Scope{UserID: userID}
	row, err := queries(db).GetHouseholdByUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return scope
		}
		return scope
	}
	scope.HouseholdID = row.ID
	scope.PrincipalID = row.PrincipalUserID
	scope.Role = row.Role
	ids, err := queries(db).ListHouseholdMemberIDs(ctx, row.ID)
	if err == nil && len(ids) > 0 {
		scope.MemberIDs = ids
	}
	return scope
}
