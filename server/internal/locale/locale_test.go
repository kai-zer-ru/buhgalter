package locale_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/locale"
)

func TestMain(m *testing.M) {
	dir := locale.ResolveDir("")
	if dir == "" {
		os.Exit(1)
	}
	if err := locale.Load(dir); err != nil {
		// tests may run from server/ — try parent path
		alt := filepath.Join("..", "..", "locales")
		if err2 := locale.Load(alt); err2 != nil {
			panic(err)
		}
	}
	os.Exit(m.Run())
}

func TestLocaleRussianDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	msg := locale.T(req, "UNAUTHORIZED", "fallback")
	if msg != "Требуется авторизация" {
		t.Fatalf("got %q", msg)
	}
}

func TestLocaleEnglish(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en-US")
	msg := locale.T(req, "UNAUTHORIZED", "fallback")
	if msg != "Authentication required" {
		t.Fatalf("got %q", msg)
	}
}

func TestLocalePasswordsMismatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en")
	msg := locale.T(req, "PASSWORDS_MISMATCH", "fallback")
	if msg != "Passwords do not match" {
		t.Fatalf("got %q", msg)
	}
}
