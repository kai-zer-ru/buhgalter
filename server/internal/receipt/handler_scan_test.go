package receipt_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/db"
	"github.com/kai-zer-ru/buhgalter/internal/receipt"
	"github.com/kai-zer-ru/buhgalter/internal/receipt/fns"
)

type mockFetcher struct{}

func (mockFetcher) FetchReceipt(_ context.Context, qr string) (fns.ParsedReceipt, error) {
	params, err := fns.ParseQR(qr)
	if err != nil {
		return fns.ParsedReceipt{}, err
	}
	return fns.ParsedReceipt{
		FiscalDriveNumber:    params.FiscalDriveNumber,
		FiscalDocumentNumber: params.FiscalDocumentNumber,
		FiscalSign:           params.FiscalSign,
		TotalSum:             params.TotalSumKopecks,
		StoreName:            "Test Store",
		Items:                []fns.Item{{Name: "Item", Quantity: 1, Price: 10000, Sum: 10000}},
		FNSCheckURL:          fns.CheckURL(params),
	}, nil
}

func withUser(req *http.Request, userID string) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), auth.AuthContextKey, auth.AuthInfo{
		User: auth.User{ID: userID, Login: "scanuser"},
	}))
}

func TestScanJobFromImage(t *testing.T) {
	dir := t.TempDir()
	mgr, err := db.NewManager(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	userID := "user-scan-1"
	_, err = mgr.DB().ExecContext(context.Background(),
		`INSERT INTO users (id, login, password_hash, timezone) VALUES (?, 'scanuser', 'hash', 'UTC')`,
		userID,
	)
	if err != nil {
		t.Fatal(err)
	}

	h := &receipt.Handler{
		Store:   db.NewHandle(mgr),
		Audit:   audit.New(filepath.Join(dir, "audit")),
		Fetcher: mockFetcher{},
	}

	fixture := filepath.Join("..", "..", "..", "web", "e2e", "fixtures", "receipt-qr.png")
	img, err := os.ReadFile(fixture)
	if err != nil {
		t.Skip("fixture missing")
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("image", "receipt.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, bytes.NewReader(img)); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()

	req := withUser(httptest.NewRequest(http.MethodPost, "/api/v1/receipts/scan-jobs", body), userID)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.CreateScanJob(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	for {
		getReq := withUser(
			httptest.NewRequest(http.MethodGet, "/api/v1/receipts/scan-jobs/"+created.ID, nil),
			userID,
		)
		getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
		getRec := httptest.NewRecorder()
		h.GetScanJob(getRec, getReq)
		if getRec.Code != http.StatusOK {
			t.Fatalf("get status %d", getRec.Code)
		}
		var job struct {
			Status string `json:"status"`
			Result *struct {
				StoreName string `json:"store_name"`
			} `json:"result"`
			ErrorMessage *string `json:"error_message"`
		}
		if err := json.Unmarshal(getRec.Body.Bytes(), &job); err != nil {
			t.Fatal(err)
		}
		switch job.Status {
		case "done":
			if job.Result == nil || job.Result.StoreName != "Test Store" {
				t.Fatalf("unexpected result: %+v", job)
			}
			return
		case "failed":
			t.Fatalf("job failed: %v", job.ErrorMessage)
		}
		if time.Now().After(deadline) {
			t.Fatal("timeout waiting for scan job")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
