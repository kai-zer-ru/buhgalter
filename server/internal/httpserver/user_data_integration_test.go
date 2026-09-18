package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
)

func TestDeleteUserDataResetsLedger(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")
	catID := getExpenseCategory(t, env)

	txBody, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "15.00",
		"category_id": catID, "transaction_date": "2020-01-15 12:00:00",
	})
	txResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	if err != nil {
		t.Fatal(err)
	}
	txResp.Body.Close()
	if txResp.StatusCode != http.StatusCreated {
		t.Fatalf("create tx status %d", txResp.StatusCode)
	}

	createDebt(t, env, map[string]any{
		"debtor_name":     "Иван",
		"direction":       "lent",
		"amount":          "200.00",
		"affects_balance": true,
		"account_id":      accID,
	})

	customBody, _ := json.Marshal(map[string]any{
		"name": "Кастомная", "type": "expense", "icon": "default",
	})
	catResp, err := env.authedRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(customBody))
	if err != nil {
		t.Fatal(err)
	}
	catResp.Body.Close()
	if catResp.StatusCode != http.StatusCreated {
		t.Fatalf("create category status %d", catResp.StatusCode)
	}

	tokenBody, _ := json.Marshal(map[string]any{"name": "keep-me", "never_expires": true})
	tokenResp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(tokenBody))
	if err != nil {
		t.Fatal(err)
	}
	if tokenResp.StatusCode != http.StatusCreated {
		tokenResp.Body.Close()
		t.Fatalf("create token status %d", tokenResp.StatusCode)
	}
	tokenResp.Body.Close()

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/user/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete data status %d", delResp.StatusCode)
	}

	meResp, err := env.authedRequest(http.MethodGet, "/api/v1/auth/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer meResp.Body.Close()
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me after wipe status %d", meResp.StatusCode)
	}

	accResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer accResp.Body.Close()
	var accounts []any
	_ = json.NewDecoder(accResp.Body).Decode(&accounts)
	if len(accounts) != 0 {
		t.Fatalf("accounts left: %d", len(accounts))
	}

	debtResp, err := env.authedRequest(http.MethodGet, "/api/v1/debts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer debtResp.Body.Close()
	var debts []any
	_ = json.NewDecoder(debtResp.Body).Decode(&debts)
	if len(debts) != 0 {
		t.Fatalf("debts left: %d", len(debts))
	}

	catList, err := env.authedRequest(http.MethodGet, "/api/v1/categories", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer catList.Body.Close()
	var cats []struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(catList.Body).Decode(&cats)
	if len(cats) != categoryseed.DefaultCount {
		t.Fatalf("categories %d want %d", len(cats), categoryseed.DefaultCount)
	}
	for _, c := range cats {
		if c.Name == "Кастомная" {
			t.Fatal("custom category survived")
		}
	}

	tokList, err := env.authedRequest(http.MethodGet, "/api/v1/user/tokens", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tokList.Body.Close()
	var tokens []struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(tokList.Body).Decode(&tokens)
	if len(tokens) != 1 || tokens[0].Name != "keep-me" {
		t.Fatalf("tokens %+v", tokens)
	}
}

func TestDeleteUserDataUnauthorized(t *testing.T) {
	env := setupConfigured(t)
	resp, err := http.NewRequest(http.MethodDelete, env.server.URL+"/api/v1/user/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := http.DefaultClient.Do(resp)
	if err != nil {
		t.Fatal(err)
	}
	got.Body.Close()
	if got.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", got.StatusCode)
	}
}
