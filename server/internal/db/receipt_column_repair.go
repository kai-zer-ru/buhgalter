package db

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/pressly/goose/v3"
)

// recoverDuplicateReceiptColumn records 057 or 059 when SQLite reports
// "duplicate column name". Those ALTERs are plain (this SQLite has no
// ADD COLUMN IF NOT EXISTS) and a scan_chek database may already have the columns.
// Returns the original error when the failure is anything else.
func recoverDuplicateReceiptColumn(db *sql.DB, cause error) error {
	if !strings.Contains(cause.Error(), "duplicate column name") {
		return cause
	}
	next, err := nextPendingMigration(db)
	if err != nil {
		return cause
	}
	base := filepath.Base(next.Source)
	if base != "057_receipt_api_token.sql" && base != "059_stores_and_receipts.sql" {
		return cause
	}
	if err := replaySkippingDuplicateColumns(db, base); err != nil {
		return fmt.Errorf("replay %s: %w", base, err)
	}
	if _, err := db.Exec(
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, 1)`,
		next.Version,
	); err != nil {
		return fmt.Errorf("record migration %d: %w", next.Version, err)
	}
	log.Printf("goose: %s already had the column, recorded version %d", base, next.Version)
	return nil
}

func nextPendingMigration(db *sql.DB) (*goose.Migration, error) {
	current, err := goose.GetDBVersion(db)
	if err != nil {
		return nil, err
	}
	migs, err := goose.CollectMigrations("migrations", current, 1<<62)
	if err != nil {
		return nil, err
	}
	if len(migs) == 0 {
		return nil, fmt.Errorf("no pending migration")
	}
	return migs[0], nil
}

func replaySkippingDuplicateColumns(db *sql.DB, filename string) error {
	body, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return err
	}
	for _, stmt := range gooseUpStatements(string(body)) {
		if _, err := db.Exec(stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return err
		}
	}
	return nil
}

func gooseUpStatements(file string) []string {
	up := file
	if i := strings.Index(up, "-- +goose Up"); i >= 0 {
		up = up[i+len("-- +goose Up"):]
	}
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	var stmts []string
	for _, part := range strings.Split(up, ";") {
		stmt := strings.TrimSpace(part)
		if stmt == "" || onlySQLComments(stmt) {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return stmts
}

func onlySQLComments(stmt string) bool {
	for _, line := range strings.Split(stmt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		return false
	}
	return true
}
