package auth

import (
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func userFromIDRow(row sqlcdb.GetUserByIDRow) *User {
	return &User{
		ID:          row.ID,
		Login:       row.Login,
		DisplayName: row.DisplayName,
		IsAdmin:     row.IsAdmin == 1,
		Status:      row.Status,
		Language:    row.Language,
		Currency:    row.Currency,
		Timezone:    row.Timezone,
		Theme:       row.Theme,
	}
}

func userFromLoginRow(row sqlcdb.GetUserByLoginRow) *User {
	return &User{
		ID:          row.ID,
		Login:       row.Login,
		DisplayName: row.DisplayName,
		IsAdmin:     row.IsAdmin == 1,
		Status:      row.Status,
		Language:    row.Language,
		Currency:    row.Currency,
		Timezone:    row.Timezone,
		Theme:       row.Theme,
	}
}

func userFromSessionRow(row sqlcdb.GetSessionWithUserRow) *User {
	return &User{
		ID:          row.UserID,
		Login:       row.Login,
		DisplayName: row.DisplayName,
		IsAdmin:     row.IsAdmin == 1,
		Status:      row.Status,
		Language:    row.Language,
		Currency:    row.Currency,
		Timezone:    row.Timezone,
		Theme:       row.Theme,
	}
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
