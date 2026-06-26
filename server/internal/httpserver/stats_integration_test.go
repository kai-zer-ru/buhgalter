package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func getCategoryByType(t *testing.T, env *testEnv, txType string) string {
	t.Helper()
	resp, err := env.authedRequest(http.MethodGet, "/api/v1/categories?type="+txType, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cats []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cats)
	if len(cats) == 0 {
		t.Fatalf("no categories for type %s", txType)
	}
	return cats[0].ID
}

func createTx(t *testing.T, env *testEnv, body map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create tx status %d", resp.StatusCode)
	}
}

func TestStatsSummaryAndCategoryTotals(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Статистика")
	expenseCat := getCategoryByType(t, env, "expense")
	incomeCat := getCategoryByType(t, env, "income")

	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "100.00",
		"category_id": expenseCat, "description": "такси", "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": accID, "type": "income", "amount": "50.00",
		"category_id": incomeCat, "description": "зарплата", "transaction_date": "2026-01-11 09:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "25.00",
		"category_id": expenseCat, "description": "будущее такси", "transaction_date": "2099-01-10 11:00:00",
	})

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/summary", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary status %d", resp.StatusCode)
	}
	var summary struct {
		IncomeTotal  int64 `json:"income_total"`
		ExpenseTotal int64 `json:"expense_total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&summary)
	if summary.IncomeTotal != 5000 || summary.ExpenseTotal != 10000 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	resp2, err := env.authedRequest(http.MethodGet, "/api/v1/stats/summary?include_future=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var summaryFuture struct {
		ExpenseTotal int64 `json:"expense_total"`
	}
	_ = json.NewDecoder(resp2.Body).Decode(&summaryFuture)
	if summaryFuture.ExpenseTotal != 12500 {
		t.Fatalf("expected future expense included, got %d", summaryFuture.ExpenseTotal)
	}

	resp3, err := env.authedRequest(http.MethodGet, "/api/v1/stats/by-category", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	var cat struct {
		Items []struct {
			Type  string `json:"type"`
			Total int64  `json:"total"`
		} `json:"items"`
	}
	_ = json.NewDecoder(resp3.Body).Decode(&cat)
	var incomeTotal, expenseTotal int64
	for _, item := range cat.Items {
		if item.Type == "income" {
			incomeTotal += item.Total
		}
		if item.Type == "expense" {
			expenseTotal += item.Total
		}
	}
	if incomeTotal != 5000 || expenseTotal != 10000 {
		t.Fatalf("category totals mismatch: income=%d expense=%d", incomeTotal, expenseTotal)
	}
}

func TestStatsSearchByDescription(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Поиск")
	catID := getCategoryByType(t, env, "expense")

	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "10.00",
		"category_id": catID, "description": "такси до дома", "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "10.00",
		"category_id": catID, "description": "продукты", "transaction_date": "2026-01-10 12:00:00",
	})

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/search?q=такси", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("search status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Description string `json:"description"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Data) != 1 || out.Data[0].Description != "такси до дома" {
		t.Fatalf("unexpected search result: %+v", out.Data)
	}
}

func TestStatsContextScopes(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Контекст")
	catID := getCategoryByType(t, env, "expense")
	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "100.00",
		"category_id": catID, "description": "контекст", "transaction_date": "2026-01-10 11:00:00",
	})

	// account scope
	accountResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?account_id="+accID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer accountResp.Body.Close()
	var accountCtx struct {
		Scope            string `json:"scope"`
		ScopeID          string `json:"scope_id"`
		TransactionCount int64  `json:"transaction_count"`
	}
	_ = json.NewDecoder(accountResp.Body).Decode(&accountCtx)
	if accountCtx.Scope != "account" || accountCtx.ScopeID != accID || accountCtx.TransactionCount == 0 {
		t.Fatalf("unexpected account context: %+v", accountCtx)
	}

	// debtor + debts scopes
	debt := createDebt(t, env, map[string]any{
		"debtor_name": "Илья", "direction": "lent", "amount": "200.00",
		"affects_balance": true, "account_id": accID, "debt_date": "2026-01-10 12:00:00",
	})
	debtorID := debt["debtor_id"].(string)

	debtorResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?debtor_id="+debtorID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer debtorResp.Body.Close()
	var debtorCtx struct {
		Scope            string `json:"scope"`
		TransactionCount int64  `json:"transaction_count"`
		LentTotal        int64  `json:"lent_total"`
	}
	_ = json.NewDecoder(debtorResp.Body).Decode(&debtorCtx)
	if debtorCtx.Scope != "debtor" || debtorCtx.TransactionCount == 0 || debtorCtx.LentTotal == 0 {
		t.Fatalf("unexpected debtor context: %+v", debtorCtx)
	}

	debtsResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?debts=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer debtsResp.Body.Close()
	var debtsCtx struct {
		Scope            string `json:"scope"`
		TransactionCount int64  `json:"transaction_count"`
	}
	_ = json.NewDecoder(debtsResp.Body).Decode(&debtsCtx)
	if debtsCtx.Scope != "debts" || debtsCtx.TransactionCount == 0 {
		t.Fatalf("unexpected debts context: %+v", debtsCtx)
	}

	// credit scope
	credit := createCredit(t, env, map[string]any{
		"name":             "Контекст-кредит",
		"principal_amount": "12000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      3,
		"interest_rate":    0,
		"debit_account_id": accID,
	})
	creditID := credit["id"].(string)
	creditResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?credit_id="+creditID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer creditResp.Body.Close()
	var creditCtx struct {
		Scope           string `json:"scope"`
		PaymentCount    int64  `json:"payment_count"`
		RemainingAmount int64  `json:"remaining_amount"`
	}
	_ = json.NewDecoder(creditResp.Body).Decode(&creditCtx)
	if creditCtx.Scope != "credit" || creditCtx.RemainingAmount <= 0 {
		t.Fatalf("unexpected credit context: %+v", creditCtx)
	}
	if creditCtx.PaymentCount != 0 {
		t.Fatalf("new credit should not have applied payments in context, got %+v", creditCtx)
	}
}

func TestStatsContextUsesFilters(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Фильтр-контекст")
	catID := getCategoryByType(t, env, "expense")

	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "10.00",
		"category_id": catID, "description": "январь", "transaction_date": "2026-01-15 10:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": accID, "type": "expense", "amount": "10.00",
		"category_id": catID, "description": "февраль", "transaction_date": "2026-02-15 10:00:00",
	})

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?account_id="+accID+"&from=2026-02-01%2000:00:00&to=2026-02-28%2023:59:59", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		TransactionCount int64 `json:"transaction_count"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.TransactionCount != 1 {
		t.Fatalf("expected 1 transaction in filtered context, got %d", out.TransactionCount)
	}
}

func TestStatsSummaryIncludesCreditPaymentTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Кредит-стат")

	credit := createCredit(t, env, map[string]any{
		"principal_amount":     "12000.00",
		"issue_date":           time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00"),
		"term_months":          6,
		"interest_rate":        0,
		"debit_account_id":     accID,
		"create_transactions":  true,
		"added_retroactively":  false,
	})
	schedule, ok := credit["schedule"].([]any)
	if !ok || len(schedule) == 0 {
		t.Fatal("expected schedule")
	}
	var firstScheduledDate string
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" {
			firstScheduledDate = row["payment_date"].(string)
			break
		}
	}
	if firstScheduledDate == "" {
		t.Fatal("expected scheduled payment date")
	}

	payBody, _ := json.Marshal(map[string]any{
		"amount":       "2000.00",
		"payment_date": firstScheduledDate,
	})
	payResp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+credit["id"].(string)+"/payments", bytes.NewReader(payBody))
	if err != nil {
		t.Fatal(err)
	}
	defer payResp.Body.Close()
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay status %d", payResp.StatusCode)
	}

	var paid map[string]any
	_ = json.NewDecoder(payResp.Body).Decode(&paid)
	var creditExpense int64
	var creditTxCount int64
	for _, item := range paid["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] != "scheduled" || row["is_applied"] != true || row["transaction_id"] == nil {
			continue
		}
		if amt, ok := row["amount"].(float64); ok {
			creditExpense += int64(amt)
			creditTxCount++
		}
	}
	if creditExpense == 0 || creditTxCount == 0 {
		t.Fatal("expected applied credit payment transaction after manual pay")
	}

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/summary", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var summary struct {
		ExpenseTotal     int64 `json:"expense_total"`
		TransactionCount int64 `json:"transaction_count"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&summary)
	if summary.ExpenseTotal < creditExpense {
		t.Fatalf("expected expense >= %d, got %d", creditExpense, summary.ExpenseTotal)
	}
	if summary.TransactionCount < creditTxCount {
		t.Fatalf("expected at least %d credit payments in stats count, got %d", creditTxCount, summary.TransactionCount)
	}
}

func TestStatsContextAccountIncludesTransfers(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	fromID := createTestAccount(t, env, "Яндекс")
	toID := createTestAccount(t, env, "Наличные")
	catID := getCategoryByType(t, env, "expense")

	transferBody, _ := json.Marshal(map[string]any{
		"from_account_id": fromID, "to_account_id": toID,
		"amount": "2000.00", "transaction_date": "2020-06-01 10:00:00",
	})
	transferResp, err := env.authedRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(transferBody))
	if err != nil {
		t.Fatal(err)
	}
	transferResp.Body.Close()
	if transferResp.StatusCode != http.StatusCreated {
		t.Fatalf("transfer status %d", transferResp.StatusCode)
	}

	expenseBody, _ := json.Marshal(map[string]any{
		"account_id": toID, "type": "expense", "amount": "100.00",
		"category_id": catID, "description": "тест", "transaction_date": "2020-06-02 10:00:00",
	})
	expenseResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(expenseBody))
	if err != nil {
		t.Fatal(err)
	}
	expenseResp.Body.Close()
	if expenseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expense status %d", expenseResp.StatusCode)
	}

	ctxResp, err := env.authedRequest(http.MethodGet, "/api/v1/stats/context?account_id="+toID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ctxResp.Body.Close()
	var ctx struct {
		IncomeTotal      int64 `json:"income_total"`
		ExpenseTotal     int64 `json:"expense_total"`
		TransactionCount int64 `json:"transaction_count"`
	}
	_ = json.NewDecoder(ctxResp.Body).Decode(&ctx)
	if ctx.IncomeTotal != 200000 {
		t.Fatalf("expected income 200000 (transfer in), got %d", ctx.IncomeTotal)
	}
	if ctx.ExpenseTotal != 10000 {
		t.Fatalf("expected expense 10000, got %d", ctx.ExpenseTotal)
	}
	if ctx.TransactionCount != 2 {
		t.Fatalf("expected 2 operations on account, got %d", ctx.TransactionCount)
	}
}
