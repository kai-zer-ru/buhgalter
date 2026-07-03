package categoryseed_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestSeedDefaults(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	ctx := context.Background()
	userID, err := auth.CreateUser(ctx, mgr.DB(), "u1", "hash", "User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	err = mgr.DB().QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = ?`, userID).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != categoryseed.DefaultCount {
		t.Fatalf("expected %d categories, got %d", categoryseed.DefaultCount, count)
	}

	var expense, income int
	_ = mgr.DB().QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = ? AND type = 'expense'`, userID).Scan(&expense)
	_ = mgr.DB().QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = ? AND type = 'income'`, userID).Scan(&income)
	if expense != 10 || income != 4 {
		t.Fatalf("expected 10 expense and 4 income, got %d/%d", expense, income)
	}

	var systemCount int
	_ = mgr.DB().QueryRow(`SELECT COUNT(*) FROM categories WHERE user_id = ? AND is_system = 1`, userID).Scan(&systemCount)
	if systemCount != 6 {
		t.Fatalf("expected 6 system categories, got %d", systemCount)
	}
}
