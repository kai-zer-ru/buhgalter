package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func createDebt(t *testing.T, env *testEnv, body map[string]any) map[string]any {
	t.Helper()
	if _, ok := body["debt_date"]; !ok {
		body["debt_date"] = "2020-06-15 10:30:00"
	}
	if _, ok := body["due_date"]; !ok {
		body["due_date"] = "2025-12-31 18:00:00"
	}
	raw, _ := json.Marshal(body)
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/debts", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create debt status %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func TestDebtWithoutBalanceNoTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Денис",
		"direction":       "lent",
		"amount":          "500.00",
		"due_date":        "2025-12-31 00:00:00",
		"affects_balance": false,
	})
	if debt["transaction_id"] != nil {
		t.Fatal("expected no transaction_id")
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("balance should be unchanged, got %d", bal.Balance)
	}
}

func TestDebtWithBalanceCreatesTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Основной")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Мария",
		"direction":       "lent",
		"amount":          "200.00",
		"due_date":        "2025-12-31 00:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	if debt["transaction_id"] == nil {
		t.Fatal("expected transaction_id")
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 80000 {
		t.Fatalf("expected 80000 kopecks after lent 200, got %d", bal.Balance)
	}
}

func TestBorrowedDebtIncreasesBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Пётр",
		"direction":       "borrowed",
		"amount":          "150.00",
		"debt_date":       "2020-06-15 10:30:00",
		"due_date":        "2025-12-31 18:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	if debt["transaction_id"] == nil {
		t.Fatal("expected transaction_id")
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 115000 {
		t.Fatalf("expected 115000 kopecks after borrowed 150, got %d", bal.Balance)
	}
}

func TestDebtWithCurrentDateAffectsBalance(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debtDate := time.Now().UTC().Format("2006-01-02 15:04:05")
	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Света",
		"direction":       "borrowed",
		"amount":          "50.00",
		"debt_date":       debtDate,
		"due_date":        "2025-12-31 18:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	txID, ok := debt["transaction_id"].(string)
	if !ok || txID == "" {
		t.Fatal("expected transaction_id")
	}

	txResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer txResp.Body.Close()
	var tx struct {
		Kind string `json:"kind"`
		Type string `json:"type"`
	}
	_ = json.NewDecoder(txResp.Body).Decode(&tx)
	if tx.Kind != "manual" {
		t.Fatalf("debt tx kind %q, want manual", tx.Kind)
	}
	if tx.Type != "income" {
		t.Fatalf("borrowed tx type %q, want income", tx.Type)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 105000 {
		t.Fatalf("expected 105000 after borrowed 50 at now, got %d", bal.Balance)
	}
}

func TestDebtTransactionUsesDebtDate(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Основной")

	const debtDate = "2020-03-10 14:00:00"
	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Олег",
		"direction":       "lent",
		"amount":          "50.00",
		"debt_date":       debtDate,
		"due_date":        "2025-12-31 18:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	if debt["debt_date"] != debtDate {
		t.Fatalf("expected debt_date %q, got %v", debtDate, debt["debt_date"])
	}
	txID, ok := debt["transaction_id"].(string)
	if !ok || txID == "" {
		t.Fatal("expected transaction_id")
	}

	txResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusOK {
		t.Fatalf("get transaction status %d", txResp.StatusCode)
	}
	var tx struct {
		TransactionDate string `json:"transaction_date"`
	}
	_ = json.NewDecoder(txResp.Body).Decode(&tx)
	if tx.TransactionDate != debtDate {
		t.Fatalf("transaction_date %q, want %q", tx.TransactionDate, debtDate)
	}
}

func TestDeleteDebtLinkedTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Катя",
		"direction":       "lent",
		"amount":          "100.00",
		"affects_balance": true,
		"account_id":      accID,
	})
	txID := debt["transaction_id"].(string)
	debtID := debt["id"].(string)

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/transactions/"+txID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete debt-linked tx status %d", delResp.StatusCode)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("balance should be restored to 100000, got %d", bal.Balance)
	}

	debtResp, _ := env.authedRequest(http.MethodGet, "/api/v1/debts/"+debtID, nil)
	defer debtResp.Body.Close()
	if debtResp.StatusCode != http.StatusNotFound {
		t.Fatalf("debt should be deleted with opening transaction, status %d", debtResp.StatusCode)
	}
}

func TestPartialSettleReducesDebt(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Олег",
		"direction":       "lent",
		"amount":          "100.00",
		"affects_balance": true,
		"account_id":      accID,
	})
	debtID := debt["id"].(string)

	body, _ := json.Marshal(map[string]any{
		"amount":          "40.00",
		"settled_at":      "2020-01-15 12:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/settle", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("partial settle status %d", resp.StatusCode)
	}
	var updated struct {
		Amount    int64 `json:"amount"`
		IsSettled bool  `json:"is_settled"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&updated)
	if updated.IsSettled || updated.Amount != 6000 {
		t.Fatalf("expected active debt 6000 kopecks, got settled=%v amount=%d", updated.IsSettled, updated.Amount)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	// 1000 - 100 lent + 40 return = 940
	if bal.Balance != 94000 {
		t.Fatalf("expected balance 94000 after partial return, got %d", bal.Balance)
	}
}

func TestDeleteDebtRemovesLinkedTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Иван",
		"direction":       "lent",
		"amount":          "100.00",
		"affects_balance": true,
		"account_id":      accID,
	})
	debtID := debt["id"].(string)
	txID := debt["transaction_id"].(string)

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/debts/"+debtID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete debt status %d", delResp.StatusCode)
	}

	txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusNotFound {
		t.Fatalf("creation tx should be deleted, status %d", txResp.StatusCode)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("balance should be restored to 100000, got %d", bal.Balance)
	}
}

func TestDeleteSettledDebtRemovesAllTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Иван",
		"direction":       "lent",
		"amount":          "100.00",
		"affects_balance": true,
		"account_id":      accID,
	})
	debtID := debt["id"].(string)
	openTxID := debt["transaction_id"].(string)

	body, _ := json.Marshal(map[string]any{
		"settled_at":      "2020-01-15 12:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	settleResp, err := env.authedRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/settle", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer settleResp.Body.Close()
	if settleResp.StatusCode != http.StatusOK {
		t.Fatalf("settle status %d", settleResp.StatusCode)
	}

	listResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+accID, nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID          string `json:"id"`
			Description string `json:"description"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&list)
	var settleTxID string
	for _, tx := range list.Data {
		if tx.ID != openTxID {
			settleTxID = tx.ID
			break
		}
	}
	if settleTxID == "" {
		t.Fatal("expected settle transaction")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/debts/"+debtID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete settled debt status %d", delResp.StatusCode)
	}

	for _, id := range []string{openTxID, settleTxID} {
		txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+id, nil)
		defer txResp.Body.Close()
		if txResp.StatusCode != http.StatusNotFound {
			t.Fatalf("tx %s should be deleted, status %d", id, txResp.StatusCode)
		}
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("balance should be restored to 100000 after debt delete, got %d", bal.Balance)
	}
}

func TestCannotDeleteDebtTransactionAfterPartialSettle(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Олег",
		"direction":       "lent",
		"amount":          "100.00",
		"affects_balance": true,
		"account_id":      accID,
	})
	debtID := debt["id"].(string)
	openTxID := debt["transaction_id"].(string)

	body, _ := json.Marshal(map[string]any{
		"amount":          "40.00",
		"settled_at":      "2020-01-15 12:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	settleResp, err := env.authedRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/settle", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer settleResp.Body.Close()
	if settleResp.StatusCode != http.StatusOK {
		t.Fatalf("partial settle status %d", settleResp.StatusCode)
	}

	for _, txID := range []string{openTxID} {
		delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/transactions/"+txID, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer delResp.Body.Close()
		if delResp.StatusCode != http.StatusConflict {
			t.Fatalf("delete open tx after partial settle status %d, want 409", delResp.StatusCode)
		}
	}

	listResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+accID, nil)
	defer listResp.Body.Close()
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&list)
	var settleTxID string
	for _, tx := range list.Data {
		if tx.ID != openTxID && tx.Type == "income" {
			settleTxID = tx.ID
			break
		}
	}
	if settleTxID == "" {
		t.Fatal("expected settle transaction")
	}

	delSettleResp, err := env.authedRequest(http.MethodDelete, "/api/v1/transactions/"+settleTxID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delSettleResp.Body.Close()
	if delSettleResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete settle tx after partial settle status %d, want 204", delSettleResp.StatusCode)
	}

	debtResp, _ := env.authedRequest(http.MethodGet, "/api/v1/debts/"+debtID, nil)
	defer debtResp.Body.Close()
	var gotDebt struct {
		Amount    int64 `json:"amount"`
		IsSettled bool  `json:"is_settled"`
	}
	_ = json.NewDecoder(debtResp.Body).Decode(&gotDebt)
	if gotDebt.IsSettled || gotDebt.Amount != 10000 {
		t.Fatalf("debt should be active with amount 10000 after settle tx delete, got settled=%v amount=%d", gotDebt.IsSettled, gotDebt.Amount)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 90000 {
		t.Fatalf("balance should be 90000 after settle tx delete, got %d", bal.Balance)
	}
}

func TestSettleCreatesReverseTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	debt := createDebt(t, env, map[string]any{
		"debtor_name":     "Иван",
		"direction":       "lent",
		"amount":          "100.00",
		"due_date":        "2025-12-31 00:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	debtID := debt["id"].(string)

	body, _ := json.Marshal(map[string]any{
		"settled_at":      "2020-01-15 12:00:00",
		"affects_balance": true,
		"account_id":      accID,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/settle", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("settle status %d", resp.StatusCode)
	}

	balResp, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	defer balResp.Body.Close()
	var bal struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balResp.Body).Decode(&bal)
	if bal.Balance != 100000 {
		t.Fatalf("expected balance restored to 100000, got %d", bal.Balance)
	}
}

func TestDebtorUniquePerUser(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	create := func(name string) string {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"name": name})
		resp, err := env.authedRequest(http.MethodPost, "/api/v1/debtors", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create debtor %q status %d", name, resp.StatusCode)
		}
		var d struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&d)
		return d.ID
	}

	id1 := create("Peter")
	id2 := create("peter")
	if id1 != id2 {
		t.Fatal("debtor names should be unique case-insensitively")
	}

	idB := create("Sergey")
	renameBody, _ := json.Marshal(map[string]string{"name": "Peter"})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/debtors/"+idB, bytes.NewReader(renameBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("rename to taken name should conflict, got %d", resp.StatusCode)
	}
}

func TestDebtsAPIWithBearerToken(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	createBody, _ := json.Marshal(map[string]string{"name": "api-token"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/user/tokens", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var created struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	req, err := http.NewRequest(http.MethodGet, env.server.URL+"/api/v1/debts/summary", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+created.Token)
	sumResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer sumResp.Body.Close()
	if sumResp.StatusCode != http.StatusOK {
		t.Fatalf("bearer summary status %d", sumResp.StatusCode)
	}
}

func TestDebtAccountFromOpenLinkWhenTransactionIDNull(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name": "Анна", "direction": "lent", "amount": "300.00",
		"due_date": "2025-12-31 00:00:00", "affects_balance": true, "account_id": accID,
	})
	debtorID := debt["debtor_id"].(string)
	debtID := debt["id"].(string)

	if _, err := env.db.Exec(`UPDATE debts SET transaction_id = NULL WHERE id = ?`, debtID); err != nil {
		t.Fatal(err)
	}

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/debtors/"+debtorID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get debtor status %d", resp.StatusCode)
	}
	var detail struct {
		Debts []struct {
			AccountID *string `json:"account_id"`
		} `json:"debts"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	if len(detail.Debts) != 1 || detail.Debts[0].AccountID == nil || *detail.Debts[0].AccountID != accID {
		t.Fatalf("expected account_id %q from open debt_transactions link, got %+v", accID, detail.Debts)
	}
}

func TestGetDebtorDetail(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name": "Анна", "direction": "lent", "amount": "300.00",
		"due_date": "2025-12-31 00:00:00", "affects_balance": true, "account_id": accID,
	})
	debtorID := debt["debtor_id"].(string)

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/debtors/"+debtorID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get debtor status %d", resp.StatusCode)
	}
	var detail struct {
		Name     string `json:"name"`
		OwedToMe int64  `json:"owed_to_me"`
		Debts    []struct {
			AccountID   *string `json:"account_id"`
			AccountName *string `json:"account_name"`
		} `json:"debts"`
		Transactions []struct {
			AccountID string `json:"account_id"`
			Deletable bool   `json:"deletable"`
		} `json:"transactions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	if detail.Name != "Анна" || detail.OwedToMe != 30000 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if len(detail.Debts) != 1 || len(detail.Transactions) != 1 {
		t.Fatalf("expected 1 debt and 1 tx, got %d/%d", len(detail.Debts), len(detail.Transactions))
	}
	if detail.Debts[0].AccountID == nil || *detail.Debts[0].AccountID != accID {
		t.Fatalf("debt account_id: want %q, got %+v", accID, detail.Debts[0].AccountID)
	}
	if detail.Transactions[0].AccountID != accID {
		t.Fatalf("tx account_id: want %q, got %q", accID, detail.Transactions[0].AccountID)
	}
	if !detail.Transactions[0].Deletable {
		t.Fatalf("single opening tx should be deletable")
	}
}

func TestGetDebtorDetailOpeningTxNotDeletableAfterSettle(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	debt := createDebt(t, env, map[string]any{
		"debtor_name": "Анна", "direction": "lent", "amount": "300.00",
		"due_date": "2025-12-31 00:00:00", "affects_balance": true, "account_id": accID,
	})
	debtorID := debt["debtor_id"].(string)
	debtID := debt["id"].(string)

	body, _ := json.Marshal(map[string]any{
		"amount": "100.00", "settled_at": "2020-01-15 12:00:00",
		"affects_balance": true, "account_id": accID,
	})
	settleResp, err := env.authedRequest(http.MethodPost, "/api/v1/debts/"+debtID+"/settle", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer settleResp.Body.Close()
	if settleResp.StatusCode != http.StatusOK {
		t.Fatalf("settle status %d", settleResp.StatusCode)
	}

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/debtors/"+debtorID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var detail struct {
		Transactions []struct {
			Deletable bool `json:"deletable"`
		} `json:"transactions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	if len(detail.Transactions) != 2 {
		t.Fatalf("expected 2 txs, got %d", len(detail.Transactions))
	}
	var protected, deletable int
	for _, tx := range detail.Transactions {
		if tx.Deletable {
			deletable++
		} else {
			protected++
		}
	}
	if protected != 1 || deletable != 1 {
		t.Fatalf("want 1 protected open tx and 1 deletable settle tx, got protected=%d deletable=%d", protected, deletable)
	}
}

func TestCannotCreateOppositeDirectionDebt(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кошелёк")

	lent := createDebt(t, env, map[string]any{
		"debtor_name": "Креветка", "direction": "lent", "amount": "100.00",
		"due_date": "2025-12-31 00:00:00", "affects_balance": false,
	})
	debtorID := lent["debtor_id"].(string)

	tryBorrow, _ := json.Marshal(map[string]any{
		"debtor_id": debtorID, "direction": "borrowed", "amount": "50.00",
		"debt_date": "2020-06-15 10:30:00", "due_date": "2025-12-31 00:00:00",
		"affects_balance": false,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/debts", bytes.NewReader(tryBorrow))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("borrow after lent: expected 409, got %d", resp.StatusCode)
	}

	borrowed := createDebt(t, env, map[string]any{
		"debtor_name": "Крокодил", "direction": "borrowed", "amount": "200.00",
		"due_date": "2025-12-31 00:00:00", "affects_balance": false, "account_id": accID,
	})
	creditorID := borrowed["debtor_id"].(string)

	tryLend, _ := json.Marshal(map[string]any{
		"debtor_id": creditorID, "direction": "lent", "amount": "50.00",
		"debt_date": "2020-06-15 10:30:00", "due_date": "2025-12-31 00:00:00",
		"affects_balance": false,
	})
	resp2, err := env.authedRequest(http.MethodPost, "/api/v1/debts", bytes.NewReader(tryLend))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("lend after borrowed: expected 409, got %d", resp2.StatusCode)
	}
}

func TestSystemDebtCategoryCannotDelete(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/categories?type=expense", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cats []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IsSystem bool   `json:"is_system"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cats)
	var debtCatID string
	for _, c := range cats {
		if c.Name == "Долги" && c.IsSystem {
			debtCatID = c.ID
			break
		}
	}
	if debtCatID == "" {
		t.Fatal("system Долги category not found")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/categories/"+debtCatID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusForbidden {
		t.Fatalf("delete system category should be forbidden, got %d", delResp.StatusCode)
	}

	subBody, _ := json.Marshal(map[string]string{"name": "Тест"})
	subResp, err := env.authedRequest(http.MethodPost, "/api/v1/categories/"+debtCatID+"/subcategories", bytes.NewReader(subBody))
	if err != nil {
		t.Fatal(err)
	}
	defer subResp.Body.Close()
	if subResp.StatusCode != http.StatusForbidden {
		t.Fatalf("create subcategory on system category should be forbidden, got %d", subResp.StatusCode)
	}
}
