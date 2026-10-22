package tag

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
	userID, err := auth.CreateUser(ctx, mgr.DB(), "tuser", "hash", "T", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, mgr.DB(), userID
}

func TestTagCRUDResolveAndFilter(t *testing.T) {
	ctx, database, userID := setup(t)

	a, err := Create(ctx, database, userID, "отпуск")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Create(ctx, database, userID, "семья")
	if err != nil {
		t.Fatal(err)
	}

	filtered, err := List(ctx, database, userID, "отп")
	if err != nil || len(filtered) != 1 || filtered[0].ID != a.ID {
		t.Fatalf("filter=%v err=%v", filtered, err)
	}

	ids, err := ResolveIDs(ctx, database, userID, []string{a.ID}, []string{"работа", "отпуск"})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("want 2 unique ids, got %v", ids)
	}

	all, err := List(ctx, database, userID, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("all=%v err=%v", all, err)
	}

	if err := Delete(ctx, database, userID, a.ID); err != nil {
		t.Fatal(err)
	}
}
