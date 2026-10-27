package httpserver_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const (
	msgToAmountRequired  = "Счета в разных валютах: укажите сумму зачисления"
	msgToAmountMismatch  = "Счета в одной валюте: сумма зачисления должна совпадать с суммой перевода"
	msgCurrencyMismatch  = "Валюты не совпадают: выберите счёт в той же валюте"
	msgBudgetCurrencyFix = "Валюту бюджета нельзя сменить: создайте новый бюджет"
)

type transferResp struct {
	GroupID         string `json:"group_id"`
	Amount          int64  `json:"amount"`
	ToAmount        int64  `json:"to_amount"`
	ToAmountDisplay string `json:"to_amount_display"`
	FromCurrency    string `json:"from_currency"`
	ToCurrency      string `json:"to_currency"`
	Rate            string `json:"rate"`
}

func currencyBalance(t *testing.T, env *testEnv, id string) int64 {
	t.Helper()
	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/accounts/"+id+"/balance", nil)
	if status != http.StatusOK {
		t.Fatalf("balance status %d: %s", status, raw)
	}
	var b struct {
		Balance int64 `json:"balance"`
	}
	_ = json.Unmarshal(raw, &b)
	return b.Balance
}

func currencyTransfer(t *testing.T, env *testEnv, method, path string, body map[string]any) (int, transferResp, []byte) {
	t.Helper()
	if _, ok := body["transaction_date"]; !ok {
		body["transaction_date"] = "2020-06-01 10:00:00"
	}
	status, raw := currencyDo(t, env, method, path, body)
	var tr transferResp
	_ = json.Unmarshal(raw, &tr)
	return status, tr, raw
}

func TestTransferExchangeCreate(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "10000.00", "currency": "RUB"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})

	status, _, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": usd.ID, "amount": "9000.00",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgToAmountRequired)

	status, tr, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": usd.ID, "amount": "9000.00", "to_amount": "100.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("exchange status %d: %s", status, raw)
	}
	if tr.Amount != 900000 || tr.ToAmount != 10000 || tr.ToAmountDisplay != "100.00" {
		t.Fatalf("amounts %+v", tr)
	}
	if tr.FromCurrency != "RUB" || tr.ToCurrency != "USD" || tr.Rate != "0.011111" {
		t.Fatalf("currencies/rate %+v", tr)
	}
	if got := currencyBalance(t, env, rub.ID); got != 100000 {
		t.Fatalf("rub balance %d", got)
	}
	if got := currencyBalance(t, env, usd.ID); got != 10000 {
		t.Fatalf("usd balance %d", got)
	}

	// Accounts summary and journal of the USD account carry to_amount and USD.
	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/accounts/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("summary status %d: %s", status, raw)
	}
	var summary struct {
		Accounts []struct {
			ID      string `json:"id"`
			Balance int64  `json:"balance"`
		} `json:"accounts"`
	}
	_ = json.Unmarshal(raw, &summary)
	found := false
	for _, a := range summary.Accounts {
		if a.ID == usd.ID {
			found = true
			if a.Balance != 10000 {
				t.Fatalf("summary usd balance %d", a.Balance)
			}
		}
	}
	if !found {
		t.Fatal("usd account missing in summary")
	}
	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/transactions?account_id="+usd.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("transactions status %d: %s", status, raw)
	}
	var list struct {
		Data []struct {
			AccountID string `json:"account_id"`
			Amount    int64  `json:"amount"`
			Currency  string `json:"currency"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &list)
	if len(list.Data) != 1 || list.Data[0].Amount != 10000 || list.Data[0].Currency != "USD" {
		t.Fatalf("usd journal %+v", list.Data)
	}
}

func TestTransferSameCurrencyToAmount(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	a := currencyCreateAccount(t, env, map[string]any{"name": "А", "type": "cash", "initial_balance": "1000.00"})
	b := currencyCreateAccount(t, env, map[string]any{"name": "Б", "type": "cash", "initial_balance": "0"})

	status, _, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": a.ID, "to_account_id": b.ID, "amount": "100.00", "to_amount": "90.00",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgToAmountMismatch)

	status, tr, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": a.ID, "to_account_id": b.ID, "amount": "100.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d: %s", status, raw)
	}
	if tr.ToAmount != 10000 || tr.Rate != "1.000000" || tr.FromCurrency != "RUB" || tr.ToCurrency != "RUB" {
		t.Fatalf("transfer %+v", tr)
	}

	status, _, raw = currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": a.ID, "to_account_id": b.ID, "amount": "50.00", "to_amount": "50.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("equal to_amount status %d: %s", status, raw)
	}
	if got := currencyBalance(t, env, b.ID); got != 15000 {
		t.Fatalf("b balance %d", got)
	}
}

func TestTransferExchangeCommission(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "10000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})

	status, _, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": usd.ID, "amount": "9000.00", "to_amount": "100.00", "commission": "50.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d: %s", status, raw)
	}
	var n int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE account_id = ? AND type = 'expense' AND amount = 5000`, rub.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("commission legs on RUB %d err %v", n, err)
	}
	if got := currencyBalance(t, env, rub.ID); got != 1000000-900000-5000 {
		t.Fatalf("rub balance %d", got)
	}
	if got := currencyBalance(t, env, usd.ID); got != 10000 {
		t.Fatalf("usd balance %d", got)
	}
}

func TestTransferUpdateCurrencyRule(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub1 := currencyCreateAccount(t, env, map[string]any{"name": "Рубли 1", "type": "cash", "initial_balance": "10000.00"})
	rub2 := currencyCreateAccount(t, env, map[string]any{"name": "Рубли 2", "type": "cash", "initial_balance": "0"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})

	status, tr, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub1.ID, "to_account_id": rub2.ID, "amount": "9000.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("status %d: %s", status, raw)
	}
	path := "/api/v1/transfers/" + tr.GroupID

	status, _, raw = currencyTransfer(t, env, http.MethodPut, path, map[string]any{
		"from_account_id": rub1.ID, "to_account_id": usd.ID, "amount": "9000.00",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgToAmountRequired)

	status, got, raw := currencyTransfer(t, env, http.MethodPut, path, map[string]any{
		"from_account_id": rub1.ID, "to_account_id": usd.ID, "amount": "9000.00", "to_amount": "100.00",
	})
	if status != http.StatusOK {
		t.Fatalf("put status %d: %s", status, raw)
	}
	if got.ToAmount != 10000 || got.ToCurrency != "USD" {
		t.Fatalf("put transfer %+v", got)
	}
	if b := currencyBalance(t, env, rub2.ID); b != 0 {
		t.Fatalf("rub2 balance %d", b)
	}
	if b := currencyBalance(t, env, usd.ID); b != 10000 {
		t.Fatalf("usd balance %d", b)
	}

	status, got, raw = currencyTransfer(t, env, http.MethodPut, path, map[string]any{
		"from_account_id": rub1.ID, "to_account_id": usd.ID, "amount": "8000.00", "to_amount": "90.00",
	})
	if status != http.StatusOK {
		t.Fatalf("put2 status %d: %s", status, raw)
	}
	if got.Amount != 800000 || got.ToAmount != 9000 {
		t.Fatalf("put2 transfer %+v", got)
	}
	if b := currencyBalance(t, env, rub1.ID); b != 1000000-800000 {
		t.Fatalf("rub1 balance %d", b)
	}
	if b := currencyBalance(t, env, usd.ID); b != 9000 {
		t.Fatalf("usd balance after put2 %d", b)
	}
}

func TestCreditCardPaymentAccountCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})
	rubPayer := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})

	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/accounts", map[string]any{
		"name": "Карта", "type": "credit_card", "bank_id": "tinkoff",
		"credit_limit": "1000.00", "initial_balance": "1000.00", "payment_account_id": usd.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	card := currencyCreateAccount(t, env, map[string]any{
		"name": "Карта", "type": "credit_card", "bank_id": "tinkoff",
		"credit_limit": "1000.00", "initial_balance": "1000.00", "payment_account_id": rubPayer.ID,
	})
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+card.ID, map[string]any{
		"name": card.Name, "bank_id": "tinkoff", "payment_account_id": usd.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	// Changing the card currency while its payment account stays RUB.
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+card.ID, map[string]any{
		"name": card.Name, "bank_id": "tinkoff", "currency": "USD",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
}

func TestTransferExchangeToCreditCardLimit(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "100000.00"})
	// USD card: limit 1000, balance 900 → room 100.
	card := currencyCreateAccount(t, env, map[string]any{
		"name": "USD карта", "type": "credit_card", "bank_id": "tinkoff",
		"credit_limit": "1000.00", "initial_balance": "900.00", "currency": "USD",
	})

	status, _, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": card.ID, "amount": "13500.00", "to_amount": "150.00",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, "ERR_CREDIT_CARD_PAYMENT_EXCEEDS_LIMIT")

	status, tr, raw := currencyTransfer(t, env, http.MethodPost, "/api/v1/transfers", map[string]any{
		"from_account_id": rub.ID, "to_account_id": card.ID, "amount": "9000.00", "to_amount": "100.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("exchange to card status %d: %s", status, raw)
	}
	if tr.Amount != 900000 || tr.ToAmount != 10000 {
		t.Fatalf("transfer %+v", tr)
	}
	if b := currencyBalance(t, env, card.ID); b != 100000 {
		t.Fatalf("card balance %d", b)
	}
}

func TestAutoTopupBeneficiaryCurrencyChange(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	src := currencyCreateAccount(t, env, map[string]any{"name": "Источник", "type": "bank", "bank_id": "tinkoff", "initial_balance": "0"})
	dst := currencyCreateAccount(t, env, map[string]any{"name": "Получатель", "type": "bank", "bank_id": "tinkoff", "initial_balance": "1000.00"})
	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+dst.ID, map[string]any{
		"name": dst.Name, "bank_id": "tinkoff",
		"auto_topup_enabled": true, "auto_topup_threshold": "10.00", "auto_topup_target": "100.00",
		"auto_topup_source_account_id": src.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("enable auto topup status %d: %s", status, raw)
	}
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+dst.ID, map[string]any{
		"name": dst.Name, "bank_id": "tinkoff", "currency": "USD",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
	var cur string
	if err := env.db.QueryRow(`SELECT currency FROM accounts WHERE id = ?`, dst.ID).Scan(&cur); err != nil || cur != "RUB" {
		t.Fatalf("currency %q err %v", cur, err)
	}
}

func TestAutoTopupSourceCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	src := currencyCreateAccount(t, env, map[string]any{"name": "Источник", "type": "bank", "bank_id": "tinkoff", "initial_balance": "0", "currency": "USD"})
	dst := currencyCreateAccount(t, env, map[string]any{"name": "Получатель", "type": "bank", "bank_id": "tinkoff", "initial_balance": "1000.00"})
	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+dst.ID, map[string]any{
		"name": dst.Name, "bank_id": "tinkoff",
		"auto_topup_enabled": true, "auto_topup_threshold": "10.00", "auto_topup_target": "100.00",
		"auto_topup_source_account_id": src.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
	var enabled int
	if err := env.db.QueryRow(`SELECT auto_topup_enabled FROM accounts WHERE id = ?`, dst.ID).Scan(&enabled); err != nil || enabled != 0 {
		t.Fatalf("auto topup enabled %d err %v", enabled, err)
	}
}

func TestArchiveDeleteTransferTargetCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})

	for _, action := range []string{"archive", "delete"} {
		t.Run(action, func(t *testing.T) {
			rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB " + action, "type": "cash", "initial_balance": "500.00"})
			method := http.MethodPost
			path := "/api/v1/accounts/" + rub.ID + "/archive?transfer_to_account_id=" + usd.ID
			if action == "delete" {
				method = http.MethodDelete
				path = "/api/v1/accounts/" + rub.ID + "?transfer_to_account_id=" + usd.ID
			}
			status, raw := currencyDo(t, env, method, path, nil)
			assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
			var st string
			if err := env.db.QueryRow(`SELECT status FROM accounts WHERE id = ?`, rub.ID).Scan(&st); err != nil || st != "active" {
				t.Fatalf("status %q err %v", st, err)
			}
			var n int
			if err := env.db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE account_id IN (?, ?)`, rub.ID, usd.ID).Scan(&n); err != nil || n != 0 {
				t.Fatalf("transactions %d err %v", n, err)
			}
		})
	}
}

func TestCreditCurrencyLinks(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "0"})
	rub2 := currencyCreateAccount(t, env, map[string]any{"name": "RUB 2", "type": "cash", "initial_balance": "0"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})

	credit := createCredit(t, env, map[string]any{
		"name":             "Кредит",
		"principal_amount": "30000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      3,
		"interest_rate":    0,
		"debit_account_id": rub.ID,
	})
	creditID := credit["id"].(string)

	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/credits/"+creditID, map[string]any{"debit_account_id": usd.ID})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/credits/"+creditID, map[string]any{"debit_account_id": rub2.ID})
	if status != http.StatusOK {
		t.Fatalf("same currency debit change status %d: %s", status, raw)
	}

	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/credits", map[string]any{
		"name":                         "Ипотека",
		"credit_kind":                  "mortgage",
		"property_price":               "1000000.00",
		"down_payment":                 "200000.00",
		"down_payment_affects_balance": true,
		"down_payment_account_id":      usd.ID,
		"issue_date":                   time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":                  12,
		"interest_rate":                10,
		"payment_interval":             "month",
		"debit_account_id":             rub.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
	var n int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM credits WHERE name = 'Ипотека'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("mortgage rows %d err %v", n, err)
	}
}

func TestCreditPaymentAccountCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "100000.00"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "1000.00", "currency": "USD"})
	credit := createCredit(t, env, map[string]any{
		"name":                "Кредит",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    rub.ID,
		"added_retroactively": false,
	})
	path := "/api/v1/credits/" + credit["id"].(string) + "/payments"
	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")

	status, raw := currencyDo(t, env, http.MethodPost, path, map[string]any{
		"amount": "5000.00", "payment_date": payDate, "account_id": usd.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/transactions", map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "50.00",
		"category_id": getExpenseCategory(t, env), "transaction_date": payDate,
	})
	if status != http.StatusCreated {
		t.Fatalf("create usd expense status %d: %s", status, raw)
	}
	var tx struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &tx)
	status, raw = currencyDo(t, env, http.MethodPost, path, map[string]any{"transaction_id": tx.ID})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	var n int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM credit_payments WHERE credit_id = ? AND is_applied = 1`, credit["id"]).Scan(&n); err != nil || n != 0 {
		t.Fatalf("applied payments %d err %v", n, err)
	}
}

func TestBudgetCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "0", "currency": "USD"})
	cat := getExpenseCategory(t, env)

	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Несовпадение", "scope": "all_expense", "amount": "100.00", "account_id": usd.ID, "currency": "RUB",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Неизвестная", "scope": "all_expense", "amount": "100.00", "currency": "XYZ",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyInvalid)

	var b struct {
		ID       string `json:"id"`
		Currency string `json:"currency"`
	}
	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Долларовый", "scope": "all_expense", "amount": "100.00", "account_id": usd.ID,
	})
	if status != http.StatusCreated {
		t.Fatalf("create status %d: %s", status, raw)
	}
	_ = json.Unmarshal(raw, &b)
	if b.Currency != "USD" {
		t.Fatalf("budget currency %q, want USD", b.Currency)
	}

	status, raw = currencyDo(t, env, http.MethodPatch, "/api/v1/budgets/"+b.ID, map[string]any{
		"name": "Долларовый", "scope": "all_expense", "amount": "100.00", "account_id": usd.ID, "currency": "RUB",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgBudgetCurrencyFix)

	status, raw = currencyDo(t, env, http.MethodPatch, "/api/v1/budgets/"+b.ID, map[string]any{
		"name": "Долларовый", "scope": "all_expense", "amount": "100.00", "account_id": usd.ID, "currency": "XYZ",
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyInvalid)

	status, raw = currencyDo(t, env, http.MethodPatch, "/api/v1/budgets/"+b.ID, map[string]any{
		"name": "Долларовый 2", "scope": "all_expense", "amount": "150.00", "account_id": usd.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("update without currency status %d: %s", status, raw)
	}
	_ = json.Unmarshal(raw, &b)
	if b.Currency != "USD" {
		t.Fatalf("updated currency %q", b.Currency)
	}

	var plain struct {
		Currency string `json:"currency"`
	}
	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Категория", "scope": "category", "category_id": cat, "amount": "100.00",
	})
	if status != http.StatusCreated {
		t.Fatalf("create plain status %d: %s", status, raw)
	}
	_ = json.Unmarshal(raw, &plain)
	if plain.Currency != "RUB" {
		t.Fatalf("plain budget currency %q, want RUB", plain.Currency)
	}

	// A RUB budget without account cannot be moved to a USD account.
	var rubBudget struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &rubBudget)
	status, raw = currencyDo(t, env, http.MethodPatch, "/api/v1/budgets/"+rubBudget.ID, map[string]any{
		"name": "Категория", "scope": "category", "category_id": cat, "amount": "100.00", "account_id": usd.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)
	var accID sql.NullString
	var cur string
	if err := env.db.QueryRow(`SELECT account_id, currency FROM budgets WHERE id = ?`, rubBudget.ID).Scan(&accID, &cur); err != nil {
		t.Fatal(err)
	}
	if accID.Valid || cur != "RUB" {
		t.Fatalf("budget changed: account %v currency %q", accID, cur)
	}

	// Copy to next month keeps the source currency.
	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/budgets/"+b.ID+"/copy-next", nil)
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("copy-next status %d: %s", status, raw)
	}
	var copied struct {
		Currency string `json:"currency"`
	}
	_ = json.Unmarshal(raw, &copied)
	if copied.Currency != "USD" {
		t.Fatalf("copied currency %q: %s", copied.Currency, raw)
	}
}

func TestBudgetUniquenessPerCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	for _, cur := range []string{"RUB", "USD"} {
		status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
			"name": "Все расходы " + cur, "scope": "all_expense", "amount": "100.00", "currency": cur,
		})
		if status != http.StatusCreated {
			t.Fatalf("create %s budget status %d: %s", cur, status, raw)
		}
	}
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
		"name": "Дубль", "scope": "all_expense", "amount": "100.00", "currency": "USD",
	})
	if status != http.StatusConflict {
		t.Fatalf("duplicate USD budget status %d: %s", status, raw)
	}
}

func TestArchiveSameCurrencyTransferSucceeds(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	src := currencyCreateAccount(t, env, map[string]any{"name": "Источник", "type": "cash", "initial_balance": "500.00"})
	dst := currencyCreateAccount(t, env, map[string]any{"name": "Цель", "type": "cash", "initial_balance": "0"})
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/accounts/"+src.ID+"/archive?transfer_to_account_id="+dst.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("archive status %d: %s", status, raw)
	}
	var st string
	if err := env.db.QueryRow(`SELECT status FROM accounts WHERE id = ?`, src.ID).Scan(&st); err != nil || st != "archived" {
		t.Fatalf("status %q err %v", st, err)
	}
	if got := currencyBalance(t, env, dst.ID); got != 50000 {
		t.Fatalf("dst balance %d", got)
	}
}

func TestAutoTopupSameCurrencySucceeds(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")
	src := currencyCreateAccount(t, env, map[string]any{"name": "Источник", "type": "bank", "bank_id": "tinkoff", "initial_balance": "0"})
	dst := currencyCreateAccount(t, env, map[string]any{"name": "Получатель", "type": "bank", "bank_id": "tinkoff", "initial_balance": "1000.00"})
	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+dst.ID, map[string]any{
		"name": dst.Name, "bank_id": "tinkoff",
		"auto_topup_enabled": true, "auto_topup_threshold": "10.00", "auto_topup_target": "100.00",
		"auto_topup_source_account_id": src.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("enable auto topup status %d: %s", status, raw)
	}
	var enabled int
	if err := env.db.QueryRow(`SELECT auto_topup_enabled FROM accounts WHERE id = ?`, dst.ID).Scan(&enabled); err != nil || enabled != 1 {
		t.Fatalf("auto topup enabled %d err %v", enabled, err)
	}
}

func TestInboxAndListCarryUsdCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD inbox", "type": "cash", "initial_balance": "0", "currency": "USD"})
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/transactions", map[string]any{
		"account_id": usd.ID, "type": "expense", "amount": "12.00",
		"transaction_date": "2026-01-15 12:00:00", "description": "без категории",
	})
	if status != http.StatusCreated {
		t.Fatalf("create inbox tx status %d: %s", status, raw)
	}
	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/transactions/inbox", nil)
	if status != http.StatusOK {
		t.Fatalf("inbox status %d: %s", status, raw)
	}
	var inbox struct {
		Data []struct {
			Currency string `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &inbox); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tx := range inbox.Data {
		if tx.Currency == "USD" {
			found = true
		}
	}
	if !found {
		t.Fatalf("inbox missing USD row: %s", raw)
	}
}

func TestDebtCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	usd := currencyCreateAccount(t, env, map[string]any{"name": "USD", "type": "cash", "initial_balance": "100.00", "currency": "USD"})
	rub := currencyCreateAccount(t, env, map[string]any{"name": "RUB", "type": "cash", "initial_balance": "100.00"})

	debt := createDebt(t, env, map[string]any{
		"debtor_name": "Денис", "direction": "lent", "amount": "50.00",
		"affects_balance": true, "account_id": usd.ID,
	})
	if debt["currency"] != "USD" {
		t.Fatalf("debt currency %v", debt["currency"])
	}
	noAcc := createDebt(t, env, map[string]any{
		"debtor_name": "Петя", "direction": "lent", "amount": "10.00", "affects_balance": false,
	})
	if noAcc["currency"] != "RUB" {
		t.Fatalf("debt without account currency %v", noAcc["currency"])
	}

	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/debts/"+debt["id"].(string)+"/settle", map[string]any{
		"settled_at": "2020-07-01 12:00:00", "affects_balance": true, "account_id": rub.ID,
	})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyMismatch)

	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/debts/"+debt["id"].(string)+"/settle", map[string]any{
		"settled_at": "2020-07-01 12:00:00", "affects_balance": true, "account_id": usd.ID,
	})
	if status != http.StatusOK {
		t.Fatalf("settle same currency status %d: %s", status, raw)
	}
}
