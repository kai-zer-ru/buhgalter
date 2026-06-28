package apicache

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

func TestMiddlewareCachesGET(t *testing.T) {
	cache := New()
	calls := 0
	handler := Middleware(cache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/banks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || calls != 1 {
		t.Fatalf("first call: status=%d calls=%d", rec.Code, calls)
	}

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/banks", nil))
	if rec2.Code != http.StatusOK || calls != 1 {
		t.Fatalf("cached call: status=%d calls=%d body=%s", rec2.Code, calls, rec2.Body.String())
	}
	if rec2.Body.String() != `{"ok":true}` {
		t.Fatalf("body=%q", rec2.Body.String())
	}
}

func TestMiddlewareInvalidatesUserCacheOnWrite(t *testing.T) {
	cache := New()
	calls := 0
	handler := Middleware(cache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"n":%d}`, calls)
	}))

	userID := "user-1"
	withUser := func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), auth.AuthContextKey, auth.AuthInfo{
			User: auth.User{ID: userID},
		})
		return r.WithContext(ctx)
	}

	getReq := withUser(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil))
	handler.ServeHTTP(httptest.NewRecorder(), getReq)
	handler.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)))
	if calls != 1 {
		t.Fatalf("expected cache hit, calls=%d", calls)
	}

	postReq := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/transactions", nil))
	handler.ServeHTTP(httptest.NewRecorder(), postReq)
	handler.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)))
	if calls != 3 {
		t.Fatalf("expected cache miss after write, calls=%d", calls)
	}
}

func TestCacheExpires(t *testing.T) {
	cache := New()
	cache.Set("k", Response{Status: http.StatusOK, Body: []byte("x")}, time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	if _, ok := cache.Get("k"); ok {
		t.Fatal("expected expired entry")
	}
}

func TestMiddlewareInvalidatesSetupStatusOnAdminSettingsWrite(t *testing.T) {
	cache := New()
	calls := 0
	handler := Middleware(cache)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"n":%d}`, calls)
	}))

	userID := "user-1"
	withUser := func(r *http.Request) *http.Request {
		ctx := context.WithValue(r.Context(), auth.AuthContextKey, auth.AuthInfo{
			User: auth.User{ID: userID},
		})
		return r.WithContext(ctx)
	}

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil))
	if calls != 1 {
		t.Fatalf("expected setup status cache hit, calls=%d", calls)
	}

	handler.ServeHTTP(httptest.NewRecorder(), withUser(httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings", nil)))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil))
	if calls != 3 {
		t.Fatalf("expected setup status cache miss after admin settings write, calls=%d", calls)
	}
}
