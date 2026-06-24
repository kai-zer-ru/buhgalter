package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func authRequest(t *testing.T, env seedEnv, method, path string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	user := auth.User{ID: env.userID, Login: "txuser", DisplayName: "Tx User"}
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, auth.AuthInfo{User: user})
	return req.WithContext(ctx)
}

func TestHandlerCreateListGetDelete(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")

	createBody, _ := json.Marshal(map[string]any{
		"account_id": env.accountID, "type": "expense", "amount": "25.00",
		"category_id": env.expenseID, "transaction_date": past,
	})
	rec := httptest.NewRecorder()
	h.Create(rec, authRequest(t, env, http.MethodPost, "/transactions", createBody))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %d: %s", rec.Code, rec.Body.String())
	}
	var created Transaction
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	listRec := httptest.NewRecorder()
	h.List(listRec, authRequest(t, env, http.MethodGet, "/transactions?type=expense&page=1&limit=10", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list %d", listRec.Code)
	}

	getReq := authRequest(t, env, http.MethodGet, "/transactions/"+created.ID, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	getRec := httptest.NewRecorder()
	h.Get(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get %d", getRec.Code)
	}

	delReq := authRequest(t, env, http.MethodDelete, "/transactions/"+created.ID, nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()
	h.Delete(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete %d", delRec.Code)
	}
}

func TestHandlerUnauthorized(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/transactions", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
	rec2 := httptest.NewRecorder()
	h.Create(rec2, httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader([]byte("{}"))))
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec2.Code)
	}
	_ = env
}

func TestHandlerInvalidCreateBody(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.Create(rec, authRequest(t, env, http.MethodPost, "/transactions", []byte("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerTransferAndBalance(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")

	body, _ := json.Marshal(map[string]any{
		"from_account_id": env.accountID, "to_account_id": env.account2,
		"amount": "10.00", "transaction_date": past,
	})
	rec := httptest.NewRecorder()
	h.CreateTransfer(rec, authRequest(t, env, http.MethodPost, "/transfers", body))
	if rec.Code != http.StatusCreated {
		t.Fatalf("transfer %d: %s", rec.Code, rec.Body.String())
	}

	balReq := authRequest(t, env, http.MethodGet, "/accounts/"+env.accountID+"/balance", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", env.accountID)
	balReq = balReq.WithContext(context.WithValue(balReq.Context(), chi.RouteCtxKey, rctx))
	balRec := httptest.NewRecorder()
	h.AccountBalance(balRec, balReq)
	if balRec.Code != http.StatusOK {
		t.Fatalf("balance %d", balRec.Code)
	}

	sumRec := httptest.NewRecorder()
	h.AccountsSummary(sumRec, authRequest(t, env, http.MethodGet, "/accounts/summary", nil))
	if sumRec.Code != http.StatusOK {
		t.Fatalf("summary %d", sumRec.Code)
	}

	dashRec := httptest.NewRecorder()
	h.Dashboard(dashRec, authRequest(t, env, http.MethodGet, "/dashboard", nil))
	if dashRec.Code != http.StatusOK {
		t.Fatalf("dashboard %d", dashRec.Code)
	}
}

func TestHandlerUpdateAndActivate(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	future := timeutil.NowUTC().Add(72 * time.Hour).Format("2006-01-02 15:04:05")

	createBody, _ := json.Marshal(map[string]any{
		"account_id": env.accountID, "type": "expense", "amount": "10.00",
		"category_id": env.expenseID, "transaction_date": future,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, authRequest(t, env, http.MethodPost, "/transactions", createBody))
	var created Transaction
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)

	actReq := authRequest(t, env, http.MethodPost, "/transactions/"+created.ID+"/activate", nil)
	actReq = actReq.WithContext(context.WithValue(actReq.Context(), chi.RouteCtxKey, rctx))
	actRec := httptest.NewRecorder()
	h.Activate(actRec, actReq)
	if actRec.Code != http.StatusOK {
		t.Fatalf("activate %d: %s", actRec.Code, actRec.Body.String())
	}

	updateBody, _ := json.Marshal(map[string]any{
		"account_id": env.accountID, "type": "expense", "amount": "15.00",
		"category_id": env.expenseID, "transaction_date": past,
	})
	upReq := authRequest(t, env, http.MethodPut, "/transactions/"+created.ID, updateBody)
	upReq = upReq.WithContext(context.WithValue(upReq.Context(), chi.RouteCtxKey, rctx))
	upRec := httptest.NewRecorder()
	h.Update(upRec, upReq)
	if upRec.Code != http.StatusOK {
		t.Fatalf("update %d: %s", upRec.Code, upRec.Body.String())
	}
}

func TestHandlerTransferUpdateDelete(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")

	body, _ := json.Marshal(map[string]any{
		"from_account_id": env.accountID, "to_account_id": env.account2,
		"amount": "20.00", "transaction_date": past,
	})
	rec := httptest.NewRecorder()
	h.CreateTransfer(rec, authRequest(t, env, http.MethodPost, "/transfers", body))
	var tr Transfer
	_ = json.NewDecoder(rec.Body).Decode(&tr)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("group_id", tr.GroupID)

	upBody, _ := json.Marshal(map[string]any{
		"from_account_id": env.accountID, "to_account_id": env.account2,
		"amount": "25.00", "transaction_date": past,
	})
	upReq := authRequest(t, env, http.MethodPut, "/transfers/"+tr.GroupID, upBody)
	upReq = upReq.WithContext(context.WithValue(upReq.Context(), chi.RouteCtxKey, rctx))
	upRec := httptest.NewRecorder()
	h.UpdateTransfer(upRec, upReq)
	if upRec.Code != http.StatusOK {
		t.Fatalf("update transfer %d", upRec.Code)
	}

	delReq := authRequest(t, env, http.MethodDelete, "/transfers/"+tr.GroupID, nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()
	h.DeleteTransfer(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete transfer %d", delRec.Code)
	}
}

func TestHandlerCreateInvalidAmount(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	body, _ := json.Marshal(map[string]any{
		"account_id": env.accountID, "type": "expense", "amount": "0",
		"category_id": env.expenseID, "transaction_date": past,
	})
	rec := httptest.NewRecorder()
	h.Create(rec, authRequest(t, env, http.MethodPost, "/transactions", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerListInvalidDate(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.List(rec, authRequest(t, env, http.MethodGet, "/transactions?from=bad-date", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerGetNotFound(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing-tx-id")
	req := authRequest(t, env, http.MethodGet, "/transactions/missing-tx-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerCreateTransferSameAccount(t *testing.T) {
	handle, env := seedEnvFull(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	past := timeutil.NowUTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	body, _ := json.Marshal(map[string]any{
		"from_account_id": env.accountID, "to_account_id": env.accountID,
		"amount": "10.00", "transaction_date": past,
	})
	rec := httptest.NewRecorder()
	h.CreateTransfer(rec, authRequest(t, env, http.MethodPost, "/transfers", body))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
