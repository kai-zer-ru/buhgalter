package db_test

import (
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestMigrationsApply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "migrate.db")

	sqlDB, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	configured, err := db.IsConfigured(sqlDB)
	if err != nil {
		t.Fatalf("configured: %v", err)
	}
	if configured {
		t.Fatal("expected not configured on fresh db")
	}

	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM system_settings WHERE id = 1`).Scan(&count); err != nil {
		t.Fatalf("settings row: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 settings row, got %d", count)
	}
}
