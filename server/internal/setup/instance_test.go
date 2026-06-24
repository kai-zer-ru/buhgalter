package setup_test

import (
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/setup"
)

func TestInstanceMarker(t *testing.T) {
	dir := t.TempDir()

	if setup.IsConfigured(dir) {
		t.Fatal("expected not configured")
	}

	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	if err := setup.SyncMarkerFromDB(dir, mgr.DB()); err != nil {
		t.Fatal(err)
	}
	if setup.IsConfigured(dir) {
		t.Fatal("marker should not exist before setup")
	}

	if err := setup.MarkConfigured(dir); err != nil {
		t.Fatal(err)
	}
	if !setup.IsConfigured(dir) {
		t.Fatal("expected configured after marker write")
	}

	if err := setup.SyncMarkerFromDB(dir, mgr.DB()); err != nil {
		t.Fatal(err)
	}
	if !setup.IsConfigured(dir) {
		t.Fatal("sync must not remove marker")
	}
}
