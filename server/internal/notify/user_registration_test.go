package notify_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	. "github.com/kai-zer-ru/buhgalter/internal/notify"
)

func TestNotifyAdminsOnUserRegistration(t *testing.T) {
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
	adminID, err := auth.CreateUser(ctx, sqlDB, "admin", hash, "Admin", true, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	newUserID, err := auth.CreateUser(ctx, sqlDB, "pending1", hash, "Pending", false, auth.UserStatusPending)
	if err != nil {
		t.Fatal(err)
	}

	secret := "12345678901234567890123456789012"
	_, err = sqlDB.ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned,
			trigger_user_registration, debt_days_before, credit_days_before,
			notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 0, 0, 1, 1, 1, '00:00', ?)`,
		adminID, token, now)
	if err != nil {
		t.Fatal(err)
	}

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	registeredAt := time.Now().UTC().Format(time.RFC3339)
	if err := NotifyAdminsOnUserRegistration(ctx, sqlDB, newUserID, "pending1", "Pending", registeredAt); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected telegram notifier to be called")
	}

	var count int
	if err := sqlDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notification_log
		WHERE user_id = ? AND trigger_type = ? AND status = 'sent'`,
		adminID, TriggerUserRegistration).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected notification log entry for user_registration")
	}
}
