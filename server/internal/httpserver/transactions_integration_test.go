package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func createTestAccount(t *testing.T, env *testEnv, name string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"name": name, "type": "cash", "initial_balance": "1000.00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var acc struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&acc)
	return acc.ID
}

func getExpenseCategory(t *testing.T, env *testEnv) string {
	t.Helper()
	resp, err := env.authedRequest(http.MethodGet, "/api/v1/categories?type=expense", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cats []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cats)
	if len(cats) == 0 {
		t.Fatal("no expense categories")
	}
	return cats[0].ID
}

func TestCreateExpenseDecreasesBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")
	catID := getExpenseCategory(t, env)
	past := "2020-01-15 12:00:00"

	body, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "100.00",
		"category_id": catID, "transaction_date": past,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create tx status %d", resp.StatusCode)
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
	if bal.Balance != 90000 {
		t.Fatalf("expected 90000 kopecks, got %d", bal.Balance)
	}
}

func TestFutureExpenseDoesNotAffectBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "План")
	catID := getExpenseCategory(t, env)
	future := "2099-12-31 12:00:00"

	body, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "500.00",
		"category_id": catID, "transaction_date": future,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tx struct {
		Kind string `json:"kind"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&tx)
	if tx.Kind != "future" {
		t.Fatalf("expected future kind, got %s", tx.Kind)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("future should not change balance, got %d", bal.Balance)
	}
}

func TestTransferUpdatesBothAccounts(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	fromID := createTestAccount(t, env, "Откуда")
	toID := createTestAccount(t, env, "Куда")

	body, _ := json.Marshal(map[string]any{
		"from_account_id": fromID, "to_account_id": toID,
		"amount": "200.00", "transaction_date": "2020-06-01 10:00:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("transfer status %d", resp.StatusCode)
	}

	checkBal := func(id string, want int64) {
		t.Helper()
		r, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+id+"/balance", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var b struct {
			Balance int64 `json:"balance"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Balance != want {
			t.Fatalf("account %s: want %d, got %d", id, want, b.Balance)
		}
	}
	checkBal(fromID, 80000)
	checkBal(toID, 120000)
}

func TestTransferRollbackOnError(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	fromID := createTestAccount(t, env, "Откат от")
	toID := createTestAccount(t, env, "Откат к")

	// Fail the second leg of a transfer pair (in-leg) while keeping validation intact.
	_, err := env.db.Exec(`
		CREATE TRIGGER test_fail_second_transfer_leg
		BEFORE INSERT ON transactions
		FOR EACH ROW
		WHEN NEW.transfer_group_id IS NOT NULL
		  AND (SELECT COUNT(*) FROM transactions t WHERE t.transfer_group_id = NEW.transfer_group_id) >= 1
		BEGIN
			SELECT RAISE(ABORT, 'test inject: second transfer leg');
		END`)
	if err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = env.db.Exec(`DROP TRIGGER IF EXISTS test_fail_second_transfer_leg`)
	})

	const wantBal int64 = 100000
	assertBalance := func(id string) {
		t.Helper()
		r, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+id+"/balance", nil)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var b struct {
			Balance int64 `json:"balance"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Balance != wantBal {
			t.Fatalf("account %s: want balance %d, got %d", id, wantBal, b.Balance)
		}
	}
	assertBalance(fromID)
	assertBalance(toID)

	body, _ := json.Marshal(map[string]any{
		"from_account_id": fromID, "to_account_id": toID,
		"amount": "150.00", "transaction_date": "2020-06-01 10:00:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		t.Fatalf("expected transfer to fail, got 201")
	}

	var n int
	err = env.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM transactions WHERE type = 'transfer'`).Scan(&n)
	if err != nil {
		t.Fatalf("count transfers: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 transfer rows after rollback, got %d", n)
	}

	assertBalance(fromID)
	assertBalance(toID)
}

func TestInlineSubcategoryCreation(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")
	catID := getExpenseCategory(t, env)

	body, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "50.00",
		"category_id": catID, "subcategory_name": "Новая подкатегория",
		"transaction_date": "2020-01-01 10:00:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var tx struct {
		SubcategoryID   *string `json:"subcategory_id"`
		SubcategoryName *string `json:"subcategory_name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&tx)
	if tx.SubcategoryID == nil || *tx.SubcategoryID == "" {
		t.Fatal("expected subcategory_id")
	}
}

func TestTransactionListPagination(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Много")
	catID := getExpenseCategory(t, env)

	for i := range 3 {
		body, _ := json.Marshal(map[string]any{
			"account_id": accID, "type": "expense", "amount": "10.00",
			"category_id": catID, "transaction_date": "2020-01-0" + string(rune('1'+i)) + " 10:00:00",
		})
		resp, _ := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
		resp.Body.Close()
	}

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?limit=2&page=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var result struct {
		Data []any `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
			Limit int   `json:"limit"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Data))
	}
	if result.Meta.Total < 3 {
		t.Fatalf("expected total >= 3, got %d", result.Meta.Total)
	}
}

func TestMoneyRejectsThreeDecimals(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Точность")
	catID := getExpenseCategory(t, env)

	body, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "10.999",
		"category_id": catID, "transaction_date": "2020-01-01 10:00:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
