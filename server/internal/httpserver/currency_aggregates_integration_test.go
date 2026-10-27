package httpserver_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAccountsSummaryByCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "1000.00", "currency": "RUB"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "100.00", "currency": "USD"})
	_ = rub
	_ = usd

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/accounts/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("summary %d: %s", status, raw)
	}
	var body struct {
		TotalBalance     int64 `json:"total_balance"`
		TotalsByCurrency []struct {
			Currency     string `json:"currency"`
			TotalBalance int64  `json:"total_balance"`
		} `json:"totals_by_currency"`
		Accounts []struct {
			ID       string `json:"id"`
			Currency string `json:"currency"`
		} `json:"accounts"`
	}
	_ = json.Unmarshal(raw, &body)
	if body.TotalBalance != 100000 {
		t.Fatalf("total_balance %d want 100000", body.TotalBalance)
	}
	if len(body.TotalsByCurrency) != 2 || body.TotalsByCurrency[0].Currency != "RUB" || body.TotalsByCurrency[0].TotalBalance != 100000 {
		t.Fatalf("totals %+v", body.TotalsByCurrency)
	}
	if body.TotalsByCurrency[1].Currency != "USD" || body.TotalsByCurrency[1].TotalBalance != 10000 {
		t.Fatalf("usd total %+v", body.TotalsByCurrency)
	}
	foundUSD := false
	for _, a := range body.Accounts {
		if a.ID == usd.ID && a.Currency == "USD" {
			foundUSD = true
		}
	}
	if !foundUSD {
		t.Fatal("usd account currency missing")
	}
}

func TestCreditCardsByCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	currencyCreateAccount(t, env, map[string]any{
		"name": "RUB карта", "type": "credit_card", "bank_id": "tinkoff",
		"initial_balance": "50000.00", "credit_limit": "100000.00", "currency": "RUB",
	})
	currencyCreateAccount(t, env, map[string]any{
		"name": "USD карта", "type": "credit_card", "bank_id": "tinkoff",
		"initial_balance": "200.00", "credit_limit": "1000.00", "currency": "USD",
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/dashboard", nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard %d: %s", status, raw)
	}
	var body struct {
		CreditCardsSummary *struct {
			Count        int   `json:"count"`
			TotalBalance int64 `json:"total_balance"`
		} `json:"credit_cards_summary"`
		CreditCardsByCurrency []struct {
			Currency     string `json:"currency"`
			Count        int    `json:"count"`
			TotalBalance int64  `json:"total_balance"`
		} `json:"credit_cards_by_currency"`
	}
	_ = json.Unmarshal(raw, &body)
	if body.CreditCardsSummary == nil || body.CreditCardsSummary.Count != 1 || body.CreditCardsSummary.TotalBalance != 5000000 {
		t.Fatalf("profile cards %+v", body.CreditCardsSummary)
	}
	if len(body.CreditCardsByCurrency) != 2 {
		t.Fatalf("by currency %+v", body.CreditCardsByCurrency)
	}
}

func TestStatsSummaryCurrencyFilter(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})
	cat := getCategoryByType(t, env, "expense")
	createTx(t, env, map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "10.00",
		"category_id": cat, "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": rub.ID, "type": "expense", "amount": "50.00",
		"category_id": cat, "transaction_date": "2026-01-10 11:00:00",
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/stats/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("summary %d: %s", status, raw)
	}
	var def struct {
		ExpenseTotal        int64    `json:"expense_total"`
		Currency            string   `json:"currency"`
		AvailableCurrencies []string `json:"available_currencies"`
	}
	_ = json.Unmarshal(raw, &def)
	if def.ExpenseTotal != 5000 || def.Currency != "RUB" {
		t.Fatalf("default %+v", def)
	}
	if len(def.AvailableCurrencies) < 2 || def.AvailableCurrencies[0] != "RUB" {
		t.Fatalf("available %+v", def.AvailableCurrencies)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/summary?currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("usd %d: %s", status, raw)
	}
	var usdSum struct {
		ExpenseTotal int64  `json:"expense_total"`
		Currency     string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &usdSum)
	if usdSum.ExpenseTotal != 1000 || usdSum.Currency != "USD" {
		t.Fatalf("usd summary %+v", usdSum)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/summary?currency=XYZ", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("xyz status %d: %s", status, raw)
	}
	if !strings.Contains(string(raw), "ERR_CURRENCY") && !strings.Contains(string(raw), "Неподдерживаемая валюта") {
		t.Fatalf("xyz body %s", raw)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?account_id="+usd.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("context %d: %s", status, raw)
	}
	var ctxSum struct {
		ExpenseTotal int64  `json:"expense_total"`
		Currency     string `json:"currency"`
		Scope        string `json:"scope"`
	}
	_ = json.Unmarshal(raw, &ctxSum)
	if ctxSum.ExpenseTotal != 1000 || ctxSum.Currency != "USD" || ctxSum.Scope != "account" {
		t.Fatalf("account context %+v", ctxSum)
	}
}

func TestNetWorthCurrencyFilter(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "100.00", "currency": "USD"})
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})
	_ = usd
	createCredit(t, env, map[string]any{
		"name":             "Кредит",
		"principal_amount": "30000.00",
		"issue_date":       time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      12,
		"interest_rate":    0,
		"debit_account_id": rub.ID,
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/stats/net-worth?currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("usd nw %d: %s", status, raw)
	}
	var usdNW struct {
		Current struct {
			CashBank         int64 `json:"cash_bank"`
			CreditsRemaining int64 `json:"credits_remaining"`
		} `json:"current"`
		Currency string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &usdNW)
	if usdNW.Currency != "USD" || usdNW.Current.CashBank != 10000 || usdNW.Current.CreditsRemaining != 0 {
		t.Fatalf("usd net worth %+v", usdNW)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/net-worth", nil)
	if status != http.StatusOK {
		t.Fatalf("default nw %d: %s", status, raw)
	}
	var rubNW struct {
		Current struct {
			CreditsRemaining int64 `json:"credits_remaining"`
		} `json:"current"`
	}
	_ = json.Unmarshal(raw, &rubNW)
	if rubNW.Current.CreditsRemaining != 3000000 {
		t.Fatalf("default credits remaining %d", rubNW.Current.CreditsRemaining)
	}
}

func TestBudgetSpentByAccountCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})
	cat := getCategoryByType(t, env, "expense")
	createTx(t, env, map[string]any{
		"account_id": rub.ID, "type": "expense", "amount": "50.00",
		"category_id": cat, "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "10.00",
		"category_id": cat, "transaction_date": "2026-01-10 11:00:00",
	})
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Все", "scope": "all_expense", "amount": "1000.00", "currency": "RUB",
	})
	if status != http.StatusCreated {
		t.Fatalf("budget %d: %s", status, raw)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/budgets/summary?month=2026-01", nil)
	if status != http.StatusOK {
		t.Fatalf("summary %d: %s", status, raw)
	}
	var sum struct {
		Items []struct {
			Spent    int64  `json:"spent"`
			Currency string `json:"currency"`
		} `json:"items"`
		AvailableCurrencies []string `json:"available_currencies"`
	}
	_ = json.Unmarshal(raw, &sum)
	if len(sum.Items) != 1 || sum.Items[0].Spent != 5000 || sum.Items[0].Currency != "RUB" {
		t.Fatalf("spent %+v", sum.Items)
	}
	if len(sum.AvailableCurrencies) < 2 || sum.AvailableCurrencies[0] != "RUB" {
		t.Fatalf("available %+v", sum.AvailableCurrencies)
	}
}

func TestDebtsSummaryByCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "1000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "1000.00", "currency": "USD"})
	createDebt(t, env, map[string]any{
		"debtor_name": "Иван", "direction": "borrowed", "amount": "100.00",
		"affects_balance": true, "account_id": rub.ID,
	})
	createDebt(t, env, map[string]any{
		"debtor_name": "John", "direction": "lent", "amount": "20.00",
		"affects_balance": true, "account_id": usd.ID,
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/debts/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("debts %d: %s", status, raw)
	}
	var sum struct {
		IOwe       int64 `json:"i_owe"`
		OwedToMe   int64 `json:"owed_to_me"`
		ByCurrency []struct {
			Currency string `json:"currency"`
			IOwe     int64  `json:"i_owe"`
			OwedToMe int64  `json:"owed_to_me"`
		} `json:"by_currency"`
	}
	_ = json.Unmarshal(raw, &sum)
	if sum.IOwe != 10000 || sum.OwedToMe != 0 {
		t.Fatalf("profile fields %+v", sum)
	}
	if len(sum.ByCurrency) != 2 {
		t.Fatalf("by_currency %+v", sum.ByCurrency)
	}
}

func TestDashboardSuccessSignal(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "9000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})
	status, _, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": usd.ID, "amount": "9000.00", "to_amount": "100.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("exchange %d: %s", status, raw)
	}
	cat := getCategoryByType(t, env, "expense")
	createTx(t, env, map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "10.00",
		"category_id": cat, "transaction_date": "2026-01-10 11:00:00",
	})

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/dashboard", nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard %d: %s", status, raw)
	}
	var dash struct {
		TotalBalance     int64 `json:"total_balance"`
		TotalsByCurrency []struct {
			Currency     string `json:"currency"`
			TotalBalance int64  `json:"total_balance"`
		} `json:"totals_by_currency"`
	}
	_ = json.Unmarshal(raw, &dash)
	if dash.TotalBalance != 0 {
		t.Fatalf("rub total %d", dash.TotalBalance)
	}
	var usdBal int64
	for _, row := range dash.TotalsByCurrency {
		if row.Currency == "USD" {
			usdBal = row.TotalBalance
		}
	}
	if usdBal != 9000 {
		t.Fatalf("usd total %d want 9000 (90$)", usdBal)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("stats default %d: %s", status, raw)
	}
	var rubStats struct {
		ExpenseTotal int64 `json:"expense_total"`
	}
	_ = json.Unmarshal(raw, &rubStats)
	if rubStats.ExpenseTotal != 0 {
		t.Fatalf("default stats expense %d", rubStats.ExpenseTotal)
	}
	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/summary?currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("stats usd %d: %s", status, raw)
	}
	var usdStats struct {
		ExpenseTotal int64 `json:"expense_total"`
	}
	_ = json.Unmarshal(raw, &usdStats)
	if usdStats.ExpenseTotal != 1000 {
		t.Fatalf("usd stats expense %d", usdStats.ExpenseTotal)
	}
}

func TestDebtorDetailByCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "1000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "1000.00", "currency": "USD"})
	rubDebt := createDebt(t, env, map[string]any{
		"debtor_name": "Иван", "direction": "lent", "amount": "100.00",
		"affects_balance": true, "account_id": rub.ID,
	})
	debtorID := rubDebt["debtor_id"].(string)
	createDebt(t, env, map[string]any{
		"debtor_id": debtorID, "direction": "lent", "amount": "20.00",
		"affects_balance": true, "account_id": usd.ID,
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/debtors/"+debtorID, nil)
	if status != http.StatusOK {
		t.Fatalf("debtor %d: %s", status, raw)
	}
	var detail struct {
		IOwe       int64 `json:"i_owe"`
		OwedToMe   int64 `json:"owed_to_me"`
		ByCurrency []struct {
			Currency string `json:"currency"`
			IOwe     int64  `json:"i_owe"`
			OwedToMe int64  `json:"owed_to_me"`
		} `json:"by_currency"`
	}
	_ = json.Unmarshal(raw, &detail)
	if detail.IOwe != 0 || detail.OwedToMe != 10000 {
		t.Fatalf("profile fields %+v", detail)
	}
	if len(detail.ByCurrency) != 2 {
		t.Fatalf("by_currency %+v", detail.ByCurrency)
	}
}

func TestBudgetSummaryCurrencyFilter(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "RUB все", "scope": "all_expense", "amount": "1000.00", "currency": "RUB",
	})
	if status != http.StatusCreated {
		t.Fatalf("rub budget %d: %s", status, raw)
	}
	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "USD все", "scope": "all_expense", "amount": "100.00", "currency": "USD",
	})
	if status != http.StatusCreated {
		t.Fatalf("usd budget %d: %s", status, raw)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/budgets/summary?month=2026-01", nil)
	if status != http.StatusOK {
		t.Fatalf("summary %d: %s", status, raw)
	}
	var all struct {
		Items []struct {
			Currency string `json:"currency"`
		} `json:"items"`
	}
	_ = json.Unmarshal(raw, &all)
	if len(all.Items) != 2 {
		t.Fatalf("unfiltered items %+v", all.Items)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/budgets/summary?month=2026-01&currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("usd filter %d: %s", status, raw)
	}
	var usdOnly struct {
		Items []struct {
			Currency string `json:"currency"`
		} `json:"items"`
	}
	_ = json.Unmarshal(raw, &usdOnly)
	if len(usdOnly.Items) != 1 || usdOnly.Items[0].Currency != "USD" {
		t.Fatalf("usd items %+v", usdOnly.Items)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/budgets/summary?month=2026-01&currency=XYZ", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("xyz status %d: %s", status, raw)
	}
	if !strings.Contains(string(raw), "ERR_CURRENCY") && !strings.Contains(string(raw), "Неподдерживаемая валюта") {
		t.Fatalf("xyz body %s", raw)
	}
}

func TestStatsSearchCurrencyFilter(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})
	cat := getCategoryByType(t, env, "expense")
	createTx(t, env, map[string]any{
		"account_id": rub.ID, "type": "expense", "amount": "50.00",
		"category_id": cat, "description": "кофе", "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "10.00",
		"category_id": cat, "description": "кофе", "transaction_date": "2026-01-10 11:00:00",
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/stats/search?q=кофе", nil)
	if status != http.StatusOK {
		t.Fatalf("search %d: %s", status, raw)
	}
	var def struct {
		Data []struct {
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`
		} `json:"data"`
		Currency string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &def)
	if def.Currency != "RUB" || len(def.Data) != 1 || def.Data[0].Amount != 5000 {
		t.Fatalf("default search %+v", def)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/search?q=кофе&currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("usd search %d: %s", status, raw)
	}
	var usdHit struct {
		Data []struct {
			Amount int64 `json:"amount"`
		} `json:"data"`
		Currency string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &usdHit)
	if usdHit.Currency != "USD" || len(usdHit.Data) != 1 || usdHit.Data[0].Amount != 1000 {
		t.Fatalf("usd search %+v", usdHit)
	}
}

func TestStatsContextMixedCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "1000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "1000.00", "currency": "USD"})
	usdDebt := createDebt(t, env, map[string]any{
		"debtor_name": "John", "direction": "lent", "amount": "20.00",
		"affects_balance": true, "account_id": usd.ID, "debt_date": "2026-01-10 12:00:00",
	})
	debtorID := usdDebt["debtor_id"].(string)
	createDebt(t, env, map[string]any{
		"debtor_name": "Иван", "direction": "lent", "amount": "100.00",
		"affects_balance": true, "account_id": rub.ID, "debt_date": "2026-01-10 12:00:00",
	})
	credit := createCredit(t, env, map[string]any{
		"name":             "USD кредит",
		"principal_amount": "300.00",
		"issue_date":       time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      12,
		"interest_rate":    0,
		"debit_account_id": usd.ID,
	})
	creditID := credit["id"].(string)

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?debts=1", nil)
	if status != http.StatusOK {
		t.Fatalf("debts default %d: %s", status, raw)
	}
	var debtsRUB struct {
		ExpenseTotal     int64  `json:"expense_total"`
		TransactionCount int64  `json:"transaction_count"`
		Currency         string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &debtsRUB)
	if debtsRUB.Currency != "RUB" || debtsRUB.ExpenseTotal != 10000 || debtsRUB.TransactionCount != 1 {
		t.Fatalf("default debts context %+v", debtsRUB)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?debts=1&currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("debts usd %d: %s", status, raw)
	}
	var debtsUSD struct {
		ExpenseTotal     int64  `json:"expense_total"`
		TransactionCount int64  `json:"transaction_count"`
		Currency         string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &debtsUSD)
	if debtsUSD.Currency != "USD" || debtsUSD.ExpenseTotal != 2000 || debtsUSD.TransactionCount != 1 {
		t.Fatalf("usd debts context %+v", debtsUSD)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?debtor_id="+debtorID, nil)
	if status != http.StatusOK {
		t.Fatalf("debtor default %d: %s", status, raw)
	}
	var debtorRUB struct {
		LentTotal        int64  `json:"lent_total"`
		TransactionCount int64  `json:"transaction_count"`
		Currency         string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &debtorRUB)
	if debtorRUB.Currency != "RUB" || debtorRUB.LentTotal != 0 || debtorRUB.TransactionCount != 0 {
		t.Fatalf("default debtor context %+v", debtorRUB)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?debtor_id="+debtorID+"&currency=USD", nil)
	if status != http.StatusOK {
		t.Fatalf("debtor usd %d: %s", status, raw)
	}
	var debtorUSD struct {
		LentTotal        int64  `json:"lent_total"`
		TransactionCount int64  `json:"transaction_count"`
		Currency         string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &debtorUSD)
	if debtorUSD.Currency != "USD" || debtorUSD.LentTotal != 2000 || debtorUSD.TransactionCount != 1 {
		t.Fatalf("usd debtor context %+v", debtorUSD)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/stats/context?credit_id="+creditID, nil)
	if status != http.StatusOK {
		t.Fatalf("credit context %d: %s", status, raw)
	}
	var creditCtx struct {
		RemainingAmount int64  `json:"remaining_amount"`
		Currency        string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &creditCtx)
	if creditCtx.Currency != "USD" || creditCtx.RemainingAmount != 30000 {
		t.Fatalf("credit context %+v", creditCtx)
	}
}
