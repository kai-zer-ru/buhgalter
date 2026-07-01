package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	if err := runMigrations(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", sqliteDSN(path, true))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := categoryseed.BackfillSystemCategories(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("backfill system categories: %w", err)
	}

	if err := runOpenHooks(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := syncDBPath(db, path); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func sqliteDSN(path string, foreignKeys bool) string {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
		path,
	)
	if foreignKeys {
		dsn += "&_pragma=foreign_keys(1)"
	}
	return dsn
}

// runMigrations opens a short-lived connection without FK enforcement in the DSN.
// Rebuild migrations (e.g. 033) need DROP TABLE accounts while child rows still exist;
// PRAGMA foreign_keys=OFF inside SQL does not override _pragma=foreign_keys(1) on modernc.
func runMigrations(path string) error {
	db, err := sql.Open("sqlite", sqliteDSN(path, false))
	if err != nil {
		return fmt.Errorf("open sqlite for migrate: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping sqlite for migrate: %w", err)
	}
	if err := migrate(db); err != nil {
		return err
	}
	return nil
}

func migrate(db *sql.DB) error {
	if err := recoverInterruptedAccountRebuild(db); err != nil {
		return fmt.Errorf("recover accounts rebuild: %w", err)
	}
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

// recoverInterruptedAccountRebuild handles a partial run of migration 033:
//   - accounts missing, accounts_new present → finish rename (data kept);
//   - both present → drop stale accounts_new so 033 can run again.
func recoverInterruptedAccountRebuild(db *sql.DB) error {
	ctx := context.Background()
	q := sqlcdb.New(db)
	accountsCnt, err := countSqliteTable(ctx, q, "accounts")
	if err != nil {
		return err
	}
	stagingCnt, err := countSqliteTable(ctx, q, "accounts_new")
	if err != nil {
		return err
	}
	if accountsCnt == 0 && stagingCnt == 1 {
		return q.RenameAccountsNewToAccounts(ctx)
	}
	if accountsCnt == 1 && stagingCnt == 1 {
		return q.DropAccountsNewTable(ctx)
	}
	return nil
}

func countSqliteTable(ctx context.Context, q *sqlcdb.Queries, name string) (int64, error) {
	return q.CountSqliteTable(ctx, &name)
}

func syncDBPath(db *sql.DB, path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if err := sqlcdb.New(db).UpdateDBPath(context.Background(), abs); err != nil {
		return err
	}
	return nil
}

func IsConfigured(db *sql.DB) (bool, error) {
	configured, err := sqlcdb.New(db).GetIsConfigured(context.Background())
	if err != nil {
		return false, err
	}
	return configured == 1, nil
}
