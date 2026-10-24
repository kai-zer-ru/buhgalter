package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kai-zer-ru/buhgalter/internal/audit"
	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestCreditPaymentFullUpdateBlockedAndClockChangesTime(t *testing.T) {
	handle, env := seedEnvFull(t)
	ctx := context.Background()
	sqlDB := handle.DB()
	if _, err := sqlDB.ExecContext(ctx, `UPDATE users SET timezone = ? WHERE id = ?`, "Asia/Irkutsk", env.userID); err != nil {
		t.Fatal(err)
	}

	debit := "08:00"
	name := "Clock credit"
	created, err := credit.Create(ctx, sqlDB, env.userID, credit.CreateInput{
		Name:            &name,
		PrincipalAmount: 300_000,
		IssueDate:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		TermMonths:      6,
		PaymentInterval: credit.IntervalMonth,
		DebitAccountID:  env.accountID,
		DebitTimeLocal:  &debit,
	})
	if err != nil {
		t.Fatal(err)
	}

	payAt := time.Date(2026, 10, 23, 17, 0, 0, 0, time.UTC)
	paid, err := credit.PayNextScheduled(ctx, sqlDB, env.userID, created.ID, credit.PayPaymentInput{
		Amount:      created.MonthlyPayment,
		PaymentDate: payAt,
	})
	if err != nil {
		t.Fatal(err)
	}

	var txID, paymentDate string
	for _, p := range paid.Schedule {
		if p.IsApplied && p.TransactionID != nil {
			txID = *p.TransactionID
			paymentDate = p.PaymentDate
			break
		}
	}
	if txID == "" {
		t.Fatal("expected applied credit payment with a transaction")
	}

	before, err := GetByID(ctx, sqlDB, env.userID, txID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Update(ctx, sqlDB, env.userID, txID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          50,
		TransactionDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, credit.ErrCannotEditPayment) {
		t.Fatalf("full update: expected ErrCannotEditPayment, got %v", err)
	}

	h := &Handler{Store: handle, Audit: audit.New(filepath.Join(t.TempDir(), "audit"))}
	fullBody, _ := json.Marshal(map[string]any{
		"account_id":       env.accountID,
		"type":             "expense",
		"amount":           "1.00",
		"transaction_date": "2026-11-02 00:00:00",
	})
	fullRec := callTx(t, h, env, http.MethodPut, txID, fullBody, h.Update)
	if fullRec.Code != http.StatusBadRequest {
		t.Fatalf("full PUT status %d: %s", fullRec.Code, fullRec.Body.String())
	}

	badRec := callClock(t, h, env, txID, "25:99")
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid clock status %d: %s", badRec.Code, badRec.Body.String())
	}
	unchanged, err := GetByID(ctx, sqlDB, env.userID, txID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.TransactionDate != before.TransactionDate {
		t.Fatalf("invalid clock wrote %s, want %s", unchanged.TransactionDate, before.TransactionDate)
	}

	var balanceBefore int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT current_balance FROM accounts WHERE id = ?`, env.accountID).Scan(&balanceBefore); err != nil {
		t.Fatal(err)
	}

	okRec := callClock(t, h, env, txID, "23:15")
	if okRec.Code != http.StatusOK {
		t.Fatalf("clock status %d: %s", okRec.Code, okRec.Body.String())
	}
	var updated Transaction
	if err := json.NewDecoder(okRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}

	loc, err := time.LoadLocation("Asia/Irkutsk")
	if err != nil {
		t.Fatal(err)
	}
	got, err := timeutil.ParseFlexibleUTC(updated.TransactionDate)
	if err != nil {
		t.Fatal(err)
	}
	local := got.In(loc)
	if local.Year() != 2026 || local.Month() != time.October || local.Day() != 24 || local.Hour() != 23 || local.Minute() != 15 || local.Second() != 0 {
		t.Fatalf("local clock = %s, want 2026-10-24 23:15:00", local.Format(time.RFC3339))
	}
	if updated.TransactionDate != "2026-10-24 15:15:00" {
		t.Fatalf("stored utc %s, want 2026-10-24 15:15:00", updated.TransactionDate)
	}
	if updated.Amount != before.Amount || updated.AccountID != before.AccountID || deref(updated.Description) != deref(before.Description) {
		t.Fatalf("clock changed other fields: amount %d→%d account %s→%s desc %q→%q",
			before.Amount, updated.Amount, before.AccountID, updated.AccountID, deref(before.Description), deref(updated.Description))
	}
	if !updated.CreditPaymentLinked {
		t.Fatal("credit link flag cleared")
	}

	again, err := credit.GetByID(ctx, sqlDB, env.userID, created.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if again.DebitTimeLocal == nil || *again.DebitTimeLocal != "08:00" {
		t.Fatalf("debit_time_local = %v, want 08:00", again.DebitTimeLocal)
	}
	var balanceAfter int64
	if err := sqlDB.QueryRowContext(ctx, `SELECT current_balance FROM accounts WHERE id = ?`, env.accountID).Scan(&balanceAfter); err != nil {
		t.Fatal(err)
	}
	if balanceAfter != balanceBefore {
		t.Fatalf("current_balance %d, want %d after moving the clock later the same day", balanceAfter, balanceBefore)
	}
	if updated.Kind != "manual" {
		t.Fatalf("kind = %s, want manual", updated.Kind)
	}
	var sawPayment bool
	for _, p := range again.Schedule {
		if p.TransactionID != nil && *p.TransactionID == txID {
			sawPayment = true
			if p.PaymentDate != paymentDate {
				t.Fatalf("payment_date %s, want %s", p.PaymentDate, paymentDate)
			}
		}
	}
	if !sawPayment {
		t.Fatal("credit payment lost its transaction")
	}

	_, err = Update(ctx, sqlDB, env.userID, txID, UpdateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          before.Amount,
		TransactionDate: got,
	})
	if !errors.Is(err, credit.ErrCannotEditPayment) {
		t.Fatalf("full update after clock: expected ErrCannotEditPayment, got %v", err)
	}

	cat := env.expenseID
	plain, err := Create(ctx, sqlDB, env.userID, CreateInput{
		AccountID:       env.accountID,
		Type:            "expense",
		Amount:          100,
		CategoryID:      &cat,
		TransactionDate: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	plainRec := callClock(t, h, env, plain.ID, "10:00")
	if plainRec.Code != http.StatusBadRequest {
		t.Fatalf("plain expense clock status %d: %s", plainRec.Code, plainRec.Body.String())
	}
	plainAgain, err := GetByID(ctx, sqlDB, env.userID, plain.ID)
	if err != nil {
		t.Fatal(err)
	}
	if plainAgain.TransactionDate != plain.TransactionDate {
		t.Fatalf("plain expense date changed to %s", plainAgain.TransactionDate)
	}
}

func callTx(t *testing.T, h *Handler, env seedEnv, method, id string, body []byte, fn func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := authRequest(t, env, method, "/transactions/"+id, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

func callClock(t *testing.T, h *Handler, env seedEnv, id, clock string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"time": clock})
	if err != nil {
		t.Fatal(err)
	}
	return callTx(t, h, env, http.MethodPut, id, body, h.UpdateClock)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
