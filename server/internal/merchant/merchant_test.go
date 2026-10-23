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

	renamed, err := Update(ctx, database, userID, m.ID, "Перекрёсток", "pyaterochka", nil)
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
	updated, err := Update(ctx, database, userID, m.ID, "Магнит", "shopping", nil)
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

func aliasSet(m Merchant) []string {
	if m.Aliases == nil {
		return []string{}
	}
	return m.Aliases
}

func TestMerchantAliasesResolveAndRename(t *testing.T) {
	ctx, database, userID := setup(t)

	m, err := CreateWithAliases(ctx, database, userID, "Пятёрочка", "shopping", []string{"Pyaterochka", "  ", "пятёрочка", "PYATEROCHKA"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Пятёрочка" || len(aliasSet(m)) != 1 || aliasSet(m)[0] != "Pyaterochka" {
		t.Fatalf("created=%+v", m)
	}

	again, err := CreateWithAliases(ctx, database, userID, "pyaterochka", "default", []string{"другой"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != m.ID || len(aliasSet(again)) != 1 {
		t.Fatalf("alias hit should return existing unchanged: %+v", again)
	}

	alias := "МАГНИТ У ДОМА"
	shop, err := CreateWithAliases(ctx, database, userID, "Магнит", "default", []string{alias, ""})
	if err != nil {
		t.Fatal(err)
	}
	name := "магнит у дома"
	id, err := Resolve(ctx, database, userID, nil, &name)
	if err != nil || id == nil || *id != shop.ID {
		t.Fatalf("resolve alias id=%v err=%v", id, err)
	}
	fresh := "Лента"
	createdID, err := Resolve(ctx, database, userID, nil, &fresh)
	if err != nil || createdID == nil || *createdID == shop.ID {
		t.Fatalf("new name id=%v err=%v", createdID, err)
	}

	if _, err := CreateWithAliases(ctx, database, userID, "Дикси", "default", []string{"магнит"}); err != ErrNameTaken {
		t.Fatalf("alias vs primary: %v", err)
	}
	otherAliases := []string{"pyaterochka"}
	if _, err := CreateWithAliases(ctx, database, userID, "Перекрёсток", "default", otherAliases); err != ErrNameTaken {
		t.Fatalf("alias vs alias: %v", err)
	}

	kept, err := Update(ctx, database, userID, m.ID, "Пятёрочка", "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	if kept.Icon != "default" || len(aliasSet(kept)) != 1 || aliasSet(kept)[0] != "Pyaterochka" {
		t.Fatalf("put without aliases changed list: %+v", kept)
	}
	replaced := []string{"Пятёрочка у дома", "пятёрочка у дома", "  "}
	replacedMerchant, err := Update(ctx, database, userID, m.ID, "Пятёрочка", "", &replaced)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliasSet(replacedMerchant)) != 1 || aliasSet(replacedMerchant)[0] != "Пятёрочка у дома" {
		t.Fatalf("replaced=%+v", replacedMerchant)
	}

	renamed, err := Update(ctx, database, userID, m.ID, "пятёрочка у дома", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "пятёрочка у дома" || len(aliasSet(renamed)) != 0 {
		t.Fatalf("rename onto own alias: %+v", renamed)
	}
	got, err := Get(ctx, database, userID, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != renamed.Name || len(got.Aliases) != 0 {
		t.Fatalf("persisted rename=%+v", got)
	}
}
