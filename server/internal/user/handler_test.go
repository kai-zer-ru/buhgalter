package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
)

func testEnv(t *testing.T) (*Handler, *db.Handle, auth.User) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	handle := db.NewHandle(mgr)

	ctx := context.Background()
	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, mgr.DB(), "testuser", hash, "Test User", false, auth.UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.LoadUser(ctx, mgr.DB(), userID)
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{
		Store: handle,
		Audit: audit.New(filepath.Join(dir, "audit")),
	}
	return h, handle, *user
}

func withUser(t *testing.T, user auth.User, method, path string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, auth.AuthInfo{User: user})
	return req.WithContext(ctx)
}

func withUserIDParam(t *testing.T, user auth.User, method, path, id string, body []byte) *http.Request {
	t.Helper()
	req := withUser(t, user, method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func decodeAPIError(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error body: %v body=%s", err, rec.Body.String())
	}
	return payload.Error.Message
}

func setNotificationSecret(t *testing.T, handle *db.Handle) {
	t.Helper()
	_, err := handle.DB().ExecContext(context.Background(),
		`UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`,
		"12345678901234567890123456789012",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerUnauthorized(t *testing.T) {
	h, _, _ := testEnv(t)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, httptest.NewRequest(http.MethodGet, "/user/settings", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("GetSettings status %d", rec.Code)
	}
}

func TestHandlerGetPutSettings(t *testing.T) {
	h, _, user := testEnv(t)

	getRec := httptest.NewRecorder()
	h.GetSettings(getRec, withUser(t, user, http.MethodGet, "/user/settings", nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", getRec.Code, getRec.Body.String())
	}
	var got map[string]string
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got["display_name"] != "Test User" {
		t.Fatalf("display_name %q", got["display_name"])
	}

	putBody, _ := json.Marshal(map[string]string{
		"display_name": "Renamed",
		"language":     "en",
		"currency":     "USD",
		"timezone":     "Europe/London",
		"theme":        "dark",
	})
	putRec := httptest.NewRecorder()
	h.PutSettings(putRec, withUser(t, user, http.MethodPut, "/user/settings", putBody))
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated map[string]string
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated["display_name"] != "Renamed" || updated["language"] != "en" || updated["theme"] != "dark" {
		t.Fatalf("updated %+v", updated)
	}

	systemBody, _ := json.Marshal(map[string]string{"theme": "system"})
	systemRec := httptest.NewRecorder()
	h.PutSettings(systemRec, withUser(t, user, http.MethodPut, "/user/settings", systemBody))
	if systemRec.Code != http.StatusOK {
		t.Fatalf("put system theme status %d: %s", systemRec.Code, systemRec.Body.String())
	}
	var systemUpdated map[string]string
	if err := json.NewDecoder(systemRec.Body).Decode(&systemUpdated); err != nil {
		t.Fatal(err)
	}
	if systemUpdated["theme"] != "system" {
		t.Fatalf("theme %q", systemUpdated["theme"])
	}
}

func TestHandlerPutSettingsValidation(t *testing.T) {
	h, _, user := testEnv(t)

	cases := []struct {
		name string
		body string
	}{
		{"language", `{"language":"de"}`},
		{"currency", `{"currency":"GBP"}`},
		{"theme", `{"theme":"auto"}`},
		{"timezone", `{"timezone":"Not/AZone"}`},
		{"invalid json", `{`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.PutSettings(rec, withUser(t, user, http.MethodPut, "/user/settings", []byte(tc.body)))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlerChangePassword(t *testing.T) {
	h, handle, user := testEnv(t)

	t.Run("success", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password":     "secret123",
			"new_password":         "newpass12",
			"new_password_confirm": "newpass12",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		_, hash, err := auth.LoadUserByLogin(context.Background(), handle.DB(), user.Login)
		if err != nil {
			t.Fatal(err)
		}
		ok, err := auth.VerifyPassword(hash, "newpass12")
		if err != nil || !ok {
			t.Fatal("expected new password to work")
		}
	})

	t.Run("old_password alias", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"old_password":         "newpass12",
			"new_password":         "another12",
			"new_password_confirm": "another12",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("wrong current", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password":     "wrong",
			"new_password":         "another99",
			"new_password_confirm": "another99",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password":     "another12",
			"new_password":         "mismatch1",
			"new_password_confirm": "mismatch2",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
		if msg := decodeAPIError(t, rec); !strings.Contains(strings.ToLower(msg), "совпада") && !strings.Contains(strings.ToLower(msg), "match") {
			t.Fatalf("message %q", msg)
		}
	})

	t.Run("unchanged", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password":     "another12",
			"new_password":         "another12",
			"new_password_confirm": "another12",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("too weak", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"current_password":     "another12",
			"new_password":         "short",
			"new_password_confirm": "short",
		})
		rec := httptest.NewRecorder()
		h.ChangePassword(rec, withUser(t, user, http.MethodPut, "/user/password", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})
}

func TestHandlerTokenCRUD(t *testing.T) {
	h, _, user := testEnv(t)

	listRec := httptest.NewRecorder()
	h.ListTokens(listRec, withUser(t, user, http.MethodGet, "/user/tokens", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status %d", listRec.Code)
	}
	var listed []tokenListItem
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected empty list, got %d", len(listed))
	}

	createBody, _ := json.Marshal(map[string]any{"name": "  HA  "})
	createRec := httptest.NewRecorder()
	h.CreateToken(createRec, withUser(t, user, http.MethodPost, "/user/tokens", createBody))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status %d: %s", createRec.Code, createRec.Body.String())
	}
	var created createTokenResponse
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "HA" || created.Token == "" || created.ExpiresAt == nil {
		t.Fatalf("created %+v", created)
	}

	listRec2 := httptest.NewRecorder()
	h.ListTokens(listRec2, withUser(t, user, http.MethodGet, "/user/tokens", nil))
	if err := json.NewDecoder(listRec2.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "HA" {
		t.Fatalf("listed %+v", listed)
	}

	delRec := httptest.NewRecorder()
	h.DeleteToken(delRec, withUserIDParam(t, user, http.MethodDelete, "/user/tokens/"+created.ID, created.ID, nil))
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete status %d", delRec.Code)
	}

	missingRec := httptest.NewRecorder()
	h.DeleteToken(missingRec, withUserIDParam(t, user, http.MethodDelete, "/user/tokens/missing", "missing", nil))
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status %d", missingRec.Code)
	}
}

func TestHandlerCreateTokenValidation(t *testing.T) {
	h, _, user := testEnv(t)

	t.Run("empty name", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": "   "})
		rec := httptest.NewRecorder()
		h.CreateToken(rec, withUser(t, user, http.MethodPost, "/user/tokens", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("never expires", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Perpetual", "never_expires": true})
		rec := httptest.NewRecorder()
		h.CreateToken(rec, withUser(t, user, http.MethodPost, "/user/tokens", body))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
		var created createTokenResponse
		_ = json.NewDecoder(rec.Body).Decode(&created)
		if !created.NeverExpires || created.ExpiresAt != nil {
			t.Fatalf("created %+v", created)
		}
	})

	t.Run("past expiry", func(t *testing.T) {
		past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		body, _ := json.Marshal(map[string]any{"name": "Expired", "expires_at": past})
		rec := httptest.NewRecorder()
		h.CreateToken(rec, withUser(t, user, http.MethodPost, "/user/tokens", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})
}

func TestHandlerNotifications(t *testing.T) {
	h, handle, user := testEnv(t)
	setNotificationSecret(t, handle)

	getRec := httptest.NewRecorder()
	h.GetNotifications(getRec, withUser(t, user, http.MethodGet, "/user/notifications", nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", getRec.Code, getRec.Body.String())
	}

	putBody, _ := json.Marshal(map[string]any{
		"telegram_enabled":   true,
		"telegram_bot_token": "bot-token",
		"telegram_chat_id":   "12345",
		"trigger_debt":       true,
	})
	putRec := httptest.NewRecorder()
	h.PutNotifications(putRec, withUser(t, user, http.MethodPut, "/user/notifications", putBody))
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status %d: %s", putRec.Code, putRec.Body.String())
	}
	var settings map[string]any
	if err := json.NewDecoder(putRec.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if settings["telegram_configured"] != true {
		t.Fatalf("settings %+v", settings)
	}

	previewBody, _ := json.Marshal(map[string]string{
		"trigger_type": "credit_payment",
		"template":     "Платёж {amount}",
	})
	previewRec := httptest.NewRecorder()
	h.PreviewNotificationTemplate(previewRec, withUser(t, user, http.MethodPost, "/user/notifications/templates/preview", previewBody))
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status %d: %s", previewRec.Code, previewRec.Body.String())
	}

	resetRec := httptest.NewRecorder()
	h.ResetNotificationTemplates(resetRec, withUser(t, user, http.MethodPost, "/user/notifications/templates/reset", []byte(`{"trigger_type":"test"}`)))
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset status %d: %s", resetRec.Code, resetRec.Body.String())
	}
}

func TestHandlerPutNotificationsRequiresSecretForTokens(t *testing.T) {
	h, _, user := testEnv(t)

	body, _ := json.Marshal(map[string]any{
		"telegram_bot_token": "bot-token",
	})
	rec := httptest.NewRecorder()
	h.PutNotifications(rec, withUser(t, user, http.MethodPut, "/user/notifications", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerSendNotificationTest(t *testing.T) {
	h, handle, user := testEnv(t)
	setNotificationSecret(t, handle)

	t.Run("invalid channel", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"channel": "email"})
		rec := httptest.NewRecorder()
		h.SendNotificationTest(rec, withUser(t, user, http.MethodPost, "/user/notifications/test", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("missing config", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"channel": notify.ChannelTelegram})
		rec := httptest.NewRecorder()
		h.SendNotificationTest(rec, withUser(t, user, http.MethodPost, "/user/notifications/test", body))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerPreviewNotificationInvalid(t *testing.T) {
	h, _, user := testEnv(t)

	body, _ := json.Marshal(map[string]string{
		"trigger_type": "unknown_trigger",
		"template":     "x",
	})
	rec := httptest.NewRecorder()
	h.PreviewNotificationTemplate(rec, withUser(t, user, http.MethodPost, "/user/notifications/templates/preview", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
