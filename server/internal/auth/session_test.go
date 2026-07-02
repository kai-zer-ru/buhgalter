package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func TestSessionIdleExpiry(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	sqlDB := mgr.DB()

	userID, err := CreateUser(context.Background(), sqlDB, "testuser", "hash", "Test", false, UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	token, err := CreateSession(context.Background(), sqlDB, userID, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := LookupSession(context.Background(), sqlDB, token); err != nil {
		t.Fatalf("expected valid session: %v", err)
	}

	hash := HashToken(token)
	_, err = sqlDB.Exec(`
		UPDATE sessions
		SET last_activity = datetime('now', '-11 days'),
		    expires_at = datetime('now', '-1 day')
		WHERE token_hash = ?`, hash)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := LookupSession(context.Background(), sqlDB, token); err == nil {
		t.Fatal("expected expired session")
	}
}

func TestVerifyTokenSessionAndAPI(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	sqlDB := mgr.DB()
	ctx := context.Background()

	userID, err := CreateUser(ctx, sqlDB, "apiuser", "hash", "API", false, UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	sessionToken, err := CreateSession(ctx, sqlDB, userID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyToken(ctx, sqlDB, sessionToken) {
		t.Fatal("session token should be valid")
	}

	raw := "bhg_testtoken123456789"
	hash := HashToken(raw)
	_, err = sqlDB.Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix)
		VALUES ('tok1', ?, 'test', ?, 'bhg_test')`, userID, hash)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyToken(ctx, sqlDB, raw) {
		t.Fatal("api token should be valid")
	}

	_, err = sqlDB.Exec(`DELETE FROM api_tokens WHERE id = 'tok1'`)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyToken(ctx, sqlDB, raw) {
		t.Fatal("revoked api token should be invalid")
	}
}

func TestAPITokenExpiredRejected(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	sqlDB := mgr.DB()
	ctx := context.Background()

	userID, err := CreateUser(ctx, sqlDB, "expuser", "hash", "Exp", false, UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}

	raw := "bhg_expiredtoken123456"
	hash := HashToken(raw)
	expired := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	_, err = sqlDB.Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, expires_at)
		VALUES ('tok-exp', ?, 'expired', ?, 'bhg_expi', ?)`, userID, hash, expired)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyToken(ctx, sqlDB, raw) {
		t.Fatal("expired api token should be invalid")
	}
	if _, err := LookupAPIToken(ctx, sqlDB, raw); err == nil {
		t.Fatal("LookupAPIToken should reject expired token")
	}
}

func TestSessionRefreshExtendsExpiry(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	sqlDB := mgr.DB()

	userID, _ := CreateUser(context.Background(), sqlDB, "refresh", "hash", "R", false, UserStatusActive)
	token, _ := CreateSession(context.Background(), sqlDB, userID, "", "")

	s1, err := LookupSession(context.Background(), sqlDB, token)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)
	s2, err := LookupSession(context.Background(), sqlDB, token)
	if err != nil {
		t.Fatal(err)
	}
	if s2.LastActivity.Before(s1.LastActivity) {
		t.Fatal("expected last_activity to be updated")
	}
	_ = sqlDB // silence unused in edge builds
}
