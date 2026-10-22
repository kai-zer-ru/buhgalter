package db_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestOpenSetsSynchronousOffInTests(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	var sync string
	if err := sqlDB.QueryRow(`PRAGMA synchronous`).Scan(&sync); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	// OFF is reported as 0 by SQLite.
	if sync != "0" && sync != "OFF" {
		t.Fatalf("expected synchronous=OFF in tests, got %q", sync)
	}
}

func TestOpenReusesMigratedTemplate(t *testing.T) {
	dir := t.TempDir()
	first, err := db.Open(filepath.Join(dir, "a.db"))
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = first.Close()

	start := time.Now()
	second, err := db.Open(filepath.Join(dir, "b.db"))
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	_ = second.Close()

	// Template copy should be far cheaper than re-running all migrations.
	if elapsed > 3*time.Second {
		t.Fatalf("second open too slow (%v); expected template reuse", elapsed)
	}
}
