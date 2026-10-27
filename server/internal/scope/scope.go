package scope

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

var ErrNotFound = errors.New("not found")
var ErrNoHousehold = errors.New("not in household")
var ErrInvalidScope = errors.New("invalid scope")

// PrincipalForCreate maps body scope (personal|household) to the catalog owner.
func PrincipalForCreate(sc auth.Scope, catalog string) (string, error) {
	switch catalog {
	case "", Personal:
		if sc.UserID == "" {
			return "", ErrNotFound
		}
		return sc.UserID, nil
	case Household:
		if sc.PrincipalID == "" {
			return "", ErrNoHousehold
		}
		return sc.PrincipalID, nil
	default:
		return "", ErrInvalidScope
	}
}

const (
	Personal  = "personal"
	Household = "household"
	All       = "all"
)

func OwnerIDs(s auth.Scope) []string {
	return s.OwnerIDs()
}

// FilterOwnerIDs applies CAP-5 view: all | personal | household.
// household without a group yields an empty slice (callers skip the query).
func FilterOwnerIDs(s auth.Scope, view string) []string {
	switch view {
	case Personal:
		if s.UserID == "" {
			return nil
		}
		return []string{s.UserID}
	case Household:
		if s.PrincipalID == "" {
			return nil
		}
		return []string{s.PrincipalID}
	default:
		return s.OwnerIDs()
	}
}

func AccountScope(viewer auth.Scope, ownerUserID string) string {
	if viewer.PrincipalID != "" && ownerUserID == viewer.PrincipalID {
		return Household
	}
	return Personal
}

func PrincipalForAccount(ctx context.Context, db sqlcdb.DBTX, ownerIDs []string, accountID string) (string, error) {
	if len(ownerIDs) == 0 {
		return "", ErrNotFound
	}
	row, err := sqlcdb.New(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{
		ID:       accountID,
		OwnerIds: ownerIDs,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

func AffectedUsers(ctx context.Context, db sqlcdb.DBTX, ownerID string) ([]string, error) {
	if ownerID == "" {
		return nil, nil
	}
	ids, err := sqlcdb.New(db).ListHouseholdMemberIDsByPrincipal(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []string{ownerID}, nil
	}
	return ids, nil
}
