package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

func withAuth(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), auth.AuthContextKey, auth.AuthInfo{
		User: auth.User{ID: "u1", Login: "test", Language: "ru"},
	}))
}

func TestI18nUnauthorized(t *testing.T) {
	h := &Handler{Version: "1.5.0"}
	r := chi.NewRouter()
	r.Get("/ui/i18n/{lang}", h.I18n)

	req := httptest.NewRequest(http.MethodGet, "/ui/i18n/ru", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", rec.Code)
	}
}

func TestI18nInvalidLang(t *testing.T) {
	h := &Handler{Version: "1.5.0"}
	r := chi.NewRouter()
	r.Get("/ui/i18n/{lang}", h.I18n)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/ui/i18n/de", nil))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestI18nOK(t *testing.T) {
	h := &Handler{Version: "1.5.0"}
	r := chi.NewRouter()
	r.Get("/ui/i18n/{lang}", h.I18n)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/ui/i18n/ru", nil))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var body I18nResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Version != "1.5.0" {
		t.Fatalf("version %q", body.Version)
	}
	if body.Lang != "ru" {
		t.Fatalf("lang %q", body.Lang)
	}
	if body.Messages["app.title"] == "" {
		t.Fatalf("missing app.title in messages")
	}
	if body.Messages["nav.home"] == "" {
		t.Fatalf("missing nav.home in messages")
	}
}

func TestI18nEn(t *testing.T) {
	h := &Handler{Version: "1.4.1"}
	r := chi.NewRouter()
	r.Get("/ui/i18n/{lang}", h.I18n)

	req := withAuth(httptest.NewRequest(http.MethodGet, "/ui/i18n/en", nil))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body I18nResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Lang != "en" || body.Messages["app.title"] == "" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
