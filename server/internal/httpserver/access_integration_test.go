package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/httpserver"
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
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer mgr.Close()

	cfg := testConfig(dir)
	cfg.AllowedHosts = []string{"192.168.1.8"}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer closer.Close()

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":            "admin",
		"admin_display_name":     "Администратор",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
		"registration_enabled":   false,
		"external_url":           "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/setup/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "192.168.1.8:8765"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("LAN host status = %d, want 200", resp.StatusCode)
	}
}

func TestExternalAccessAllowedOnLocalhostWithoutAllowedHostsInEnv(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer mgr.Close()

	cfg := testConfig(dir)
	cfg.AllowedHosts = []string{"203.0.113.10"}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer closer.Close()

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":            "admin",
		"admin_display_name":     "Администратор",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
		"registration_enabled":   false,
		"external_url":           "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("localhost without env entry status = %d, want 200", resp.StatusCode)
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

func TestExternalAccessAllowedWithConfiguredAllowedHost(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer mgr.Close()

	cfg := testConfig(dir)
	cfg.AllowedHosts = []string{"203.0.113.10"}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	defer closer.Close()

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":            "admin",
		"admin_display_name":     "Администратор",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
		"registration_enabled":   false,
		"external_url":           "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/setup/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "203.0.113.10:8765"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("allowed host status = %d, want 200", resp.StatusCode)
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