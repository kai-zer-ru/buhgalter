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

func TestSyncFromMerchants_backfillsStoreFilterCatalog(t *testing.T) {
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
	m1ID, m2ID := uuid.NewString(), uuid.NewString()
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO merchants (id, user_id, name, icon, created_at) VALUES
		(?, ?, 'Пятёрочка', 'default', ?),
		(?, ?, 'Лента', 'default', ?)`,
		m1ID, userID, now, m2ID, userID, now)
	if err != nil {
		t.Fatal(err)
	}

	catID := uuid.NewString()
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO categories (id, user_id, name, type, icon, is_system, sort_order, created_at)
		VALUES (?, ?, 'Еда', 'expense', 'default', 0, 0, ?)`, catID, userID, now)
	if err != nil {
		t.Fatal(err)
	}
	accID := uuid.NewString()
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Кошелёк', 'cash', 100000, 'active', ?, ?)`, accID, userID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	for i, mid := range []string{m1ID, m2ID} {
		txID := uuid.NewString()
		_, err = sqlDB.ExecContext(ctx, `
			INSERT INTO transactions (
				id, user_id, account_id, type, kind, amount, category_id, merchant_id,
				transaction_date, affects_balance, created_at, updated_at
			) VALUES (?, ?, ?, 'expense', 'manual', ?, ?, ?, ?, 1, ?, ?)`,
			txID, userID, accID, int64(5000+i*1000), catID, mid, now, now, now)
		if err != nil {
			t.Fatal(err)
		}
	}

	stores, err := List(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stores) < 2 {
		t.Fatalf("expected at least 2 stores after sync, got %d: %+v", len(stores), stores)
	}
	names := map[string]bool{}
	for _, s := range stores {
		names[s.Name] = true
	}
	if !names["Пятёрочка"] || !names["Лента"] {
		t.Fatalf("expected merchant names in store catalog, got %+v", names)
	}
}
