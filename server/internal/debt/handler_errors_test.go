package debt

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteDebtErrorUnmapped(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/debts", nil)
	if !writeDebtError(rec, req, errors.New("unexpected")) {
		t.Fatal("expected handled error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWriteDebtorErrorUnmapped(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/debtors", nil)
	if !writeDebtorError(rec, req, errors.New("unexpected")) {
		t.Fatal("expected handled error")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestWriteDebtErrorNil(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/debts/x", nil)
	if writeDebtError(rec, req, nil) {
		t.Fatal("expected false for nil err")
	}
	if rec.Code != http.StatusOK && rec.Code != 0 {
		t.Fatalf("unexpected write on nil err: %d", rec.Code)
	}
}
