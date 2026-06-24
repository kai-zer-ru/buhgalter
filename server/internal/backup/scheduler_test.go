package backup_test

import (
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/backup"
)

func TestShouldRunAt(t *testing.T) {
	settings := backup.Settings{BackupEnabled: true, BackupTime: "03:00"}
	now := time.Date(2026, 6, 23, 3, 0, 0, 0, time.Local)

	ok, key := backup.ShouldRunAt(settings, now, "")
	if !ok || key == "" {
		t.Fatal("expected scheduled run")
	}

	ok, _ = backup.ShouldRunAt(settings, now, key)
	if ok {
		t.Fatal("should not run twice in same slot")
	}

	ok, _ = backup.ShouldRunAt(settings, now.Add(time.Hour), key)
	if ok {
		t.Fatal("should not run at different time")
	}

	ok, _ = backup.ShouldRunAt(backup.Settings{BackupEnabled: false, BackupTime: "03:00"}, now, "")
	if ok {
		t.Fatal("disabled backup should not run")
	}
}
