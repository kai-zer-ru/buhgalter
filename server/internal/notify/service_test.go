package notify_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/features"
	. "github.com/kai-zer-ru/buhgalter/internal/notify"
	"github.com/kai-zer-ru/buhgalter/internal/settingscache"
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
	userID, err := auth.CreateUser(ctx, sqlDB, "notifyuser", hash, "Notify", false, auth.UserStatusActive)
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

	preview, err := PreviewTemplate(ctx, sqlDB, userID, PreviewInput{
		TriggerType: TriggerDebtDueSoon,
		Template:    "Долг {amount} до {due_date}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Text == "" {
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
	phReset := AvailablePlaceholders(TriggerPasswordReset)
	found := false
	for _, p := range phReset {
		if p == "reset_url" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("password_reset placeholders: %v", phReset)
	}
	phShortfall := AvailablePlaceholders(TriggerBalanceShortfall)
	foundAmount, foundAccountURL := false, false
	for _, p := range phShortfall {
		if p == "amount" {
			foundAmount = true
		}
		if p == "account_url" {
			foundAccountURL = true
		}
	}
	if !foundAmount || !foundAccountURL {
		t.Fatalf("balance_shortfall placeholders: %v", phShortfall)
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

func TestPreviewPasswordResetTemplateWithExternalURL(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	settingscache.Invalidate()

	externalURL := "https://buhgalter.example.com"
	_, err := sqlDB.ExecContext(ctx, `UPDATE system_settings SET external_url = ? WHERE id = 1`, externalURL)
	if err != nil {
		t.Fatal(err)
	}

	template := "Запрос: {display_name} (@{login}), {requested_at}.\n{reset_url}"
	preview, err := PreviewTemplate(ctx, sqlDB, userID, PreviewInput{TriggerType: TriggerPasswordReset, Template: template})
	if err != nil {
		t.Fatal(err)
	}
	wantLink := "https://buhgalter.example.com/admin/users?reset=00000000-0000-0000-0000-000000000001"
	if !strings.Contains(preview.Text, wantLink) {
		t.Fatalf("preview %q should contain %q", preview.Text, wantLink)
	}
}

func TestPreviewPasswordResetTemplateWithoutExternalURL(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	settingscache.Invalidate()
	_, err := sqlDB.ExecContext(ctx, `UPDATE system_settings SET external_url = NULL WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}

	template := "Запрос: {display_name} (@{login}), {requested_at}.\n{reset_url}"
	preview, err := PreviewTemplate(ctx, sqlDB, userID, PreviewInput{TriggerType: TriggerPasswordReset, Template: template})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Text, "Нет внешней ссылки — настройте внешний URL в админке.") {
		t.Fatalf("preview %q should contain hint", preview.Text)
	}
}

func TestGetSettingsHidesUserRegistrationWhenDisabled(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	adminID, err := auth.CreateUser(ctx, sqlDB, "adminnotify", hash, "Admin", true, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	_ = userID

	settings, err := GetSettings(ctx, sqlDB, adminID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tpl := range settings.Templates {
		if tpl.TriggerType == TriggerUserRegistration {
			t.Fatal("expected user_registration template to be hidden when registration is disabled")
		}
	}
	if settings.TriggerUserRegistration {
		t.Fatal("expected trigger_user_registration to be false when registration is disabled")
	}

	_, err = sqlDB.ExecContext(ctx, `INSERT INTO feature_flags (key, enabled, updated_at) VALUES ('registration', 1, datetime('now')) ON CONFLICT(key) DO UPDATE SET enabled = 1`)
	if err != nil {
		t.Fatal(err)
	}
	features.Invalidate()
	settings, err = GetSettings(ctx, sqlDB, adminID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tpl := range settings.Templates {
		if tpl.TriggerType == TriggerUserRegistration {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected user_registration template when registration is enabled")
	}
}

func strPtr(s string) *string { return &s }

func TestDeliverNoPanicWithoutSecretKey(t *testing.T) {
	t.Parallel()

	ctx, sqlDB, userID := seedNotifyUser(t)
	_, err := sqlDB.ExecContext(ctx, `UPDATE system_settings SET notification_secret_key = '' WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	q := sqlcdb.New(sqlDB)
	if err := q.EnsureNotificationSettings(ctx, userID); err != nil {
		t.Fatal(err)
	}
	settings, err := q.GetNotificationSettings(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Deliver panicked: %v", rec)
		}
	}()
	Deliver(ctx, sqlDB, settings, userID, TriggerBudgetThreshold, "budget-1", "2026-07-01", "test message", FormatData{})
}

func TestKeyboardSettingsRoundTripAndPreview(t *testing.T) {
	ctx, sqlDB, userID := seedNotifyUser(t)
	settingscache.Invalidate()
	_, err := sqlDB.ExecContext(ctx, `UPDATE system_settings SET external_url = ? WHERE id = 1`, "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}

	tg := InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "url", Text: "Долги", URL: "{debt_url}"}},
	}}
	maxKB := InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "link", Text: "Долги", URL: "{debt_url}"}},
	}}
	updated, err := UpdateSettings(ctx, sqlDB, userID, UpdateSettingsInput{
		KeyboardTelegram: &tg,
		KeyboardMax:      &maxKB,
		Templates: []TemplateUpdate{{
			TriggerType: TriggerDebtDueSoon,
			Template:    "Долг {amount}",
			KeyboardTelegram: &InlineKeyboard{Rows: [][]KeyboardButton{
				{{Type: "callback_data", Text: "OK", CallbackData: "ok"}},
			}},
		}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.KeyboardTelegram.IsEmpty() || updated.KeyboardTelegram.Rows[0][0].URL != "{debt_url}" {
		t.Fatalf("global tg: %+v", updated.KeyboardTelegram)
	}
	var debtTpl *TemplateView
	for i := range updated.Templates {
		if updated.Templates[i].TriggerType == TriggerDebtDueSoon {
			debtTpl = &updated.Templates[i]
			break
		}
	}
	if debtTpl == nil || debtTpl.KeyboardTelegram.IsEmpty() {
		t.Fatalf("template kb: %+v", debtTpl)
	}

	preview, err := PreviewTemplate(ctx, sqlDB, userID, PreviewInput{
		TriggerType: TriggerDebtDueSoon,
		Template:    "Долг {amount}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.KeyboardTelegram.ButtonCount() < 2 {
		t.Fatalf("preview merge expected global+template, got %+v", preview.KeyboardTelegram)
	}
	foundURL := false
	for _, row := range preview.KeyboardTelegram.Rows {
		for _, btn := range row {
			if btn.Type == "url" && strings.HasPrefix(btn.URL, "https://app.example.com") {
				foundURL = true
			}
		}
	}
	if !foundURL {
		t.Fatalf("expected resolved debt_url in preview: %+v", preview.KeyboardTelegram)
	}

	trigger := TriggerDebtDueSoon
	if err := ResetTemplates(ctx, sqlDB, userID, &trigger); err != nil {
		t.Fatal(err)
	}
	after, err := GetSettings(ctx, sqlDB, userID)
	if err != nil {
		t.Fatal(err)
	}
	if after.KeyboardTelegram.IsEmpty() {
		t.Fatal("reset must not clear global keyboard")
	}
	for _, tpl := range after.Templates {
		if tpl.TriggerType == TriggerDebtDueSoon && (!tpl.KeyboardTelegram.IsEmpty() || tpl.IsCustom) {
			t.Fatalf("template keyboard should reset: %+v", tpl)
		}
	}
}
