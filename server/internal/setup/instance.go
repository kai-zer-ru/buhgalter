package setup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

const markerFileName = ".configured"

func markerPath(dataDir string) string {
	return filepath.Join(dataDir, markerFileName)
}

// IsConfigured reports whether first-run setup has completed on this instance.
// The marker lives in the data directory and is not replaced when restoring a DB backup.
func IsConfigured(dataDir string) bool {
	_, err := os.Stat(markerPath(dataDir))
	return err == nil
}

func MarkConfigured(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	return os.WriteFile(markerPath(dataDir), []byte("1\n"), 0o644)
}

// SyncMarkerFromDB creates the marker for deployments that were configured before
// the marker file existed (is_configured in SQLite only).
func SyncMarkerFromDB(dataDir string, sqlDB *sql.DB) error {
	if IsConfigured(dataDir) {
		return nil
	}
	configured, err := db.IsConfigured(sqlDB)
	if err != nil || !configured {
		return err
	}
	return MarkConfigured(dataDir)
}
