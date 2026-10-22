package merchant

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func setup(t *testing.T) (context.Context, *sql.DB, string) {
	t.Helper()
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "muser", "hash", "M", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, mgr.DB(), userID
}

func TestMerchantCRUDAndResolve(t *testing.T) {
	ctx, database, userID := setup(t)

	m, err := Create(ctx, database, userID, "  Пятёрочка  ", "shopping")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Пятёрочка" || m.Icon != "shopping" {
		t.Fatalf("merchant=%+v", m)
	}

	again, err := Create(ctx, database, userID, "пятёрочка", "default")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != m.ID {
		t.Fatalf("expected get-or-create same id, got %s vs %s", again.ID, m.ID)
	}

	list, err := List(ctx, database, userID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}

	renamed, err := Update(ctx, database, userID, m.ID, "Перекрёсток", "pyaterochka")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Перекрёсток" || renamed.Icon != "pyaterochka" {
		t.Fatalf("renamed=%+v", renamed)
	}

	name := "Магнит"
	id, err := Resolve(ctx, database, userID, nil, &name)
	if err != nil || id == nil {
		t.Fatalf("resolve name: %v %v", id, err)
	}
	list, _ = List(ctx, database, userID)
	if len(list) != 2 {
		t.Fatalf("want 2 merchants, got %d", len(list))
	}

	if err := Delete(ctx, database, userID, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(ctx, database, userID, m.ID); err != ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestMerchantUpdateIconOnly(t *testing.T) {
	ctx, database, userID := setup(t)
	m, err := Create(ctx, database, userID, "Магнит", "default")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := Update(ctx, database, userID, m.ID, "Магнит", "shopping")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Icon != "shopping" || updated.Name != "Магнит" {
		t.Fatalf("updated=%+v", updated)
	}
	got, err := Get(ctx, database, userID, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Icon != "shopping" {
		t.Fatalf("persisted icon=%q", got.Icon)
	}
}
