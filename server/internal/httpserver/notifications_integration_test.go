package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationsSettingsAndPreview(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	body, _ := json.Marshal(map[string]any{
		"telegram_enabled":        true,
		"telegram_bot_token":      "telegram-test-token",
		"telegram_chat_id":        "123456",
		"trigger_debt":            true,
		"trigger_credit":          true,
		"trigger_planned":         true,
		"debt_days_before":        2,
		"credit_days_before":      3,
		"notification_time_local": "09:30",
		"templates": []map[string]string{
			{
				"trigger_type": "debt_overdue",
				"template":     "Просрочен: {debtor} {amount}",
			},
		},
	})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put notifications status %d", resp.StatusCode)
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/user/notifications", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get notifications status %d", getResp.StatusCode)
	}
	var payload map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&payload)
	if _, ok := payload["telegram_bot_token"]; ok {
		t.Fatal("token must not be returned in GET")
	}
	if payload["telegram_configured"] != true {
		t.Fatal("expected telegram_configured=true")
	}
	if payload["notification_time_local"] != "09:30" {
		t.Fatalf("expected notification_time_local=09:30, got %v", payload["notification_time_local"])
	}

	previewBody, _ := json.Marshal(map[string]string{
		"trigger_type": "credit_payment",
		"template":     "Платёж: {credit} {amount} {when}",
	})
	previewResp, err := env.authedRequest(http.MethodPost, "/api/v1/user/notifications/templates/preview", bytes.NewReader(previewBody))
	if err != nil {
		t.Fatal(err)
	}
	defer previewResp.Body.Close()
	if previewResp.StatusCode != http.StatusOK {
		t.Fatalf("preview status %d", previewResp.StatusCode)
	}
	var preview map[string]string
	_ = json.NewDecoder(previewResp.Body).Decode(&preview)
	if strings.TrimSpace(preview["text"]) == "" {
		t.Fatal("preview text is empty")
	}
}

func TestNotificationsSendTestTelegram(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	var called bool
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !strings.Contains(r.URL.Path, "/bottelegram-test-token/sendMessage") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	putBody, _ := json.Marshal(map[string]any{
		"telegram_enabled":   true,
		"telegram_bot_token": "telegram-test-token",
		"telegram_chat_id":   "123456",
	})
	putResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(putBody))
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put notifications status %d", putResp.StatusCode)
	}

	testBody, _ := json.Marshal(map[string]string{"channel": "telegram"})
	testResp, err := env.authedRequest(http.MethodPost, "/api/v1/user/notifications/test", bytes.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}
	defer testResp.Body.Close()
	if testResp.StatusCode != http.StatusOK {
		t.Fatalf("test send status %d", testResp.StatusCode)
	}
	if !called {
		t.Fatal("telegram API mock was not called")
	}
}

func TestNotificationsSendTestMaxUsesMaxChannelInTemplate(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	var called bool
	var gotPath string
	var gotQuery string
	var gotText string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(raw, &payload)
		gotText = payload["text"]
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer mock.Close()

	t.Setenv("BUHGALTER_MAX_A161_BASE_URL", mock.URL)

	putBody, _ := json.Marshal(map[string]any{
		"max_enabled":  true,
		"max_provider": "a161",
		"max_token":    "1234567890123456",
		"max_user_id":  777,
	})
	putResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(putBody))
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put notifications status %d", putResp.StatusCode)
	}

	testBody, _ := json.Marshal(map[string]string{"channel": "max"})
	testResp, err := env.authedRequest(http.MethodPost, "/api/v1/user/notifications/test", bytes.NewReader(testBody))
	if err != nil {
		t.Fatal(err)
	}
	defer testResp.Body.Close()
	if testResp.StatusCode != http.StatusOK {
		t.Fatalf("test send status %d", testResp.StatusCode)
	}
	if !called {
		t.Fatal("max API mock was not called")
	}
	if gotPath != "/messages" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery != "user_id=777" {
		t.Fatalf("unexpected query: %s", gotQuery)
	}
	if !strings.Contains(gotText, "Канал: max.") {
		t.Fatalf("unexpected text: %q", gotText)
	}
}

func TestNotificationsTemplateReset(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	putBody, _ := json.Marshal(map[string]any{
		"templates": []map[string]string{
			{
				"trigger_type": "test",
				"template":     "Custom {channel}",
			},
		},
	})
	putResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(putBody))
	if err != nil {
		t.Fatal(err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("put notifications status %d", putResp.StatusCode)
	}

	resetResp, err := env.authedRequest(http.MethodPost, "/api/v1/user/notifications/templates/reset", bytes.NewReader([]byte(`{"trigger_type":"test"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resetResp.Body.Close()
	if resetResp.StatusCode != http.StatusOK {
		t.Fatalf("reset status %d", resetResp.StatusCode)
	}
	var view map[string]any
	_ = json.NewDecoder(resetResp.Body).Decode(&view)
	templates, _ := view["templates"].([]any)
	for _, item := range templates {
		row := item.(map[string]any)
		if row["trigger_type"] == "test" && row["is_custom"] == true {
			t.Fatal("test template must be reset to default")
		}
	}
}

func TestNotificationsDebtTemplateRejectedWhenDisabled(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	disableBody, _ := json.Marshal(map[string]any{
		"trigger_debt": false,
	})
	disableResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(disableBody))
	if err != nil {
		t.Fatal(err)
	}
	disableResp.Body.Close()
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("disable trigger status %d", disableResp.StatusCode)
	}

	templateBody, _ := json.Marshal(map[string]any{
		"templates": []map[string]string{
			{
				"trigger_type": "debt_overdue",
				"template":     "Долг: {debtor} {amount}",
			},
		},
	})
	templateResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(templateBody))
	if err != nil {
		t.Fatal(err)
	}
	defer templateResp.Body.Close()
	if templateResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when saving debt template with disabled trigger, got %d", templateResp.StatusCode)
	}
}

func TestNotificationsDebtPolicyRejectedWhenDisabled(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	disableBody, _ := json.Marshal(map[string]any{
		"trigger_debt": false,
	})
	disableResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(disableBody))
	if err != nil {
		t.Fatal(err)
	}
	disableResp.Body.Close()
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("disable trigger status %d", disableResp.StatusCode)
	}

	policyBody, _ := json.Marshal(map[string]any{
		"debt_days_before": 5,
	})
	policyResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(policyBody))
	if err != nil {
		t.Fatal(err)
	}
	defer policyResp.Body.Close()
	if policyResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when saving debt_days_before with disabled trigger, got %d", policyResp.StatusCode)
	}
}

func TestNotificationsTriggerNegativeBalanceRoundtrip(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	disableBody, _ := json.Marshal(map[string]any{
		"trigger_negative_balance": false,
	})
	disableResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(disableBody))
	if err != nil {
		t.Fatal(err)
	}
	defer disableResp.Body.Close()
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("disable trigger status %d", disableResp.StatusCode)
	}
	var putPayload map[string]any
	if err := json.NewDecoder(disableResp.Body).Decode(&putPayload); err != nil {
		t.Fatal(err)
	}
	if putPayload["trigger_negative_balance"] != false {
		t.Fatalf("PUT response: expected trigger_negative_balance=false, got %v", putPayload["trigger_negative_balance"])
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/user/notifications", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	var getPayload map[string]any
	if err := json.NewDecoder(getResp.Body).Decode(&getPayload); err != nil {
		t.Fatal(err)
	}
	if getPayload["trigger_negative_balance"] != false {
		t.Fatalf("GET response: expected trigger_negative_balance=false, got %v", getPayload["trigger_negative_balance"])
	}
}

func TestNotificationsBalanceShortfallTemplateRejectedWhenDisabled(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	setNotificationSecretKey(t, env)

	disableBody, _ := json.Marshal(map[string]any{
		"trigger_negative_balance": false,
	})
	disableResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(disableBody))
	if err != nil {
		t.Fatal(err)
	}
	disableResp.Body.Close()
	if disableResp.StatusCode != http.StatusOK {
		t.Fatalf("disable trigger status %d", disableResp.StatusCode)
	}

	templateBody, _ := json.Marshal(map[string]any{
		"templates": []map[string]string{
			{
				"trigger_type": "balance_shortfall",
				"template":     "Не хватает {amount}",
			},
		},
	})
	templateResp, err := env.authedRequest(http.MethodPut, "/api/v1/user/notifications", bytes.NewReader(templateBody))
	if err != nil {
		t.Fatal(err)
	}
	defer templateResp.Body.Close()
	if templateResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when saving balance_shortfall template with disabled trigger, got %d", templateResp.StatusCode)
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/user/notifications", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&payload)
	if payload["trigger_negative_balance"] != false {
		t.Fatalf("expected trigger_negative_balance=false, got %v", payload["trigger_negative_balance"])
	}
}

func setNotificationSecretKey(t *testing.T, env *testEnv) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"notification_secret_key": "12345678901234567890123456789012",
	})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/settings/notification-secret", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set notification secret status %d", resp.StatusCode)
	}
}
