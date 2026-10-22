package backup_test

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func testBackupService(t *testing.T) (*backup.Service, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", "buhgalter.db")
	mgr, err := db.NewManager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return &backup.Service{Manager: mgr, BackupDir: filepath.Join(dir, "backups")}, dbPath
}

func TestRestoreRemovesBakAfterSuccessfulReopen(t *testing.T) {
	svc, dbPath := testBackupService(t)
	if _, err := svc.Manager.DB().Exec(`CREATE TABLE restore_probe(v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Manager.DB().Exec(`INSERT INTO restore_probe VALUES ('orig')`); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "src.db")
	if err := svc.Manager.VacuumInto(src); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Manager.DB().Exec(`INSERT INTO restore_probe VALUES ('after')`); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := svc.Restore(f); err != nil {
		t.Fatal(err)
	}

	var orig string
	if err := svc.Manager.DB().QueryRow(`SELECT v FROM restore_probe WHERE v = 'orig'`).Scan(&orig); err != nil {
		t.Fatalf("orig row: %v", err)
	}
	err = svc.Manager.DB().QueryRow(`SELECT v FROM restore_probe WHERE v = 'after'`).Scan(new(string))
	if err != sql.ErrNoRows {
		t.Fatalf("after-row want ErrNoRows, got %v", err)
	}
	if _, err := os.Stat(dbPath + ".bak"); !os.IsNotExist(err) {
		t.Fatalf(".bak should be removed after successful reopen, stat=%v", err)
	}
}

func TestRestoreKeepsPreviousDBWhenReopenFails(t *testing.T) {
	svc, dbPath := testBackupService(t)
	if _, err := svc.Manager.DB().Exec(`CREATE TABLE restore_probe(v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Manager.DB().Exec(`INSERT INTO restore_probe VALUES ('keep')`); err != nil {
		t.Fatal(err)
	}

	err := svc.Restore(bytes.NewReader([]byte("this is not a sqlite database")))
	if err == nil {
		t.Fatal("expected reopen error for garbage restore")
	}

	var keep string
	if qerr := svc.Manager.DB().QueryRow(`SELECT v FROM restore_probe WHERE v = 'keep'`).Scan(&keep); qerr != nil {
		t.Fatalf("previous db should still open after failed restore: %v (restore err: %v)", qerr, err)
	}
	if _, sterr := os.Stat(dbPath + ".bak"); !os.IsNotExist(sterr) {
		t.Fatalf(".bak leftover after rollback, stat=%v", sterr)
	}
}
