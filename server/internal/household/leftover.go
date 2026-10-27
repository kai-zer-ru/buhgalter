package household

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func principalLeftoverTable(ctx context.Context, db *sql.Tx, principalID string) (string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	for _, table := range tables {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'user_id'`, table).Scan(&n); err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		if table == "household_members" {
			continue
		}
		q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE user_id = ?", quoteIdent(table))
		var c int
		if err := db.QueryRowContext(ctx, q, principalID).Scan(&c); err != nil {
			return "", err
		}
		if c > 0 {
			return table, nil
		}
	}
	return "", nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
