package features

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func testDB(t *testing.T) *db.Handle {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	Invalidate()
	return db.NewHandle(mgr)
}

func TestSnapshotDefaultsAndOverrides(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()

	snap, err := Snapshot(ctx, store.DB())
	if err != nil {
		t.Fatal(err)
	}
	if snap[Registration] {
		t.Fatalf("registration default = true, want false")
	}
	if !snap[Budget] {
		t.Fatalf("budget default = false, want true")
	}

	if err := SetOne(ctx, store.DB(), Budget, false); err != nil {
		t.Fatal(err)
	}
	enabled, err := IsEnabled(ctx, store.DB(), Budget)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("budget still enabled after disable")
	}
}

func TestSetManyRejectsUnknownKey(t *testing.T) {
	store := testDB(t)
	err := SetMany(context.Background(), store.DB(), map[string]bool{"not_a_flag": true})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestRequireFeatureDisabled(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()
	if err := SetOne(ctx, store.DB(), Budget, false); err != nil {
		t.Fatal(err)
	}

	h := RequireFeature(store, Budget)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/budgets", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
	var body apperror.Response
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != apperror.FeatureDisabled {
		t.Fatalf("code = %s, want %s", body.Error.Code, apperror.FeatureDisabled)
	}
}

func TestRegistrationMigratedFromDefaults(t *testing.T) {
	store := testDB(t)
	ctx := context.Background()
	if err := SetOne(ctx, store.DB(), Registration, true); err != nil {
		t.Fatal(err)
	}
	enabled, err := IsEnabled(ctx, store.DB(), Registration)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("expected registration enabled")
	}
	if !strings.Contains(Registration, "reg") {
		t.Fatal("sanity")
	}
}
