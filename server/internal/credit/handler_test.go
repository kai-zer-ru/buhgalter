package credit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func creditAuthRequest(t *testing.T, userID, method, path string, body []byte) *http.Request {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	user := auth.User{ID: userID, Login: "credituser"}
	ctx := context.WithValue(req.Context(), auth.AuthContextKey, auth.AuthInfo{User: user})
	return req.WithContext(ctx)
}

func TestHandlerCreateListPreview(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	_ = ctx
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	issue := timeutil.NowUTC().AddDate(0, 2, 0).Format("2006-01-02 00:00:00")

	createBody, _ := json.Marshal(map[string]any{
		"principal_amount":    "50000.00",
		"issue_date":          issue,
		"term_months":         6,
		"payment_interval":    "month",
		"debit_account_id":    accountID,
		"create_transactions": false,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", createBody))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create %d: %s", createRec.Code, createRec.Body.String())
	}
	var created Credit
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	listRec := httptest.NewRecorder()
	h.List(listRec, creditAuthRequest(t, userID, http.MethodGet, "/credits?status=active", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list %d", listRec.Code)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	getRec := httptest.NewRecorder()
	getReq := creditAuthRequest(t, userID, http.MethodGet, "/credits/"+created.ID, nil)
	getReq = getReq.WithContext(context.WithValue(getReq.Context(), chi.RouteCtxKey, rctx))
	h.Get(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get %d", getRec.Code)
	}

	previewBody, _ := json.Marshal(map[string]any{
		"principal": "10000.00", "term": 3, "payment_interval": "month",
		"issue_date": issue, "interest_rate": 0,
	})
	previewRec := httptest.NewRecorder()
	h.PreviewSchedule(previewRec, creditAuthRequest(t, userID, http.MethodPost, "/credits/preview-schedule", previewBody))
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview %d: %s", previewRec.Code, previewRec.Body.String())
	}

	upBody, _ := json.Marshal(map[string]any{"name": "Renamed"})
	upReq := creditAuthRequest(t, userID, http.MethodPut, "/credits/"+created.ID, upBody)
	upReq = upReq.WithContext(context.WithValue(upReq.Context(), chi.RouteCtxKey, rctx))
	upRec := httptest.NewRecorder()
	h.Update(upRec, upReq)
	if upRec.Code != http.StatusOK {
		t.Fatalf("update %d", upRec.Code)
	}

	closeBody, _ := json.Marshal(map[string]any{
		"payment_date": timeutil.NowUTC().Format("2006-01-02 15:04:05"),
	})
	closeReq := creditAuthRequest(t, userID, http.MethodPost, "/credits/"+created.ID+"/close", closeBody)
	closeReq = closeReq.WithContext(context.WithValue(closeReq.Context(), chi.RouteCtxKey, rctx))
	closeRec := httptest.NewRecorder()
	h.Close(closeRec, closeReq)
	if closeRec.Code != http.StatusOK {
		t.Fatalf("close %d", closeRec.Code)
	}

	schedRec := httptest.NewRecorder()
	schedReq := creditAuthRequest(t, userID, http.MethodGet, "/credits/"+created.ID+"/schedule", nil)
	schedReq = schedReq.WithContext(context.WithValue(schedReq.Context(), chi.RouteCtxKey, rctx))
	h.Schedule(schedRec, schedReq)
	if schedRec.Code != http.StatusOK {
		t.Fatalf("schedule %d", schedRec.Code)
	}

	payBody, _ := json.Marshal(map[string]any{
		"amount": "8333.33", "payment_date": timeutil.NowUTC().Format("2006-01-02 15:04:05"),
	})
	payReq := creditAuthRequest(t, userID, http.MethodPost, "/credits/"+created.ID+"/payments", payBody)
	payReq = payReq.WithContext(context.WithValue(payReq.Context(), chi.RouteCtxKey, rctx))
	payRec := httptest.NewRecorder()
	h.AddPayment(payRec, payReq)
	if payRec.Code != http.StatusOK && payRec.Code != http.StatusBadRequest {
		t.Fatalf("add payment %d: %s", payRec.Code, payRec.Body.String())
	}
}

func TestHandlerAddPaymentSuccess(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	_ = ctx
	issue := timeutil.NowUTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00")
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	body, _ := json.Marshal(map[string]any{
		"principal_amount": "60000.00", "issue_date": issue, "term_months": 6,
		"payment_interval": "month", "debit_account_id": accountID, "create_transactions": true,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", body))
	var created Credit
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	payBody, _ := json.Marshal(map[string]any{
		"amount":       money.FormatRubles(created.MonthlyPayment),
		"payment_date": timeutil.NowUTC().Format("2006-01-02 15:04:05"),
	})
	payReq := creditAuthRequest(t, userID, http.MethodPost, "/credits/"+created.ID+"/payments", payBody)
	payReq = payReq.WithContext(context.WithValue(payReq.Context(), chi.RouteCtxKey, rctx))
	payRec := httptest.NewRecorder()
	h.AddPayment(payRec, payReq)
	if payRec.Code != http.StatusOK {
		t.Fatalf("add payment %d: %s", payRec.Code, payRec.Body.String())
	}
}

func TestHandlerDeletePayment(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	issue := timeutil.NowUTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00")
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	body, _ := json.Marshal(map[string]any{
		"principal_amount": "12000.00", "issue_date": issue, "term_months": 6,
		"payment_interval": "month", "debit_account_id": accountID, "create_transactions": true,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", body))
	var created Credit
	_ = json.NewDecoder(createRec.Body).Decode(&created)
	var paymentID string
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	payBody, _ := json.Marshal(map[string]any{
		"amount":       money.FormatRubles(created.MonthlyPayment),
		"payment_date": timeutil.NowUTC().Format("2006-01-02 15:04:05"),
	})
	payReq := creditAuthRequest(t, userID, http.MethodPost, "/credits/"+created.ID+"/payments", payBody)
	payReq = payReq.WithContext(context.WithValue(payReq.Context(), chi.RouteCtxKey, rctx))
	payRec := httptest.NewRecorder()
	h.AddPayment(payRec, payReq)
	if payRec.Code != http.StatusOK {
		t.Fatalf("add payment %d: %s", payRec.Code, payRec.Body.String())
	}
	var paid Credit
	_ = json.NewDecoder(payRec.Body).Decode(&paid)
	for _, p := range paid.Schedule {
		if p.Kind == "scheduled" && p.IsApplied && p.TransactionID != nil {
			paymentID = p.ID
			break
		}
	}
	if paymentID == "" {
		t.Fatal("no applied payment")
	}
	rctx.URLParams.Add("paymentId", paymentID)
	delReq := creditAuthRequest(t, userID, http.MethodDelete, "/credits/"+created.ID+"/payments/"+paymentID, nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()
	h.DeletePayment(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("delete payment %d: %s", delRec.Code, delRec.Body.String())
	}
	_ = ctx
}

func TestHandlerDeleteCredit(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	issue := timeutil.NowUTC().AddDate(0, 2, 0).Format("2006-01-02 00:00:00")
	body, _ := json.Marshal(map[string]any{
		"principal_amount": "10000.00", "issue_date": issue, "term_months": 3,
		"payment_interval": "month", "debit_account_id": accountID, "create_transactions": false,
	})
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", body))
	var created Credit
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	delReq := creditAuthRequest(t, userID, http.MethodDelete, "/credits/"+created.ID+"?mode=cascade", nil)
	delReq = delReq.WithContext(context.WithValue(delReq.Context(), chi.RouteCtxKey, rctx))
	delRec := httptest.NewRecorder()
	h.Delete(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete %d: %s", delRec.Code, delRec.Body.String())
	}
	_ = ctx
}

func TestHandlerInvalidStatus(t *testing.T) {
	_, handle, userID, _ := seedCreditEnv(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.List(rec, creditAuthRequest(t, userID, http.MethodGet, "/credits?status=invalid", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerUnauthorized(t *testing.T) {
	_, handle, _, _ := seedCreditEnv(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/credits", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerInvalidCreateJSON(t *testing.T) {
	_, handle, userID, _ := seedCreditEnv(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.Create(rec, creditAuthRequest(t, userID, http.MethodPost, "/credits", []byte("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerGetNotFound(t *testing.T) {
	_, handle, userID, _ := seedCreditEnv(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "missing-id")
	req := creditAuthRequest(t, userID, http.MethodGet, "/credits/missing-id", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.Get(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestUpdateDebitAccount(t *testing.T) {
	ctx, handle, userID, accountID := seedCreditEnv(t)
	sqlDB := handle.DB()
	account2 := "acc-credit-2"
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, initial_balance, status, created_at, updated_at)
		VALUES (?, ?, 'Второй', 'cash', 0, 'active', datetime('now'), datetime('now'))`,
		account2, userID)
	if err != nil {
		t.Fatal(err)
	}
	issue := timeutil.NowUTC().AddDate(0, 2, 0)
	c, err := Create(ctx, sqlDB, userID, CreateInput{
		PrincipalAmount: 60_000, IssueDate: issue, TermMonths: 6,
		PaymentInterval: IntervalMonth, DebitAccountID: accountID, CreateTransactions: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	newDebit := account2
	updated, err := Update(ctx, sqlDB, userID, c.ID, UpdateInput{DebitAccountID: &newDebit})
	if err != nil {
		t.Fatal(err)
	}
	if updated.DebitAccountID != account2 {
		t.Fatalf("debit %s", updated.DebitAccountID)
	}
}

func TestHandlerUpdateCredit(t *testing.T) {
	_, handle, userID, accountID := seedCreditEnv(t)
	issue := timeutil.NowUTC().AddDate(0, 2, 0).Format("2006-01-02 00:00:00")
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	body, _ := json.Marshal(map[string]any{
		"principal_amount": "30000.00", "issue_date": issue, "term_months": 3,
		"payment_interval": "month", "debit_account_id": accountID, "create_transactions": false,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", body))
	var created Credit
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	upBody, _ := json.Marshal(map[string]any{"name": "Новое имя", "monthly_payment": "12000.00"})
	upReq := creditAuthRequest(t, userID, http.MethodPut, "/credits/"+created.ID, upBody)
	upReq = upReq.WithContext(context.WithValue(upReq.Context(), chi.RouteCtxKey, rctx))
	upRec := httptest.NewRecorder()
	h.Update(upRec, upReq)
	if upRec.Code != http.StatusOK {
		t.Fatalf("update %d: %s", upRec.Code, upRec.Body.String())
	}
}

func TestHandlerListClosedCredits(t *testing.T) {
	_, handle, userID, accountID := seedCreditEnv(t)
	issue := timeutil.NowUTC().AddDate(0, 2, 0).Format("2006-01-02 00:00:00")
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	body, _ := json.Marshal(map[string]any{
		"principal_amount": "10000.00", "issue_date": issue, "term_months": 3,
		"payment_interval": "month", "debit_account_id": accountID, "create_transactions": false,
	})
	createRec := httptest.NewRecorder()
	h.Create(createRec, creditAuthRequest(t, userID, http.MethodPost, "/credits", body))
	var created Credit
	_ = json.NewDecoder(createRec.Body).Decode(&created)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", created.ID)
	closeBody, _ := json.Marshal(map[string]any{
		"payment_date": timeutil.NowUTC().Format("2006-01-02 15:04:05"),
	})
	closeReq := creditAuthRequest(t, userID, http.MethodPost, "/credits/"+created.ID+"/close", closeBody)
	closeReq = closeReq.WithContext(context.WithValue(closeReq.Context(), chi.RouteCtxKey, rctx))
	closeRec := httptest.NewRecorder()
	h.Close(closeRec, closeReq)
	if closeRec.Code != http.StatusOK {
		t.Fatalf("close %d", closeRec.Code)
	}

	listRec := httptest.NewRecorder()
	h.List(listRec, creditAuthRequest(t, userID, http.MethodGet, "/credits?status=closed", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list closed %d", listRec.Code)
	}
}

func TestHandlerPreviewInvalidJSON(t *testing.T) {
	_, handle, userID, _ := seedCreditEnv(t)
	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	rec := httptest.NewRecorder()
	h.PreviewSchedule(rec, creditAuthRequest(t, userID, http.MethodPost, "/credits/preview-schedule", []byte("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
