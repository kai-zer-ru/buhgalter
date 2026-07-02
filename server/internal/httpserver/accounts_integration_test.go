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
		var acc struct {
			ID string `json:"id"`
		}
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

	targetID := createTestAccount(t, env, "Приёмник")

	archResp, err := env.authedRequest(
		http.MethodPost,
		"/api/v1/accounts/"+acc.ID+"/archive?transfer_to_account_id="+targetID,
		nil,
	)
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
	var active []struct {
		ID string `json:"id"`
	}
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
	var archived []struct {
		ID string `json:"id"`
	}
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
	var cat struct {
		ID string `json:"id"`
	}
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
	var cats []struct {
		Name string `json:"name"`
	}
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

func TestCreateCreditCardAccount(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]string{
		"name": "Тинькофф Black", "type": "cash", "initial_balance": "0",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create debit status %d", resp.StatusCode)
	}

	bodyCC, _ := json.Marshal(map[string]string{
		"name":            "Кредитка",
		"type":            "credit_card",
		"bank_id":         "tinkoff",
		"credit_limit":    "65000.00",
		"initial_balance": "0",
	})
	respCC, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(bodyCC))
	if err != nil {
		t.Fatal(err)
	}
	defer respCC.Body.Close()
	if respCC.StatusCode != http.StatusCreated {
		t.Fatalf("create credit card status %d", respCC.StatusCode)
	}
	var acc struct {
		Type               string `json:"type"`
		CreditLimit        int64  `json:"credit_limit"`
		CreditLimitDisplay string `json:"credit_limit_display"`
	}
	_ = json.NewDecoder(respCC.Body).Decode(&acc)
	if acc.Type != "credit_card" || acc.CreditLimit != 6500000 {
		t.Fatalf("unexpected credit card: %+v", acc)
	}
	if acc.CreditLimitDisplay != "65000.00" {
		t.Fatalf("credit limit display %q", acc.CreditLimitDisplay)
	}
}

func TestArchiveCreditCardRequiresFullBalance(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")

	bodyCC, _ := json.Marshal(map[string]string{
		"name":            "Кредитка архив",
		"type":            "credit_card",
		"bank_id":         "tinkoff",
		"credit_limit":    "1000.00",
		"initial_balance": "0",
	})
	respCC, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(bodyCC))
	if err != nil {
		t.Fatal(err)
	}
	defer respCC.Body.Close()
	if respCC.StatusCode != http.StatusCreated {
		t.Fatalf("create credit card status %d", respCC.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(respCC.Body).Decode(&created)

	archResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+created.ID+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer archResp.Body.Close()
	if archResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("archive unpaid card status %d, want 400", archResp.StatusCode)
	}

	bodyPaid, _ := json.Marshal(map[string]string{
		"name":            "Кредитка оплачена",
		"type":            "credit_card",
		"bank_id":         "tinkoff",
		"credit_limit":    "1000.00",
		"initial_balance": "1000.00",
	})
	respPaid, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(bodyPaid))
	if err != nil {
		t.Fatal(err)
	}
	defer respPaid.Body.Close()
	if respPaid.StatusCode != http.StatusCreated {
		t.Fatalf("create paid credit card status %d", respPaid.StatusCode)
	}
	var paid struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(respPaid.Body).Decode(&paid)

	archResp2, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+paid.ID+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer archResp2.Body.Close()
	if archResp2.StatusCode != http.StatusOK {
		t.Fatalf("archive paid card status %d", archResp2.StatusCode)
	}
}

func TestDeleteCreditCardRequiresFullBalance(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")

	bodyCC, _ := json.Marshal(map[string]string{
		"name":            "Кредитка удаление",
		"type":            "credit_card",
		"bank_id":         "tinkoff",
		"credit_limit":    "1000.00",
		"initial_balance": "0",
	})
	respCC, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(bodyCC))
	if err != nil {
		t.Fatal(err)
	}
	defer respCC.Body.Close()
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(respCC.Body).Decode(&created)

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/accounts/"+created.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete unpaid card status %d, want 400", delResp.StatusCode)
	}

	bodyPaid, _ := json.Marshal(map[string]string{
		"name":            "Кредитка удалена",
		"type":            "credit_card",
		"bank_id":         "tinkoff",
		"credit_limit":    "1000.00",
		"initial_balance": "1000.00",
	})
	respPaid, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(bodyPaid))
	if err != nil {
		t.Fatal(err)
	}
	defer respPaid.Body.Close()
	var paid struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(respPaid.Body).Decode(&paid)

	delResp2, err := env.authedRequest(http.MethodDelete, "/api/v1/accounts/"+paid.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp2.Body.Close()
	if delResp2.StatusCode != http.StatusNoContent {
		t.Fatalf("delete paid card status %d, want 204", delResp2.StatusCode)
	}
}

func TestDeleteAccountSoftDelete(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	targetID := createTestAccount(t, env, "Целевой")
	accID := createTestAccount(t, env, "На удаление")
	catID := getExpenseCategory(t, env)

	txBody, _ := json.Marshal(map[string]any{
		"account_id": accID, "type": "expense", "amount": "50.00",
		"category_id": catID, "transaction_date": "2020-01-15 12:00:00",
	})
	txResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	if err != nil {
		t.Fatal(err)
	}
	txResp.Body.Close()
	if txResp.StatusCode != http.StatusCreated {
		t.Fatalf("create transaction status %d", txResp.StatusCode)
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/accounts/"+accID+"?transfer_to_account_id="+targetID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete account status %d", delResp.StatusCode)
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get deleted account status %d", getResp.StatusCode)
	}
	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(getResp.Body).Decode(&got)
	if got.ID != accID || got.Status != "deleted" {
		t.Fatalf("expected deleted account, got %+v", got)
	}

	activeResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer activeResp.Body.Close()
	var active []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(activeResp.Body).Decode(&active)
	for _, a := range active {
		if a.ID == accID {
			t.Fatal("deleted account should not appear in active list")
		}
	}

	deletedListResp, err := env.authedRequest(http.MethodGet, "/api/v1/accounts?status=deleted", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer deletedListResp.Body.Close()
	var deleted []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(deletedListResp.Body).Decode(&deleted)
	found := false
	for _, a := range deleted {
		if a.ID == accID {
			found = true
		}
	}
	if !found {
		t.Fatal("deleted account should appear in deleted list")
	}

	updBody, _ := json.Marshal(map[string]string{"name": "Новое имя"})
	updResp, err := env.authedRequest(http.MethodPut, "/api/v1/accounts/"+accID, bytes.NewReader(updBody))
	if err != nil {
		t.Fatal(err)
	}
	updResp.Body.Close()
	if updResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("update deleted account status %d, want 400", updResp.StatusCode)
	}

	listTxResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+accID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listTxResp.Body.Close()
	var txList struct {
		Data []struct {
			AccountStatus string `json:"account_status"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listTxResp.Body).Decode(&txList)
	if len(txList.Data) == 0 {
		t.Fatal("expected transaction on deleted account")
	}
	if txList.Data[0].AccountStatus != "deleted" {
		t.Fatalf("expected account_status deleted, got %q", txList.Data[0].AccountStatus)
	}

	transferResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+targetID+"&type=transfer", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transferResp.Body.Close()
	var transferList struct {
		Data []struct {
			Description string `json:"description"`
			Amount      int64  `json:"amount"`
		} `json:"data"`
	}
	_ = json.NewDecoder(transferResp.Body).Decode(&transferList)
	if len(transferList.Data) == 0 {
		t.Fatal("expected transfer to target account on delete")
	}
	if transferList.Data[0].Description != "Удаление счёта \"На удаление\"" {
		t.Fatalf("unexpected transfer description %q", transferList.Data[0].Description)
	}
}

func TestDeleteAccountRequiresTransferTarget(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	accID := createTestAccount(t, env, "С балансом")

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/accounts/"+accID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete without transfer target status %d, want 400", delResp.StatusCode)
	}
}

func TestArchiveAccountWithBalanceTransfer(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	targetID := createTestAccount(t, env, "Целевой")
	accID := createTestAccount(t, env, "На архив")

	archResp, err := env.authedRequest(
		http.MethodPost,
		"/api/v1/accounts/"+accID+"/archive?transfer_to_account_id="+targetID,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer archResp.Body.Close()
	if archResp.StatusCode != http.StatusOK {
		t.Fatalf("archive account status %d", archResp.StatusCode)
	}
	var archived struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(archResp.Body).Decode(&archived)
	if archived.ID != accID || archived.Status != "archived" {
		t.Fatalf("expected archived account, got %+v", archived)
	}

	transferResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+targetID+"&type=transfer", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transferResp.Body.Close()
	var transferList struct {
		Data []struct {
			Description string `json:"description"`
		} `json:"data"`
	}
	_ = json.NewDecoder(transferResp.Body).Decode(&transferList)
	if len(transferList.Data) == 0 {
		t.Fatal("expected transfer to target account on archive")
	}
	if transferList.Data[0].Description != "Архивация счёта \"На архив\"" {
		t.Fatalf("unexpected transfer description %q", transferList.Data[0].Description)
	}
}

func TestArchiveAccountRequiresTransferTarget(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	accID := createTestAccount(t, env, "С балансом")

	archResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+accID+"/archive", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer archResp.Body.Close()
	if archResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("archive without transfer target status %d, want 400", archResp.StatusCode)
	}
}

func TestArchiveAccountTransfersStoredBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	targetID := createTestAccount(t, env, "Целевой")

	body, _ := json.Marshal(map[string]string{
		"name": "Дрейф", "type": "cash", "initial_balance": "0",
	})
	createResp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	if created.ID == "" {
		t.Fatal("expected account id")
	}

	const driftBalance int64 = 75_000
	if _, err := env.db.Exec(`UPDATE accounts SET current_balance = ? WHERE id = ?`, driftBalance, created.ID); err != nil {
		t.Fatal(err)
	}

	archResp, err := env.authedRequest(
		http.MethodPost,
		"/api/v1/accounts/"+created.ID+"/archive?transfer_to_account_id="+targetID,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer archResp.Body.Close()
	if archResp.StatusCode != http.StatusOK {
		t.Fatalf("archive account status %d", archResp.StatusCode)
	}

	transferResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+targetID+"&type=transfer", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer transferResp.Body.Close()
	var transferList struct {
		Data []struct {
			Amount int64 `json:"amount"`
		} `json:"data"`
	}
	_ = json.NewDecoder(transferResp.Body).Decode(&transferList)
	if len(transferList.Data) == 0 {
		t.Fatal("expected transfer to target account on archive")
	}
	if transferList.Data[0].Amount != driftBalance {
		t.Fatalf("transfer amount %d, want %d", transferList.Data[0].Amount, driftBalance)
	}
}
