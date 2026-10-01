package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReopenKeepsHandleWhenOpenFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live.db")
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	if _, err := mgr.DB().Exec(`CREATE TABLE reopen_probe(v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.DB().Exec(`INSERT INTO reopen_probe VALUES ('keep')`); err != nil {
		t.Fatal(err)
	}
	old := mgr.DB()

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(path) })

	if err := mgr.Reopen(); err == nil {
		t.Fatal("expected Reopen error when path is a directory")
	}
	if mgr.DB() != old {
		t.Fatal("failed Reopen must keep the previous handle")
	}
	if err := old.Ping(); err != nil {
		t.Fatalf("previous handle should stay live: %v", err)
	}
	var v string
	if err := mgr.DB().QueryRow(`SELECT v FROM reopen_probe`).Scan(&v); err != nil || v != "keep" {
		t.Fatalf("probe row: v=%q err=%v", v, err)
	}
}

func TestReopenReplacesHandleOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "live.db")
	mgr, err := NewManager(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	if _, err := mgr.DB().Exec(`CREATE TABLE reopen_probe(v TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.DB().Exec(`INSERT INTO reopen_probe VALUES ('ok')`); err != nil {
		t.Fatal(err)
	}
	old := mgr.DB()
	if err := mgr.Reopen(); err != nil {
		t.Fatal(err)
	}
	if mgr.DB() == old {
		t.Fatal("successful Reopen must install a new handle")
	}
	if err := old.Ping(); err == nil {
		t.Fatal("old handle should be closed")
	}
	var v string
	if err := mgr.DB().QueryRow(`SELECT v FROM reopen_probe`).Scan(&v); err != nil || v != "ok" {
		t.Fatalf("probe row after reopen: v=%q err=%v", v, err)
	}
}
