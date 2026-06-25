package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestPasswordResetRequestFlow(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	sqlDB := mgr.DB()
	ctx := context.Background()

	if err := RequestPasswordReset(ctx, sqlDB, "nobody"); err != nil {
		t.Fatal(err)
	}
	pending, err := ListPendingPasswordResetRequests(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("unknown login should not create request, got %d", len(pending))
	}

	hash, err := HashPassword("userpass1")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := CreateUser(ctx, sqlDB, "alice", hash, "Alice", false)
	if err != nil {
		t.Fatal(err)
	}

	if err := RequestPasswordReset(ctx, sqlDB, "alice"); err != nil {
		t.Fatal(err)
	}
	pending, err = ListPendingPasswordResetRequests(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].UserID != userID {
		t.Fatalf("pending %+v", pending)
	}

	if err := DismissPasswordResetRequest(ctx, sqlDB, pending[0].ID); err != nil {
		t.Fatal(err)
	}
	pending, err = ListPendingPasswordResetRequests(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatal("expected dismissed request hidden")
	}

	if err := RequestPasswordReset(ctx, sqlDB, "alice"); err != nil {
		t.Fatal(err)
	}
	newHash, err := HashPassword("newpass12")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetUserPassword(ctx, sqlDB, userID, newHash); err != nil {
		t.Fatal(err)
	}
	if err := DismissPasswordResetRequestsForUser(ctx, sqlDB, userID); err != nil {
		t.Fatal(err)
	}
	pending, err = ListPendingPasswordResetRequests(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatal("reset password should dismiss pending request")
	}
}
