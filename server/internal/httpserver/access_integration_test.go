package httpserver_test

import (
	"bytes"
	"net/http"
	"testing"
)

func TestExternalAccessDeniedWithoutExternalURL(t *testing.T) {
	env := setupConfigured(t)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/setup/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "203.0.113.10:8765"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("public host status = %d, want 403", resp.StatusCode)
	}
}

func TestExternalAccessAllowedOnPrivateLANWithoutExternalURL(t *testing.T) {
	env := setupConfigured(t)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/setup/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "192.168.1.8:8765"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("LAN host status = %d, want 200", resp.StatusCode)
	}
}

func TestExternalAccessAllowedOnLocalhostWithoutExternalURL(t *testing.T) {
	env := setupConfigured(t)

	resp, err := http.Get(env.server.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("localhost status = %d, want 200", resp.StatusCode)
	}
}

func TestExternalAccessAllowedWithConfiguredExternalURL(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body := []byte(`{"registration_enabled":false,"external_url":"https://buhgalter.example.com"}`)
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin settings status = %d", resp.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "buhgalter.example.com"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("configured host status = %d, want 200", resp.StatusCode)
	}
}