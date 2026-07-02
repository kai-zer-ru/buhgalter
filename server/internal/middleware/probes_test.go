package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsNoiseProbe(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{"/.env", true},
		{"/.env.production", true},
		{"/.env.backup", true},
		{"/.env~", true},
		{"/.env?foo=bar", true},
		{"/api/v1/health", false},
		{"/settings", false},
		{"/docs", false},
	}
	for _, tc := range tests {
		if got := isNoiseProbe(tc.path); got != tc.want {
			t.Errorf("isNoiseProbe(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestRejectNoiseProbes(t *testing.T) {
	t.Parallel()
	called := false
	handler := RejectNoiseProbes(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	probe := httptest.NewRequest(http.MethodGet, "/.env", nil)
	probeRec := httptest.NewRecorder()
	handler.ServeHTTP(probeRec, probe)
	if probeRec.Code != http.StatusNotFound {
		t.Fatalf("probe status = %d, want 404", probeRec.Code)
	}
	if called {
		t.Fatal("next handler should not run for probe path")
	}

	ok := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	okRec := httptest.NewRecorder()
	handler.ServeHTTP(okRec, ok)
	if okRec.Code != http.StatusOK {
		t.Fatalf("normal path status = %d, want 200", okRec.Code)
	}
	if !called {
		t.Fatal("next handler should run for normal path")
	}
}
