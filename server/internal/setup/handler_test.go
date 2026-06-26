package setup_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/httpserver"
)

func testServer(t *testing.T) (*httptest.Server, *db.Manager) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	cfg := config.Config{Version: "test", StaticEmbed: false, DataDir: dir}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	return httptest.NewServer(srv.Handler()), mgr
}

func TestHealth(t *testing.T) {
	ts, _ := testServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		DB      string `json:"db"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Version != "test" || body.DB != "connected" {
		t.Fatalf("body %+v", body)
	}
}

func TestHealthDBUnavailable(t *testing.T) {
	ts, mgr := testServer(t)
	defer ts.Close()
	if err := mgr.Close(); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(ts.URL + "/api/v1/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var body struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		DB      string `json:"db"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "error" || body.Version != "test" || body.DB != "error" {
		t.Fatalf("body %+v", body)
	}
}

func TestSetupSuccessAndConflict(t *testing.T) {
	ts, _ := testServer(t)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":              "admin",
		"admin_display_name":       "Администратор",
		"admin_password":           "secret123",
		"admin_password_confirm":   "secret123",
		"registration_enabled":     false,
		"external_url":             "https://buhgalter.mys-ite.ru",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup status %d", resp.StatusCode)
	}

	statusResp, err := http.Get(ts.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	var status struct {
		Configured  bool   `json:"configured"`
		ExternalURL string `json:"external_url"`
	}
	_ = json.NewDecoder(statusResp.Body).Decode(&status)
	if !status.Configured {
		t.Fatal("expected configured true")
	}
	if status.ExternalURL != "https://buhgalter.mys-ite.ru" {
		t.Fatalf("external_url = %q, want https://buhgalter.mys-ite.ru", status.ExternalURL)
	}

	resp2, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp2.StatusCode)
	}
}

func TestSetupPasswordMismatch(t *testing.T) {
	ts, _ := testServer(t)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":              "admin",
		"admin_password":           "secret123",
		"admin_password_confirm":   "different",
		"admin_display_name":       "Admin",
		"registration_enabled":     false,
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSetupMissingDisplayName(t *testing.T) {
	ts, _ := testServer(t)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":              "admin",
		"admin_display_name":       "",
		"admin_password":           "secret123",
		"admin_password_confirm":   "secret123",
		"registration_enabled":     false,
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSetupStatusSyncsMarkerFromDB(t *testing.T) {
	ts, mgr := testServer(t)
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{
		"admin_login":            "admin",
		"admin_display_name":     "Администратор",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("setup status %d", resp.StatusCode)
	}

	// Simulate partial failure after DB commit: marker removed while DB stays configured.
	dataDir := filepath.Dir(mgr.Path())
	if err := os.Remove(filepath.Join(dataDir, ".configured")); err != nil {
		t.Fatal(err)
	}

	statusResp, err := http.Get(ts.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	var status struct {
		Configured bool `json:"configured"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Configured {
		t.Fatal("status must sync marker from configured DB")
	}

	resp2, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp2.StatusCode)
	}
}

func TestSetupRestoreFromBackup(t *testing.T) {
	sourceTS, sourceMgr := testServer(t)
	defer sourceTS.Close()

	setupBody, _ := json.Marshal(map[string]any{
		"admin_login":            "admin",
		"admin_display_name":     "Администратор",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
	})
	setupResp, err := http.Post(sourceTS.URL+"/api/v1/setup", "application/json", bytes.NewReader(setupBody))
	if err != nil {
		t.Fatal(err)
	}
	_ = setupResp.Body.Close()
	if setupResp.StatusCode != http.StatusCreated {
		t.Fatalf("source setup status %d", setupResp.StatusCode)
	}

	targetTS, _ := testServer(t)
	defer targetTS.Close()

	sourceBackupSvc := &backup.Service{
		Manager:   sourceMgr,
		BackupDir: httpserver.BackupDir(filepath.Dir(sourceMgr.Path())),
	}
	backupName, err := sourceBackupSvc.Create()
	if err != nil {
		t.Fatal(err)
	}
	dbFile, err := os.Open(filepath.Join(sourceBackupSvc.BackupDir, backupName))
	if err != nil {
		t.Fatal(err)
	}
	defer dbFile.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("confirm", "RESTORE")
	part, err := writer.CreateFormFile("file", "backup.db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, dbFile); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, targetTS.URL+"/api/v1/setup/restore", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore status %d", resp.StatusCode)
	}

	var restoreBody struct {
		Configured bool `json:"configured"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&restoreBody); err != nil {
		t.Fatal(err)
	}
	if !restoreBody.Configured {
		t.Fatal("expected configured=true after restoring configured backup")
	}

	statusResp, err := http.Get(targetTS.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	var status struct {
		Configured bool `json:"configured"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.Configured {
		t.Fatal("expected configured=true in setup status")
	}
}
