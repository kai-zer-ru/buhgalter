package categoryseed_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func TestEnsureSubscriptionsCategoryRenamesUserCategory(t *testing.T) {
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "subcat", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate pre-existing user category «Подписки» (as if created before feature).
	// Seed already created system «Подписки» — rename system away and insert user one, then re-ensure.
	q := sqlcdb.New(mgr.DB())
	sys, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: categoryseed.SubscriptionsCategoryName, Type: "expense",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.DeleteCategory(ctx, sqlcdb.DeleteCategoryParams{ID: sys.ID, UserID: userID}); err != nil {
		t.Fatal(err)
	}
	userCatID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
		ID: userCatID, UserID: userID, Name: categoryseed.SubscriptionsCategoryName, Type: "expense",
		Icon: "default", SortOrder: 10, IsPrimary: 0, IsSystem: 0, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	if err := categoryseed.EnsureSubscriptionsCategory(ctx, mgr.DB(), userID); err != nil {
		t.Fatal(err)
	}

	var legacySystem, newSystem int
	_ = mgr.DB().QueryRow(`
		SELECT is_system FROM categories WHERE user_id = ? AND name = ? AND type = 'expense'`,
		userID, categoryseed.SubscriptionsLegacyName,
	).Scan(&legacySystem)
	if legacySystem != 0 {
		t.Fatalf("legacy category should remain user-owned, is_system=%d", legacySystem)
	}
	var renamedID string
	_ = mgr.DB().QueryRow(`
		SELECT id FROM categories WHERE user_id = ? AND name = ? AND type = 'expense'`,
		userID, categoryseed.SubscriptionsLegacyName,
	).Scan(&renamedID)
	if renamedID != userCatID {
		t.Fatalf("expected user category renamed in place, got id=%q want %q", renamedID, userCatID)
	}

	sysRow, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: categoryseed.SubscriptionsCategoryName, Type: "expense",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sysRow.IsSystem != 1 {
		t.Fatalf("new Подписки must be system, got %d", sysRow.IsSystem)
	}
	if sysRow.ID == userCatID {
		t.Fatal("system Подписки must be a new row, not the renamed user category")
	}
	_ = newSystem
}

func TestEnsureSubscriptionsCategoryLegacyNameCollision(t *testing.T) {
	mgr, err := db.NewManager(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "subcat2", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	q := sqlcdb.New(mgr.DB())
	sys, err := q.GetCategoryByNameAndType(ctx, sqlcdb.GetCategoryByNameAndTypeParams{
		UserID: userID, Name: categoryseed.SubscriptionsCategoryName, Type: "expense",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.DeleteCategory(ctx, sqlcdb.DeleteCategoryParams{ID: sys.ID, UserID: userID}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range []string{categoryseed.SubscriptionsLegacyName, categoryseed.SubscriptionsCategoryName} {
		if err := q.InsertCategory(ctx, sqlcdb.InsertCategoryParams{
			ID: uuid.NewString(), UserID: userID, Name: name, Type: "expense",
			Icon: "default", SortOrder: 10, IsPrimary: 0, IsSystem: 0, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := categoryseed.EnsureSubscriptionsCategory(ctx, mgr.DB(), userID); err != nil {
		t.Fatal(err)
	}

	var got string
	err = mgr.DB().QueryRow(`
		SELECT name FROM categories
		WHERE user_id = ? AND type = 'expense' AND is_system = 0 AND name LIKE 'Подписки (старые)%'
		ORDER BY name DESC LIMIT 1`, userID).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != categoryseed.SubscriptionsLegacyName+" 2" {
		t.Fatalf("expected %q, got %q", categoryseed.SubscriptionsLegacyName+" 2", got)
	}
}
