package httpserver_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/backup"
	"github.com/kai-zer-ru/buhgalter/internal/config"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/httpserver"
)

type testEnv struct {
	server *httptest.Server
	db     *sql.DB
	dir    string
	cookie string
	token  string
}

func setupConfigured(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	cfg := config.Config{
		Version:     "test",
		StaticEmbed: false,
		DataDir:     dir,
	}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatalf("logger: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())

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

	env := &testEnv{server: ts, db: mgr.DB(), dir: dir}
	t.Cleanup(ts.Close)
	return env
}

func (e *testEnv) login(t *testing.T, login, password string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"login": login, "password": password})
	resp, err := http.Post(e.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status %d", resp.StatusCode)
	}
	var result struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	e.token = result.Token
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			e.cookie = c.Value
		}
	}
}

func (e *testEnv) authedRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, e.server.URL+path, body)
	if err != nil {
		return nil, err
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: e.cookie})
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

func TestLoginMeLogout(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status %d", resp.StatusCode)
	}

	resp2, err := env.authedRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status %d", resp2.StatusCode)
	}

	resp3, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d", resp3.StatusCode)
	}
}

func TestChangePasswordUnchanged(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"current_password":     "secret123",
		"new_password":         "secret123",
		"new_password_confirm": "secret123",
	})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/user/password", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var errBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %q", errBody.Error.Code)
	}
}

func TestVerifyToken(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := http.Get(env.server.URL + "/api/v1/auth/verify?token=" + env.token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var v struct {
		Valid bool `json:"valid"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&v)
	if !v.Valid {
		t.Fatal("expected valid token")
	}

	resp2, err := http.Get(env.server.URL + "/api/v1/auth/verify?token=invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	_ = json.NewDecoder(resp2.Body).Decode(&v)
	if v.Valid {
		t.Fatal("expected invalid token")
	}
}

func TestAdminForbiddenForRegularUser(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]any{
		"login": "user1", "password": "password1", "password_confirm": "password1", "display_name": "User", "is_admin": false,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	env.login(t, "user1", "password1")
	resp2, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp2.StatusCode)
	}
}

func TestAdminDiagnostics(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/diagnostics", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body struct {
		AppVersion         string            `json:"app_version"`
		DBMigrationVersion int64             `json:"db_migration_version"`
		UsersCount         int64             `json:"users_count"`
		Env                map[string]string `json:"env"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.AppVersion == "" {
		t.Fatal("expected app_version")
	}
	if body.DBMigrationVersion <= 0 {
		t.Fatalf("expected migration version > 0, got %d", body.DBMigrationVersion)
	}
	if body.UsersCount < 1 {
		t.Fatalf("expected at least one user, got %d", body.UsersCount)
	}
	if _, ok := body.Env["BUHGALTER_ADDR"]; !ok {
		t.Fatal("expected BUHGALTER_ADDR in env")
	}
}

func TestRegisterDisabled(t *testing.T) {
	env := setupConfigured(t)
	body, _ := json.Marshal(map[string]string{
		"login": "newbie", "password": "password1", "display_name": "New",
	})
	resp, err := http.Post(env.server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestBackupCreateAndList(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/backups/run", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("backup run status %d", resp.StatusCode)
	}

	resp2, err := env.authedRequest(http.MethodGet, "/api/v1/admin/backups", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var files []struct {
		Filename string `json:"filename"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&files)
	if len(files) == 0 {
		t.Fatal("expected backup files")
	}
}

func TestLoginRateLimit(t *testing.T) {
	env := setupConfigured(t)
	body, _ := json.Marshal(map[string]string{"login": "admin", "password": "wrong"})

	var lastStatus int
	for i := 0; i < 7; i++ {
		resp, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		lastStatus = resp.StatusCode
		resp.Body.Close()
	}
	if lastStatus != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", lastStatus)
	}
}

func TestAPITokenLifecycle(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]any{"name": "HA", "expires_at": nil})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created.Token == "" {
		t.Fatal("expected token in response")
	}

	resp2, err := http.Get(env.server.URL + "/api/v1/auth/verify?token=" + created.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var v struct {
		Valid bool `json:"valid"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&v)
	if !v.Valid {
		t.Fatal("api token should be valid")
	}

	resp3, err := env.authedRequest(http.MethodDelete, "/api/v1/user/tokens/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp3.Body.Close()

	resp4, err := http.Get(env.server.URL + "/api/v1/auth/verify?token=" + created.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	_ = json.NewDecoder(resp4.Body).Decode(&v)
	if v.Valid {
		t.Fatal("revoked token should be invalid")
	}
}

func TestRestoreRequiresConfirm(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "NOPE")
	part, _ := w.CreateFormFile("file", "test.db")
	_, _ = part.Write([]byte("not a db"))
	_ = w.Close()

	req, _ := http.NewRequest(http.MethodPost, env.server.URL+"/api/v1/admin/backups/restore", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: env.cookie})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	_ = time.Now()
}

func TestRegisterPasswordMismatch(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	cfg := config.Config{
		Version:     "test",
		StaticEmbed: false,
		DataDir:     dir,
	}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	setupBody, _ := json.Marshal(map[string]any{
		"admin_login": "admin", "admin_display_name": "Admin",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123", "registration_enabled": true, "external_url": "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(setupBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	body, _ := json.Marshal(map[string]string{
		"login": "newbie", "password": "password1", "password_confirm": "different", "display_name": "New",
	})
	regResp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer regResp.Body.Close()
	if regResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", regResp.StatusCode)
	}
}

func TestRegisterPasswordTooWeak(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	cfg := config.Config{
		Version:     "test",
		StaticEmbed: false,
		DataDir:     dir,
	}
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	defer closer.Close()

	backupSvc := &backup.Service{Manager: mgr, BackupDir: httpserver.BackupDir(dir)}
	auditLogger := audit.New(filepath.Join(dir, "logs", "audit"))
	srv := httpserver.New(cfg, mgr, logger, auditLogger, backupSvc)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	setupBody, _ := json.Marshal(map[string]any{
		"admin_login": "admin", "admin_display_name": "Admin",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123", "registration_enabled": true, "external_url": "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(setupBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	body, _ := json.Marshal(map[string]string{
		"login": "newbie", "password": "12345678", "password_confirm": "12345678", "display_name": "New",
	})
	regResp, err := http.Post(ts.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer regResp.Body.Close()
	if regResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", regResp.StatusCode)
	}
	var errBody struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(regResp.Body).Decode(&errBody)
	if errBody.Error.Message == "" {
		t.Fatal("expected error message")
	}
}

func TestChangePasswordTooWeak(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"current_password":     "secret123",
		"new_password":         "12345678",
		"new_password_confirm": "12345678",
	})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/user/password", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestBackupRestore(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	runResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/backups/run", nil)
	if err != nil {
		t.Fatal(err)
	}
	runResp.Body.Close()
	if runResp.StatusCode != http.StatusCreated {
		t.Fatalf("backup run status %d", runResp.StatusCode)
	}

	markerBody, _ := json.Marshal(map[string]any{
		"login": "marker", "password": "password1", "password_confirm": "password1",
		"display_name": "Marker", "is_admin": false,
	})
	markerResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(markerBody))
	if err != nil {
		t.Fatal(err)
	}
	markerResp.Body.Close()
	if markerResp.StatusCode != http.StatusCreated {
		t.Fatalf("create marker user status %d", markerResp.StatusCode)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/backups", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var files []struct {
		Filename string `json:"filename"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&files)
	if len(files) == 0 {
		t.Fatal("no backup files")
	}

	backupPath := filepath.Join(env.dir, "backups", files[0].Filename)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("confirm", "RESTORE")
	part, _ := w.CreateFormFile("file", files[0].Filename)
	_, _ = part.Write(data)
	_ = w.Close()

	req, _ := http.NewRequest(http.MethodPost, env.server.URL+"/api/v1/admin/backups/restore", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: env.cookie})
	restoreResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	restoreResp.Body.Close()
	if restoreResp.StatusCode != http.StatusOK {
		t.Fatalf("restore status %d", restoreResp.StatusCode)
	}

	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer usersResp.Body.Close()
	var users []struct {
		Login string `json:"login"`
	}
	_ = json.NewDecoder(usersResp.Body).Decode(&users)
	for _, u := range users {
		if u.Login == "marker" {
			t.Fatal("marker user should be gone after restore")
		}
	}

	statusResp, err := http.Get(env.server.URL + "/api/v1/setup/status")
	if err != nil {
		t.Fatal(err)
	}
	defer statusResp.Body.Close()
	var status struct {
		Configured bool `json:"configured"`
	}
	_ = json.NewDecoder(statusResp.Body).Decode(&status)
	if !status.Configured {
		t.Fatal("setup status must stay configured after DB restore (instance marker)")
	}

	env.cookie = ""
	env.token = ""
	loginBody, _ := json.Marshal(map[string]string{"login": "admin", "password": "secret123"})
	loginResp, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login after restore without restart status %d", loginResp.StatusCode)
	}
}
