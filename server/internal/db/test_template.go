package db

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

var (
	testTemplateOnce sync.Once
	testTemplatePath string
	testTemplateErr  error
)

// prepareSchema ensures path has the current schema. In tests, a once-per-process
// migrated template is copied so each Open does not re-run all goose migrations
// (slow under Docker/act SQLite fsync, and looks like a migration loop in logs).
func prepareSchema(path string) error {
	if testing.Testing() {
		if _, err := os.Stat(path); err == nil {
			// Existing DB (e.g. Manager.Reopen after backup restore): migrate in place.
			return runMigrations(path)
		} else if !os.IsNotExist(err) {
			return err
		}
		return materializeFromTestTemplate(path)
	}
	return runMigrations(path)
}

func materializeFromTestTemplate(path string) error {
	src, err := ensureTestTemplate()
	if err != nil {
		return err
	}
	if err := copyFile(src, path); err != nil {
		return fmt.Errorf("copy test db template: %w", err)
	}
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
	return nil
}

func ensureTestTemplate() (string, error) {
	testTemplateOnce.Do(func() {
		dir, err := os.MkdirTemp("", "buhgalter-db-template-*")
		if err != nil {
			testTemplateErr = err
			return
		}
		path := filepath.Join(dir, "template.db")
		if err := runMigrations(path); err != nil {
			testTemplateErr = err
			return
		}
		sqlDB, err := sql.Open("sqlite", sqliteDSN(path, false))
		if err != nil {
			testTemplateErr = err
			return
		}
		if _, err := sqlDB.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			_ = sqlDB.Close()
			testTemplateErr = fmt.Errorf("checkpoint test template: %w", err)
			return
		}
		if err := sqlDB.Close(); err != nil {
			testTemplateErr = err
			return
		}
		_ = os.Remove(path + "-wal")
		_ = os.Remove(path + "-shm")
		testTemplatePath = path
	})
	return testTemplatePath, testTemplateErr
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
