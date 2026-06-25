package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/bank"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
)

func seedBanks(t *testing.T, env *testEnv) {
	t.Helper()
	if err := bank.SeedIfEmpty(context.Background(), env.db); err != nil {
		t.Fatalf("seed banks: %v", err)
	}
}

func TestBanksEndpoint(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)

	resp, err := http.Get(env.server.URL + "/api/v1/banks")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("banks status %d", resp.StatusCode)
	}
	var banks []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&banks)
	if len(banks) < 10 {
		t.Fatalf("expected seeded banks, got %d", len(banks))
	}
}

func TestCreateCashAccount(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"name": "Наличные", "type": "cash", "initial_balance": "1500.00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create cash status %d", resp.StatusCode)
	}
	var acc struct {
		InitialBalance int64  `json:"initial_balance"`
		Balance        int64  `json:"balance"`
		BalanceDisplay string `json:"balance_display"`
		IsPrimary      bool   `json:"is_primary"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&acc)
	if acc.InitialBalance != 150000 || acc.Balance != 150000 {
		t.Fatalf("expected 150000 kopecks, got %d/%d", acc.InitialBalance, acc.Balance)
	}
	if acc.BalanceDisplay != "1500.00" {
		t.Fatalf("balance display %q", acc.BalanceDisplay)
	}
	if !acc.IsPrimary {
		t.Fatal("first account should be primary")
	}
}

func TestCreateBankAccountRequiresBankID(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"name": "Тинькофф", "type": "bank", "initial_balance": "0",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	body2, _ := json.Marshal(map[string]string{
		"name": "Тинькофф", "type": "bank", "bank_id": "tinkoff", "initial_balance": "0",
	})
	resp2, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body2))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("create bank status %d", resp2.StatusCode)
	}
}

func TestAccountPrimary(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	create := func(name string) string {
		t.Helper()
		body, _ := json.Marshal(map[string]string{
			"name": name, "type": "cash", "initial_balance": "0",
		})
		resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var acc struct{ ID string `json:"id"` }
		_ = json.NewDecoder(resp.Body).Decode(&acc)
		return acc.ID
	}

	first := create("Первый")
	second := create("Второй")

	primResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+second+"/primary", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer primResp.Body.Close()
	if primResp.StatusCode != http.StatusOK {
		t.Fatalf("set primary status %d", primResp.StatusCode)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var accounts []struct {
		ID        string `json:"id"`
		IsPrimary bool   `json:"is_primary"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&accounts)
	var primaryCount int
	for _, a := range accounts {
		if a.IsPrimary {
			primaryCount++
			if a.ID != second {
				t.Fatalf("expected second account primary, got %s", a.ID)
			}
		}
	}
	if primaryCount != 1 {
		t.Fatalf("expected exactly one primary, got %d", primaryCount)
	}

	archResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+second+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	archResp.Body.Close()

	listResp2, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp2.Body.Close()
	accounts = nil
	_ = json.NewDecoder(listResp2.Body).Decode(&accounts)
	primaryCount = 0
	for _, a := range accounts {
		if a.IsPrimary {
			primaryCount++
			if a.ID != first {
				t.Fatalf("after archive expected first promoted, got %s", a.ID)
			}
		}
	}
	if primaryCount != 1 {
		t.Fatalf("expected one primary after archive, got %d", primaryCount)
	}
}

func TestArchiveAccountHiddenFromActiveList(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"name": "Кошелёк", "type": "cash", "initial_balance": "100",
	})
	createResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var acc struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&acc)
	createResp.Body.Close()

	archResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+acc.ID+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	archResp.Body.Close()
	if archResp.StatusCode != http.StatusOK {
		t.Fatalf("archive status %d", archResp.StatusCode)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var active []struct{ ID string `json:"id"` }
	_ = json.NewDecoder(listResp.Body).Decode(&active)
	for _, a := range active {
		if a.ID == acc.ID {
			t.Fatal("archived account should not appear in active list")
		}
	}

	archListResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts?status=archived", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer archListResp.Body.Close()
	var archived []struct{ ID string `json:"id"` }
	_ = json.NewDecoder(archListResp.Body).Decode(&archived)
	found := false
	for _, a := range archived {
		if a.ID == acc.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("archived account should appear in archived list")
	}
}

func TestCategoriesIsolatedPerUser(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	catBody, _ := json.Marshal(map[string]string{
		"name": "Моя категория", "type": "expense", "icon": "default",
	})
	catResp, err := env.authedRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(catBody))
	if err != nil {
		t.Fatal(err)
	}
	var cat struct{ ID string `json:"id"` }
	_ = json.NewDecoder(catResp.Body).Decode(&cat)
	catResp.Body.Close()

	userBody, _ := json.Marshal(map[string]any{
		"login": "user2", "password": "password1", "password_confirm": "password1",
		"display_name": "User2", "is_admin": false,
	})
	userResp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(userBody))
	if err != nil {
		t.Fatal(err)
	}
	userResp.Body.Close()

	env.login(t, "user2", "password1")
	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/categories?type=expense", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var cats []struct{ Name string `json:"name"` }
	_ = json.NewDecoder(listResp.Body).Decode(&cats)
	for _, c := range cats {
		if c.Name == "Моя категория" {
			t.Fatal("other user should not see admin's custom category")
		}
	}
}

func TestDefaultCategoriesOnUserCreate(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]any{
		"login": "catuser", "password": "password1", "password_confirm": "password1",
		"display_name": "Cat", "is_admin": false,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	env.login(t, "catuser", "password1")
	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/categories", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var cats []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		IsSystem bool   `json:"is_system"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&cats)
	if len(cats) != categoryseed.DefaultCount {
		t.Fatalf("expected %d default categories, got %d", categoryseed.DefaultCount, len(cats))
	}
	var debtCount, creditCount, commissionCount int
	for _, c := range cats {
		if c.Name == "Долги" && c.IsSystem {
			debtCount++
		}
		if c.Name == "Кредиты" && c.IsSystem && c.Type == "expense" {
			creditCount++
		}
		if c.Name == categoryseed.CommissionCategoryName && c.IsSystem && c.Type == "expense" {
			commissionCount++
		}
	}
	if debtCount != 2 {
		t.Fatalf("expected 2 system Долги categories, got %d", debtCount)
	}
	if creditCount != 1 {
		t.Fatalf("expected 1 system Кредиты category, got %d", creditCount)
	}
	if commissionCount != 1 {
		t.Fatalf("expected 1 system %s category, got %d", categoryseed.CommissionCategoryName, commissionCount)
	}
}
