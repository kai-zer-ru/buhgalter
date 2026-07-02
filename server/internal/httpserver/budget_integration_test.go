package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestBudgetCRUDAndSummary(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Бюджет")
	catID := getExpenseCategory(t, env)

	createBody, _ := json.Marshal(map[string]any{
		"name":             "Продукты",
		"scope":            "category",
		"category_id":      catID,
		"amount":           "30000.00",
		"alert_at_percent": 80,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/budgets?month=2026-01", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create budget status %d", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	txBody, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "10000.00",
		"category_id": catID, "description": "магазин", "transaction_date": "2026-01-10 11:00:00",
	})
	respTx, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	if err != nil {
		t.Fatal(err)
	}
	respTx.Body.Close()
	if respTx.StatusCode != http.StatusCreated {
		t.Fatalf("create tx status %d", respTx.StatusCode)
	}

	respSum, err := env.authedRequest(http.MethodGet, "/api/v1/budgets/summary?month=2026-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer respSum.Body.Close()
	if respSum.StatusCode != http.StatusOK {
		t.Fatalf("summary status %d", respSum.StatusCode)
	}
	var summary struct {
		Items []struct {
			ID      string `json:"id"`
			Planned int64  `json:"planned"`
			Spent   int64  `json:"spent"`
			Percent int64  `json:"percent"`
			Status  string `json:"status"`
		} `json:"items"`
	}
	_ = json.NewDecoder(respSum.Body).Decode(&summary)
	if len(summary.Items) != 1 {
		t.Fatalf("expected 1 budget, got %d", len(summary.Items))
	}
	item := summary.Items[0]
	if item.Planned != 3000000 || item.Spent != 1000000 {
		t.Fatalf("planned/spent mismatch: %+v", item)
	}
	if item.Percent != 33 {
		t.Fatalf("expected percent 33, got %d", item.Percent)
	}
	if item.Status != "ok" {
		t.Fatalf("expected status ok, got %s", item.Status)
	}

	patchBody, _ := json.Marshal(map[string]any{
		"name": "Продукты", "scope": "category", "category_id": catID,
		"amount": "30000.00", "alert_at_percent": 80, "is_active": false,
	})
	respPatch, err := env.authedRequest(http.MethodPatch, "/api/v1/budgets/"+created.ID, bytes.NewReader(patchBody))
	if err != nil {
		t.Fatal(err)
	}
	respPatch.Body.Close()
	if respPatch.StatusCode != http.StatusOK {
		t.Fatalf("patch status %d", respPatch.StatusCode)
	}

	respDel, err := env.authedRequest(http.MethodDelete, "/api/v1/budgets/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	respDel.Body.Close()
	if respDel.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status %d", respDel.StatusCode)
	}
}

func TestBudgetDuplicateActive(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	catID := getExpenseCategory(t, env)

	body, _ := json.Marshal(map[string]any{
		"name": "A", "scope": "category", "category_id": catID, "amount": "1000.00",
	})
	resp1, _ := env.authedRequest(http.MethodPost, "/api/v1/budgets?month=2026-01", bytes.NewReader(body))
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first create %d", resp1.StatusCode)
	}

	body2, _ := json.Marshal(map[string]any{
		"name": "B", "scope": "category", "category_id": catID, "amount": "2000.00",
	})
	resp2, _ := env.authedRequest(http.MethodPost, "/api/v1/budgets?month=2026-01", bytes.NewReader(body2))
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected conflict, got %d", resp2.StatusCode)
	}
}

func TestBudgetSummaryExcludesTransferCommission(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	fromID := createTestAccount(t, env, "Откуда")
	toID := createTestAccount(t, env, "Куда")
	catID := getExpenseCategory(t, env)

	createBody, _ := json.Marshal(map[string]any{
		"name": "Все расходы", "scope": "all_expense", "amount": "100000.00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/budgets?month=2026-01", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create budget status %d", resp.StatusCode)
	}

	txBody, _ := json.Marshal(map[string]any{
		"account_id": fromID, "type": "expense", "amount": "10000.00",
		"category_id": catID, "description": "магазин", "transaction_date": "2026-01-10 11:00:00",
	})
	respTx, _ := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	respTx.Body.Close()
	if respTx.StatusCode != http.StatusCreated {
		t.Fatalf("create tx status %d", respTx.StatusCode)
	}

	transferBody, _ := json.Marshal(map[string]any{
		"from_account_id": fromID, "to_account_id": toID,
		"amount": "5000.00", "commission": "100.00", "transaction_date": "2026-01-11 12:00:00",
	})
	respTr, _ := env.authedRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(transferBody))
	respTr.Body.Close()
	if respTr.StatusCode != http.StatusCreated {
		t.Fatalf("transfer status %d", respTr.StatusCode)
	}

	respSum, err := env.authedRequest(http.MethodGet, "/api/v1/budgets/summary?month=2026-01", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer respSum.Body.Close()
	var summary struct {
		Items []struct {
			Scope string `json:"scope"`
			Spent int64  `json:"spent"`
		} `json:"items"`
	}
	_ = json.NewDecoder(respSum.Body).Decode(&summary)
	var spent int64
	for _, item := range summary.Items {
		if item.Scope == "all_expense" {
			spent = item.Spent
			break
		}
	}
	if spent != 1000000 {
		t.Fatalf("expected spent 1000000 (expense only, no transfer commission), got %d", spent)
	}
}
