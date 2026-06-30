package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/recurring"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func TestRecurringApplyDueUpdatesBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	ctx := context.Background()

	accID := createTestAccount(t, env, "Периодический")
	catID := getExpenseCategory(t, env)

	startDate := timeutil.FormatUTC(timeutil.NowUTC().AddDate(0, -1, 0))
	body, _ := json.Marshal(map[string]any{
		"type":         "expense",
		"amount":       "75.00",
		"account_id":   accID,
		"category_id":  catID,
		"period":       "month",
		"day_of_month": timeutil.NowUTC().Day(),
		"start_date":   startDate,
		"time_local":   "08:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/recurring-operations", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create recurring status %d", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	past := timeutil.FormatUTC(timeutil.NowUTC().Add(-time.Hour))
	_, err = env.db.ExecContext(ctx, `UPDATE recurring_operations SET next_run_at = ? WHERE id = ?`, past, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	applied, err := recurring.ApplyDue(ctx, env.db, mustAdminUserID(t, env), timeutil.NowUTC(), "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied recurring operation, got %d", applied)
	}

	balResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 92_500 {
		t.Fatalf("expected balance 92500 after expense 75 from 100000, got %d", bal.Balance)
	}
}

func mustAdminUserID(t *testing.T, env *testEnv) string {
	t.Helper()
	resp, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var me struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&me)
	if me.ID == "" {
		t.Fatal("expected admin user id")
	}
	return me.ID
}
