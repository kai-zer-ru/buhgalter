package importexport

import (
	"bytes"
	"context"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
)

func seedImportHandle(t *testing.T) (context.Context, *db.Handle, string) {
	t.Helper()
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	ctx := context.Background()
	sqlDB := mgr.DB()
	hash, err := auth.HashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := auth.CreateUser(ctx, sqlDB, "importuser", hash, "Import", false)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, db.NewHandle(mgr), userID
}

func importAuthRequest(userID string, req *http.Request) *http.Request {
	user := auth.User{ID: userID, Login: "importuser", DisplayName: "Import"}
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, auth.AuthInfo{User: user})
	return req.WithContext(ctx)
}

func TestHandlerExport(t *testing.T) {
	ctx, handle, userID := seedImportHandle(t)
	data := sampleCSVRows()
	_, err := Import(ctx, handle.DB(), userID, "sample.csv", data, ImportOptions{
		Preset: "cubux", Deduplicate: true, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit")), Logger: slog.Default()}
	req := importAuthRequest(userID, httptest.NewRequest(http.MethodGet, "/export", nil))
	rec := httptest.NewRecorder()
	h.Export(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("export %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Fatal("expected content type")
	}
}

func TestHandlerPreviewMultipart(t *testing.T) {
	ctx, handle, userID := seedImportHandle(t)
	_ = ctx
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("preset", "cubux")
	_ = mw.WriteField("deduplicate", "true")
	part, _ := mw.CreateFormFile("file", "sample.csv")
	_, _ = part.Write(sampleCSVRows())
	_ = mw.Close()

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit")), Logger: slog.Default()}
	req := importAuthRequest(userID, httptest.NewRequest(http.MethodPost, "/import/preview", &buf))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Preview(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("preview %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerImportMultipart(t *testing.T) {
	_, handle, userID := seedImportHandle(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("preset", "cubux")
	_ = mw.WriteField("deduplicate", "true")
	_ = mw.WriteField("confirm", "true")
	part, _ := mw.CreateFormFile("file", "sample.csv")
	_, _ = part.Write(sampleCSVRows())
	_ = mw.Close()

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit")), Logger: slog.Default()}
	req := importAuthRequest(userID, httptest.NewRequest(http.MethodPost, "/import", &buf))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Idempotency-Key", "handler-idem-1")
	rec := httptest.NewRecorder()
	h.Import(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("import %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerHeadersMultipart(t *testing.T) {
	_, handle, userID := seedImportHandle(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "sample.csv")
	_, _ = part.Write(sampleCSVRows())
	_ = mw.Close()

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit")), Logger: slog.Default()}
	req := importAuthRequest(userID, httptest.NewRequest(http.MethodPost, "/import/headers", &buf))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Headers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("headers %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerImportWithoutConfirm(t *testing.T) {
	_, handle, userID := seedImportHandle(t)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("preset", "cubux")
	part, _ := mw.CreateFormFile("file", "sample.csv")
	_, _ = part.Write(sampleCSVRows())
	_ = mw.Close()

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit")), Logger: slog.Default()}
	req := importAuthRequest(userID, httptest.NewRequest(http.MethodPost, "/import", &buf))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Import(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerExportUnauthorized(t *testing.T) {
	_, handle, _ := seedImportHandle(t)
	h := &Handler{Store: handle, Audit: audit.New(t.TempDir()), Logger: slog.Default()}
	rec := httptest.NewRecorder()
	h.Export(rec, httptest.NewRequest(http.MethodGet, "/export", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}
