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
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/httpserver"
	"github.com/kai-zer-ru/buhgalter/internal/notify"
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

	cfg := testConfig(dir)
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
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

	createBody, _ := json.Marshal(map[string]any{"name": "HA"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created struct {
		ID        string  `json:"id"`
		Token     string  `json:"token"`
		ExpiresAt *string `json:"expires_at"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if created.Token == "" {
		t.Fatal("expected token in response")
	}
	if created.ExpiresAt == nil || *created.ExpiresAt == "" {
		t.Fatal("expected default expiry (~1 month)")
	}
	exp, err := time.Parse(time.RFC3339, *created.ExpiresAt)
	if err != nil {
		t.Fatalf("parse expires_at: %v", err)
	}
	delta := exp.Sub(time.Now().UTC())
	if delta < 29*24*time.Hour || delta > 31*24*time.Hour {
		t.Fatalf("expected ~30 days expiry, got %v", delta)
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

func TestAPITokenNeverExpires(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]any{"name": "Perpetual", "never_expires": true})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created struct {
		Token        string  `json:"token"`
		NeverExpires bool    `json:"never_expires"`
		ExpiresAt    *string `json:"expires_at"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	if !created.NeverExpires || created.ExpiresAt != nil {
		t.Fatalf("expected perpetual token, got never_expires=%v expires_at=%v", created.NeverExpires, created.ExpiresAt)
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
		t.Fatal("perpetual api token should be valid")
	}
}

func TestAPITokenRevokeOtherUserForbidden(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]any{"name": "Admin token"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	userBody, _ := json.Marshal(map[string]any{
		"login": "user1", "password": "password1", "password_confirm": "password1", "display_name": "User", "is_admin": false,
	})
	userResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(userBody))
	if err != nil {
		t.Fatal(err)
	}
	userResp.Body.Close()

	env.login(t, "user1", "password1")
	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/user/tokens/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when revoking another user's token, got %d", delResp.StatusCode)
	}
}

func TestAPITokenExpiredRejected(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	createBody, _ := json.Marshal(map[string]any{"name": "Expired", "expires_at": past})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for past expiry, got %d", resp.StatusCode)
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

	cfg := testConfig(dir)
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
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

	cfg := testConfig(dir)
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
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

func TestPasswordResetRequestAdminFlow(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]any{
		"login": "bob", "password": "bobpass1", "password_confirm": "bobpass1",
		"display_name": "Bob", "is_admin": false,
	})
	createResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create user status %d", createResp.StatusCode)
	}

	reqBody, _ := json.Marshal(map[string]string{"login": "bob"})
	reqResp, err := http.Post(env.server.URL+"/api/v1/auth/request-password-reset", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	reqResp.Body.Close()
	if reqResp.StatusCode != http.StatusNoContent {
		t.Fatalf("request status %d", reqResp.StatusCode)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/password-reset-requests", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var items []map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("pending requests %v", items)
	}
	requestID, _ := items[0]["id"].(string)
	userID, _ := items[0]["user_id"].(string)

	ackResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/password-reset-requests/"+requestID+"/ack", nil)
	if err != nil {
		t.Fatal(err)
	}
	ackResp.Body.Close()
	if ackResp.StatusCode != http.StatusNoContent {
		t.Fatalf("ack status %d", ackResp.StatusCode)
	}

	listResp2, err := env.authedRequest(http.MethodGet, "/api/v1/admin/password-reset-requests", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp2.Body.Close()
	items = nil
	if err := json.NewDecoder(listResp2.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no pending after ack, got %v", items)
	}

	resetBody, _ := json.Marshal(map[string]string{
		"new_password": "newbobpass1", "new_password_confirm": "newbobpass1",
	})
	resetResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/users/"+userID+"/password", bytes.NewReader(resetBody))
	if err != nil {
		t.Fatal(err)
	}
	resetResp.Body.Close()
	if resetResp.StatusCode != http.StatusNoContent {
		t.Fatalf("reset password status %d", resetResp.StatusCode)
	}

	env.cookie = ""
	env.token = ""
	env.login(t, "bob", "newbobpass1")
}

func setupWithRegistration(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	cfg := testConfig(dir)
	logger, closer, err := httpserver.InitLogger(filepath.Join(dir, "logs"), "prod")
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
		"admin_display_name":     "Admin",
		"admin_password":         "secret123",
		"admin_password_confirm": "secret123",
		"registration_enabled":   true,
		"external_url":           "",
	})
	resp, err := http.Post(ts.URL+"/api/v1/setup", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	env := &testEnv{server: ts, db: mgr.DB(), dir: dir}
	env.login(t, "admin", "secret123")
	t.Cleanup(ts.Close)
	return env
}

func apiErrorCode(t *testing.T, resp *http.Response) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code
}

func TestUserStatusRegistrationAndModeration(t *testing.T) {
	env := setupWithRegistration(t)

	regBody, _ := json.Marshal(map[string]string{
		"login": "pending1", "password": "userpass1", "password_confirm": "userpass1",
		"display_name": "Pending",
	})
	regResp, err := http.Post(env.server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	defer regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register status %d", regResp.StatusCode)
	}
	var regResult struct {
		User struct {
			Status string `json:"status"`
		} `json:"user"`
	}
	if err := json.NewDecoder(regResp.Body).Decode(&regResult); err != nil {
		t.Fatal(err)
	}
	if regResult.User.Status != "pending" {
		t.Fatalf("expected pending, got %q", regResult.User.Status)
	}
	for _, c := range regResp.Cookies() {
		if c.Name == "session" && c.Value != "" {
			t.Fatal("expected no session cookie")
		}
	}

	loginBody, _ := json.Marshal(map[string]string{"login": "pending1", "password": "userpass1"})
	loginResp, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusForbidden {
		t.Fatalf("login pending status %d", loginResp.StatusCode)
	}
	if code := apiErrorCode(t, loginResp); code != "USER_PENDING_MODERATION" {
		t.Fatalf("expected USER_PENDING_MODERATION, got %q", code)
	}

	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer usersResp.Body.Close()
	var users []struct {
		ID     string `json:"id"`
		Login  string `json:"login"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(usersResp.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}
	var pendingID string
	for _, u := range users {
		if u.Login == "pending1" {
			pendingID = u.ID
			if u.Status != "pending" {
				t.Fatalf("list status %q", u.Status)
			}
		}
	}
	if pendingID == "" {
		t.Fatal("pending user not in list")
	}

	statusBody, _ := json.Marshal(map[string]string{"status": "active"})
	actResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/users/"+pendingID+"/status", bytes.NewReader(statusBody))
	if err != nil {
		t.Fatal(err)
	}
	actResp.Body.Close()
	if actResp.StatusCode != http.StatusOK {
		t.Fatalf("activate status %d", actResp.StatusCode)
	}

	loginResp2, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResp2.Body.Close()
	if loginResp2.StatusCode != http.StatusOK {
		t.Fatalf("login after activate status %d", loginResp2.StatusCode)
	}
}

func TestUserStatusRegistrationNotificationLog(t *testing.T) {
	env := setupWithRegistration(t)

	secret := "12345678901234567890123456789012"
	_, err := env.db.Exec(`UPDATE system_settings SET notification_secret_key = ? WHERE id = 1`, secret)
	if err != nil {
		t.Fatal(err)
	}
	box, err := notify.NewSecretBox(secret)
	if err != nil {
		t.Fatal(err)
	}
	token, err := box.Encrypt("telegram-token")
	if err != nil {
		t.Fatal(err)
	}
	var adminID string
	if err := env.db.QueryRow(`SELECT id FROM users WHERE login = 'admin'`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = env.db.Exec(`
		INSERT INTO notification_settings (
			user_id, telegram_enabled, telegram_bot_token, telegram_chat_id,
			max_enabled, trigger_debt, trigger_credit, trigger_planned,
			trigger_user_registration, debt_days_before, credit_days_before,
			notification_time_local, updated_at
		) VALUES (?, 1, ?, '12345', 0, 0, 0, 0, 1, 1, 1, '00:00', ?)
		ON CONFLICT(user_id) DO UPDATE SET
			telegram_enabled = 1,
			telegram_bot_token = excluded.telegram_bot_token,
			telegram_chat_id = excluded.telegram_chat_id,
			trigger_user_registration = 1`,
		adminID, token, now)
	if err != nil {
		t.Fatal(err)
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	regBody, _ := json.Marshal(map[string]string{
		"login": "notifyreg", "password": "userpass1", "password_confirm": "userpass1",
		"display_name": "Notify",
	})
	regResp, err := http.Post(env.server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register status %d", regResp.StatusCode)
	}

	var newUserID string
	if err := env.db.QueryRow(`SELECT id FROM users WHERE login = 'notifyreg'`).Scan(&newUserID); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := env.db.QueryRow(`
		SELECT COUNT(*) FROM notification_log
		WHERE user_id = ? AND trigger_type = ? AND entity_id = ?`,
		adminID, notify.TriggerUserRegistration, newUserID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected notification_log entry for user_registration")
	}
}

func TestUserStatusBanInvalidatesSession(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]any{
		"login": "banme", "password": "userpass1", "password_confirm": "userpass1",
		"display_name": "Ban", "is_admin": false,
	})
	createResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", createResp.StatusCode)
	}

	env.login(t, "banme", "userpass1")
	banmeCookie := env.cookie
	meResp, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me before ban %d", meResp.StatusCode)
	}

	env.login(t, "admin", "secret123")
	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer usersResp.Body.Close()
	var users []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	}
	_ = json.NewDecoder(usersResp.Body).Decode(&users)
	var banID string
	for _, u := range users {
		if u.Login == "banme" {
			banID = u.ID
		}
	}
	if banID == "" {
		t.Fatal("banme not found")
	}

	banBody, _ := json.Marshal(map[string]string{"status": "banned"})
	banResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/users/"+banID+"/status", bytes.NewReader(banBody))
	if err != nil {
		t.Fatal(err)
	}
	banResp.Body.Close()
	if banResp.StatusCode != http.StatusOK {
		t.Fatalf("ban status %d", banResp.StatusCode)
	}

	env.cookie = banmeCookie
	meResp2, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	meResp2.Body.Close()
	if meResp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me after ban %d, expected session invalidated", meResp2.StatusCode)
	}

	loginBody, _ := json.Marshal(map[string]string{"login": "banme", "password": "userpass1"})
	loginResp, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusForbidden {
		t.Fatalf("login banned status %d", loginResp.StatusCode)
	}
	if code := apiErrorCode(t, loginResp); code != "USER_BANNED" {
		t.Fatalf("expected USER_BANNED, got %q", code)
	}
}

func TestUserStatusSelfChangeForbidden(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer usersResp.Body.Close()
	var users []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(usersResp.Body).Decode(&users)
	if len(users) == 0 {
		t.Fatal("no users")
	}

	statusBody, _ := json.Marshal(map[string]string{"status": "banned"})
	selfResp, err := env.authedRequest(http.MethodPut, "/api/v1/admin/users/"+users[0].ID+"/status", bytes.NewReader(statusBody))
	if err != nil {
		t.Fatal(err)
	}
	selfResp.Body.Close()
	if selfResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("self status change %d", selfResp.StatusCode)
	}
}
