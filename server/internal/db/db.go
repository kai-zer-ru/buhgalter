package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := categoryseed.BackfillSystemCategories(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("backfill system categories: %w", err)
	}

	if err := syncDBPath(db, path); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

func syncDBPath(db *sql.DB, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	_, err = db.Exec(`UPDATE system_settings SET db_path = ? WHERE id = 1`, abs)
	return err
}

func IsConfigured(db *sql.DB) (bool, error) {
	var configured int
	err := db.QueryRow(`SELECT is_configured FROM system_settings WHERE id = 1`).Scan(&configured)
	if err != nil {
		return false, err
	}
	return configured == 1, nil
}
