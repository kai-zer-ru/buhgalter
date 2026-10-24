package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestList_doesNotCopyMerchants(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	sqlDB := mgr.DB()

	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, sqlDB, "storeuser", hash, "Store User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO merchants (id, user_id, name, icon, created_at) VALUES
		(?, ?, 'Пятёрочка', 'default', ?),
		(?, ?, 'Лента', 'default', ?)`,
		uuid.NewString(), userID, now, uuid.NewString(), userID, now)
	if err != nil {
		t.Fatal(err)
	}

	stores, err := List(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) != 0 {
		t.Fatalf("GET /stores copied merchants: %+v", stores)
	}
}
