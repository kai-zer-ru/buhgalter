package notify_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	. "github.com/kai-zer-ru/buhgalter/internal/notify"
)

func seedNotifyUser(t *testing.T) (context.Context, *sql.DB, string) {
	t.Helper()
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
	userID, err := auth.CreateUser(ctx, sqlDB, "notifyuser", hash, "Notify", false)
	if err != nil {
		t.Fatal(err)
	}
	secret := "12345678901234567890123456789012"
	_, err = sqlDB.ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, sqlDB, userID
}

func TestSecretBoxRoundTrip(t *testing.T) {
	key := "12345678901234567890123456789012"
	box, err := NewSecretBox(key)
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := box.Encrypt("my-bot-token")
	if err != nil || cipher == "" {
		t.Fatal(err)
	}
	plain, err := box.Decrypt(cipher)
	if err != nil || plain != "my-bot-token" {
		t.Fatalf("decrypt: %q %v", plain, err)
	}
}

func TestNewSecretBoxErrors(t *testing.T) {
	if _, err := NewSecretBox(""); err == nil {
		t.Fatal("expected error for empty secret")
	}
	if _, err := NewSecretBox("short"); err == nil {
		t.Fatal("expected error for invalid secret")
	}
}

func TestGetAndUpdateSettings(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	box, err := NewSecretBox("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := GetSettings(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.Templates) == 0 {
		t.Fatal("expected default templates")
	}

	enabled := true
	days := int64(3)
	chatID := "12345"
	token := "bot-token"
	updated, err := UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		TelegramEnabled:       &enabled,
		TelegramBotToken:      &token,
		TelegramChatID:        &chatID,
		DebtDaysBefore:        &days,
		NotificationTimeLocal: strPtr("09:30"),
	}, box)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.TelegramEnabled || !updated.TelegramConfigured {
		t.Fatalf("settings: %+v", updated)
	}
	if updated.DebtDaysBefore != 3 {
		t.Fatalf("debt days %d", updated.DebtDaysBefore)
	}
}

func TestPreviewTemplateAndReset(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)

	text, err := PreviewTemplate(ctx, sqlDB, userID, TriggerDebtDueSoon, "Долг {amount} до {due_date}")
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("expected preview text")
	}

	custom := "Мой шаблон {amount}"
	_, err = UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		Templates: []TemplateUpdate{{TriggerType: TriggerDebtDueSoon, Template: custom}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	trigger := TriggerDebtDueSoon
	if err := ResetTemplates(ctx, sqlDB, userID, &trigger); err != nil {
		t.Fatal(err)
	}
	if err := ResetTemplates(ctx, sqlDB, userID, nil); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateSettingsMaxOfficialLargeRecipientID(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	box, err := NewSecretBox("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	provider := MaxProviderOfficial
	token := "f9LHodD0cOK3w0x2QnwLjAO1Gb39lDxH3qQRwv4sw5WVaSOw28fzFBIhpTW9DrRvfI6CZHsYYPbhxdKg5d73"
	rid := int64(-70955246010435)
	updated, err := UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		MaxEnabled:     &enabled,
		MaxProvider:    &provider,
		MaxToken:       &token,
		MaxRecipientID: &rid,
	}, box)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.MaxRecipientID == nil || *updated.MaxRecipientID != rid {
		t.Fatalf("recipient id: %+v", updated.MaxRecipientID)
	}
}

func TestUpdateSettingsValidation(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	bad := int64(99)
	_, err := UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		DebtDaysBefore: &bad,
	}, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAvailablePlaceholders(t *testing.T) {
	ph := AvailablePlaceholders(TriggerDebtDueSoon)
	if len(ph) == 0 {
		t.Fatal("expected placeholders")
	}
}

func TestSendTestTelegram(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	box, err := NewSecretBox("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("bot-token")
	if err != nil {
		t.Fatal(err)
	}
	chat := "999"
	enabled := true
	_, err = UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		TelegramEnabled:  &enabled,
		TelegramBotToken: strPtr("bot-token"),
		TelegramChatID:   &chat,
	}, box)
	if err != nil {
		t.Fatal(err)
	}
	_ = token
	if err := SendTest(ctx, sqlDB, userID, ChannelTelegram, box); err != nil {
		t.Fatal(err)
	}
}

func TestSendTestMaxChannel(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_MAX_A161_BASE_URL", mock.URL)

	box, err := NewSecretBox("12345678901234567890123456789012")
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	token := "max-token-1234567890"
	provider := MaxProviderA161
	maxUser := int64(12345)
	_, err = UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		MaxEnabled:  &enabled,
		MaxProvider: &provider,
		MaxToken:    &token,
		MaxUserID:   &maxUser,
	}, box)
	if err != nil {
		t.Fatal(err)
	}
	if err := SendTest(ctx, sqlDB, userID, ChannelMax, box); err != nil {
		t.Fatal(err)
	}
}

func strPtr(s string) *string { return &s }
