package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	appmw "github.com/kai-zer-ru/buhgalter/internal/middleware"
)

func testDB(t *testing.T) *db.Handle {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return db.NewHandle(mgr)
}

func seedAdmin(t *testing.T, mgr *db.Handle) string {
	t.Helper()
	ctx := context.Background()
	hash, err := HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := CreateUser(ctx, mgr.DB(), "admin", hash, "Admin", true, UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	return userID
}

func testHandler(t *testing.T, mgr *db.Handle) *Handler {
	t.Helper()
	dir := t.TempDir()
	return &Handler{
		Store:        mgr,
		Audit:        audit.New(filepath.Join(dir, "audit")),
		Logger:       slog.Default(),
		LoginLimiter: appmw.NewIPRateLimiter(100, time.Minute),
	}
}

func TestHandlerLoginLogoutMe(t *testing.T) {
	mgr := testDB(t)
	seedAdmin(t, mgr)
	h := testHandler(t, mgr)

	loginBody, _ := json.Marshal(map[string]string{"login": "admin", "password": "secret123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rec.Code, rec.Body.String())
	}
	var loginResp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&loginResp); err != nil {
		t.Fatal(err)
	}
	if loginResp.Token == "" || loginResp.User.Login != "admin" {
		t.Fatalf("login resp: %+v", loginResp)
	}

	cookie := rec.Result().Cookies()[0]
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	mw := RequireAuth(mgr)
	mw(http.HandlerFunc(h.Me)).ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status %d", meRec.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq = logoutReq.WithContext(context.WithValue(logoutReq.Context(), AuthContextKey, AuthInfo{
		User: loginResp.User, Token: loginResp.Token,
	}))
	logoutRec := httptest.NewRecorder()
	h.Logout(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status %d", logoutRec.Code)
	}
}

func TestHandlerLoginInvalidCredentials(t *testing.T) {
	mgr := testDB(t)
	seedAdmin(t, mgr)
	h := testHandler(t, mgr)

	body, _ := json.Marshal(map[string]string{"login": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerRegister(t *testing.T) {
	mgr := testDB(t)
	seedAdmin(t, mgr)
	_, err := mgr.DB().Exec(`UPDATE system_settings SET registration_enabled = 1 WHERE id = 1`)
	if err != nil {
		t.Fatal(err)
	}
	h := testHandler(t, mgr)

	body, _ := json.Marshal(map[string]string{
		"login": "newuser", "password": "userpass1", "password_confirm": "userpass1",
		"display_name": "New",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d: %s", rec.Code, rec.Body.String())
	}
	var regResp struct {
		User User `json:"user"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&regResp); err != nil {
		t.Fatal(err)
	}
	if regResp.User.Status != string(UserStatusPending) {
		t.Fatalf("expected pending status, got %q", regResp.User.Status)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == SessionCookieName && c.Value != "" {
			t.Fatal("expected no session cookie after register")
		}
	}
}

func TestHandlerVerifyToken(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	token, err := CreateSession(context.Background(), mgr.DB(), userID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	h := testHandler(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify?token="+token, nil)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)
	var resp verifyResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.Valid {
		t.Fatal("expected valid token")
	}
}

func TestRequireAuthAndAdmin(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	token, err := CreateSession(context.Background(), mgr.DB(), userID, "", "")
	if err != nil {
		t.Fatal(err)
	}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, ok := FromContext(r.Context())
		if !ok || !info.User.IsAdmin {
			t.Fatal("expected admin in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	RequireAuth(mgr)(RequireAdmin(okHandler)).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestRequireAuthUnauthorized(t *testing.T) {
	mgr := testDB(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	RequireAuth(mgr)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestSessionCookieHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()
	SetSessionCookie(rec, req, "tok")
	ClearSessionCookie(rec, req)
	if len(rec.Result().Cookies()) < 2 {
		t.Fatal("expected cookies")
	}
}

func TestDeleteSessionByToken(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	token, err := CreateSession(context.Background(), mgr.DB(), userID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := DeleteSessionByToken(context.Background(), mgr.DB(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := LookupSession(context.Background(), mgr.DB(), token); err == nil {
		t.Fatal("session should be deleted")
	}
}

func TestLoadUser(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	u, err := LoadUser(context.Background(), mgr.DB(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if u.Login != "admin" || !u.IsAdmin {
		t.Fatalf("user: %+v", u)
	}
}

func TestLookupAPIToken(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	raw := "bhg_apitoken123456789012345"
	hash := HashToken(raw)
	_, err := mgr.DB().Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix)
		VALUES ('t1', ?, 'api', ?, 'bhg_apit')`, userID, hash)
	if err != nil {
		t.Fatal(err)
	}
	got, err := LookupAPIToken(context.Background(), mgr.DB(), raw)
	if err != nil || got != userID {
		t.Fatalf("LookupAPIToken: %s %v", got, err)
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if ip := ClientIP(req); ip == "" {
		t.Fatal("expected ip")
	}
}

func TestRequireAPIToken(t *testing.T) {
	mgr := testDB(t)
	userID := seedAdmin(t, mgr)
	raw := "bhg_handlerapitoken12345678901"
	hash := HashToken(raw)
	_, err := mgr.DB().Exec(`
		INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix)
		VALUES ('t2', ?, 'api', ?, 'bhg_hand')`, userID, hash)
	if err != nil {
		t.Fatal(err)
	}

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, ok := FromContext(r.Context())
		if !ok || !info.APIToken {
			t.Fatal("expected api token auth")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rec := httptest.NewRecorder()
	RequireAPIToken(mgr)(okHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerRegisterDisabled(t *testing.T) {
	mgr := testDB(t)
	seedAdmin(t, mgr)
	h := testHandler(t, mgr)

	body, _ := json.Marshal(map[string]string{
		"login": "x", "password": "userpass1", "password_confirm": "userpass1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Register(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerLoginValidation(t *testing.T) {
	mgr := testDB(t)
	h := testHandler(t, mgr)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("{}")))
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestLoginRateLimited(t *testing.T) {
	mgr := testDB(t)
	seedAdmin(t, mgr)
	h := &Handler{
		Store:        mgr,
		Audit:        audit.New(filepath.Join(t.TempDir(), "audit")),
		Logger:       slog.Default(),
		LoginLimiter: appmw.NewIPRateLimiter(1, time.Minute),
	}
	body, _ := json.Marshal(map[string]string{"login": "admin", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.RemoteAddr = "9.9.9.9:1234"
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.Login(rec, req)
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d status %d", i, rec.Code)
		}
	}
}

func TestRequireAdminForbidden(t *testing.T) {
	mgr := testDB(t)
	hash, _ := HashPassword("secret123")
	userID, _ := CreateUser(context.Background(), mgr.DB(), "user", hash, "U", false, UserStatusActive)
	token, _ := CreateSession(context.Background(), mgr.DB(), userID, "", "")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	RequireAuth(mgr)(RequireAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestGenerateToken(t *testing.T) {
	raw, hash, err := GenerateToken()
	if err != nil || raw == "" || hash == "" {
		t.Fatalf("token %q hash %q err %v", raw, hash, err)
	}
	if HashToken(raw) != hash {
		t.Fatal("hash mismatch")
	}
}

func TestHandlerMeUnauthorized(t *testing.T) {
	mgr := testDB(t)
	h := testHandler(t, mgr)
	rec := httptest.NewRecorder()
	h.Me(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerVerifyEmptyToken(t *testing.T) {
	mgr := testDB(t)
	h := testHandler(t, mgr)
	rec := httptest.NewRecorder()
	h.Verify(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/verify", nil))
	var resp verifyResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Valid {
		t.Fatal("expected invalid")
	}
}

func TestCreateUserDuplicateLogin(t *testing.T) {
	mgr := testDB(t)
	ctx := context.Background()
	hash, _ := HashPassword("secret123")
	_, err := CreateUser(ctx, mgr.DB(), "dup", hash, "D", false, UserStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateUser(ctx, mgr.DB(), "dup", hash, "D2", false, UserStatusActive)
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestExtractTokenBearer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer mytoken")
	if got := extractToken(req); got != "mytoken" {
		t.Fatalf("got %q", got)
	}
}
