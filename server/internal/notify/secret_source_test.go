package notify

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestSecretKeyConfiguredFromDB(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	ctx := context.Background()
	sqlDB := mgr.DB()
	if SecretKeyConfigured(ctx, sqlDB) {
		t.Fatal("expected false before key is set")
	}

	secret := "12345678901234567890123456789012"
	if _, err := sqlDB.ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret); err != nil {
		t.Fatal(err)
	}
	if !SecretKeyConfigured(ctx, sqlDB) {
		t.Fatal("expected true after key is set")
	}
}

func TestResolveSecretKeyIgnoresEnv(t *testing.T) {
	t.Setenv("BUHGALTER_SECRET_KEY", "12345678901234567890123456789012")

	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	ctx := context.Background()
	sqlDB := mgr.DB()

	secret, err := ResolveSecretKey(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if secret != "" {
		t.Fatalf("expected empty DB key, got %q", secret)
	}
	if SecretKeyConfigured(ctx, sqlDB) {
		t.Fatal("env key must not enable encryption without DB key")
	}
}
