package tag

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteErrorUnmapped(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tags", nil)
	if !writeError(rec, req, errors.New("unexpected")) {
		t.Fatal("expected handled error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
