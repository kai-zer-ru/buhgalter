package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

const (
	msgCurrencyInvalid = "Неподдерживаемая валюта. Допустимы: RUB, USD, EUR, CNY, KZT, BYN, TRY, GBP"
	msgCurrencyLocked  = "Валюту счёта нельзя сменить: по счёту уже есть операции или он используется в кредите, подписке, плановой операции, бюджете или другом счёте"
)

type currencyAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

func currencyDo(t *testing.T, env *testEnv, method, path string, body any) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	resp, err := env.authedRequest(method, path, r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out
}

func currencyCreateAccount(t *testing.T, env *testEnv, body map[string]any) currencyAccount {
	t.Helper()
	status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/accounts", body)
	if status != http.StatusCreated {
		t.Fatalf("create account status %d: %s", status, raw)
	}
	var acc currencyAccount
	if err := json.Unmarshal(raw, &acc); err != nil {
		t.Fatal(err)
	}
	return acc
}

func assertErrorMessage(t *testing.T, status int, raw []byte, wantStatus int, wantMsg string) {
	t.Helper()
	if status != wantStatus {
		t.Fatalf("status %d, want %d: %s", status, wantStatus, raw)
	}
	var errBody struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &errBody)
	if errBody.Error.Code != "VALIDATION_ERROR" {
		t.Fatalf("error code %q", errBody.Error.Code)
	}
	if errBody.Error.Message != wantMsg {
		t.Fatalf("error message %q, want %q", errBody.Error.Message, wantMsg)
	}
}

func TestCurrenciesEndpoint(t *testing.T) {
	env := setupConfigured(t)

	resp, err := http.Get(env.server.URL + "/api/v1/currencies")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("currencies status %d", resp.StatusCode)
	}
	var got []struct {
		Code   string `json:"code"`
		Symbol string `json:"symbol"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := [][2]string{
		{"RUB", "₽"}, {"USD", "$"}, {"EUR", "€"}, {"CNY", "¥"},
		{"KZT", "₸"}, {"BYN", "Br"}, {"TRY", "₺"}, {"GBP", "£"},
	}
	if len(got) != len(want) {
		t.Fatalf("currencies len %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Code != w[0] || got[i].Symbol != w[1] {
			t.Fatalf("[%d] = %+v, want %v", i, got[i], w)
		}
	}
}

func TestProfileCurrencyAllowlist(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/user/settings", map[string]string{"currency": "KZT"})
	if status != http.StatusOK {
		t.Fatalf("put KZT status %d: %s", status, raw)
	}
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/user/settings", map[string]string{"currency": "XYZ"})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyInvalid)
}

func TestCreateAccountCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/user/settings", map[string]string{"currency": "EUR"})
	if status != http.StatusOK {
		t.Fatalf("put EUR status %d: %s", status, raw)
	}

	def := currencyCreateAccount(t, env, map[string]any{"name": "Профиль", "type": "cash", "initial_balance": "0"})
	if def.Currency != "EUR" {
		t.Fatalf("default currency %q, want EUR", def.Currency)
	}
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})
	if usd.Currency != "USD" {
		t.Fatalf("currency %q, want USD", usd.Currency)
	}

	status, raw = currencyDo(t, env, http.MethodGet, "/api/v1/accounts/"+usd.ID, nil)
	if status != http.StatusOK {
		t.Fatalf("get status %d", status)
	}
	var got currencyAccount
	_ = json.Unmarshal(raw, &got)
	if got.Currency != "USD" {
		t.Fatalf("get currency %q", got.Currency)
	}

	status, raw = currencyDo(t, env, http.MethodPost, "/api/v1/accounts",
		map[string]any{"name": "Неизвестная", "type": "cash", "initial_balance": "0", "currency": "XYZ"})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyInvalid)

	lower := currencyCreateAccount(t, env, map[string]any{"name": "Нижний регистр", "type": "cash", "initial_balance": "0", "currency": " usd "})
	if lower.Currency != "USD" {
		t.Fatalf("normalized currency %q, want USD", lower.Currency)
	}
}

func TestUpdateAccountCurrencyLock(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")

	// Empty account: change allowed; initial_balance does not lock.
	empty := currencyCreateAccount(t, env, map[string]any{"name": "Пустой", "type": "cash", "initial_balance": "100.00"})
	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+empty.ID,
		map[string]any{"name": empty.Name, "currency": "USD"})
	if status != http.StatusOK {
		t.Fatalf("update empty status %d: %s", status, raw)
	}
	var upd currencyAccount
	_ = json.Unmarshal(raw, &upd)
	if upd.Currency != "USD" {
		t.Fatalf("currency %q, want USD", upd.Currency)
	}

	// Old client without the field keeps the currency.
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+empty.ID,
		map[string]any{"name": "Пустой 2"})
	if status != http.StatusOK {
		t.Fatalf("update without currency status %d: %s", status, raw)
	}
	_ = json.Unmarshal(raw, &upd)
	if upd.Currency != "USD" {
		t.Fatalf("currency changed without field: %q", upd.Currency)
	}

	// Unknown code.
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+empty.ID,
		map[string]any{"name": "Пустой 2", "currency": "XYZ"})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyInvalid)

	// Account with an income: change locked, same currency still OK.
	withTx := currencyCreateAccount(t, env, map[string]any{"name": "С операцией", "type": "cash", "initial_balance": "0"})
	createTx(t, env, map[string]any{
		"account_id": withTx.ID, "type": "income", "amount": "10.00",
		"category_id": getCategoryByType(t, env, "income"), "transaction_date": "2026-01-10 11:00:00",
	})
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+withTx.ID,
		map[string]any{"name": withTx.Name, "currency": "USD"})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyLocked)
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+withTx.ID,
		map[string]any{"name": withTx.Name, "currency": withTx.Currency})
	if status != http.StatusOK {
		t.Fatalf("same currency status %d: %s", status, raw)
	}

	// Account referenced as a credit card payment account: locked.
	payer := currencyCreateAccount(t, env, map[string]any{"name": "Плательщик", "type": "bank", "bank_id": "tinkoff", "initial_balance": "0"})
	currencyCreateAccount(t, env, map[string]any{
		"name": "Кредитка", "type": "credit_card", "bank_id": "tinkoff",
		"credit_limit": "1000.00", "initial_balance": "1000.00", "payment_account_id": payer.ID,
	})
	status, raw = currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+payer.ID,
		map[string]any{"name": payer.Name, "bank_id": "tinkoff", "currency": "USD"})
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyLocked)
}

func TestTransactionsCarryAccountCurrency(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	rub := currencyCreateAccount(t, env, map[string]any{"name": "Рубли", "type": "cash", "initial_balance": "0", "currency": "RUB"})
	usd := currencyCreateAccount(t, env, map[string]any{"name": "Доллары", "type": "cash", "initial_balance": "0", "currency": "USD"})
	incomeCat := getCategoryByType(t, env, "income")
	createTx(t, env, map[string]any{
		"account_id": rub.ID, "type": "income", "amount": "10.00",
		"category_id": incomeCat, "transaction_date": "2026-01-10 11:00:00",
	})
	createTx(t, env, map[string]any{
		"account_id": usd.ID, "type": "income", "amount": "20.00",
		"category_id": incomeCat, "transaction_date": "2026-01-11 11:00:00",
	})

	status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/transactions", nil)
	if status != http.StatusOK {
		t.Fatalf("list status %d", status)
	}
	var list struct {
		Data []struct {
			ID        string `json:"id"`
			AccountID string `json:"account_id"`
			Currency  string `json:"currency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{rub.ID: "RUB", usd.ID: "USD"}
	seen := 0
	for _, tx := range list.Data {
		w, ok := want[tx.AccountID]
		if !ok {
			continue
		}
		seen++
		if tx.Currency != w {
			t.Fatalf("tx on %s currency %q, want %q", tx.AccountID, tx.Currency, w)
		}
		status, raw := currencyDo(t, env, http.MethodGet, "/api/v1/transactions/"+tx.ID, nil)
		if status != http.StatusOK {
			t.Fatalf("get tx status %d", status)
		}
		var one struct {
			Currency string `json:"currency"`
		}
		_ = json.Unmarshal(raw, &one)
		if one.Currency != w {
			t.Fatalf("get tx currency %q, want %q", one.Currency, w)
		}
	}
	if seen != 2 {
		t.Fatalf("saw %d transactions, want 2", seen)
	}
}

// assertNoAccountTransactions guards reference cases: the lock must come from the
// reference itself, not from a transaction the API created on the account.
func assertNoAccountTransactions(t *testing.T, env *testEnv, accountID string) {
	t.Helper()
	var n int
	if err := env.db.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE account_id = ? OR transfer_account_id = ?`,
		accountID, accountID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("account %s has %d transactions; reference case must not create any", accountID, n)
	}
}

func assertCurrencyLocked(t *testing.T, env *testEnv, acc currencyAccount, extra map[string]any) {
	t.Helper()
	body := map[string]any{"name": acc.Name, "currency": "USD"}
	for k, v := range extra {
		body[k] = v
	}
	status, raw := currencyDo(t, env, http.MethodPut, "/api/v1/accounts/"+acc.ID, body)
	assertErrorMessage(t, status, raw, http.StatusBadRequest, msgCurrencyLocked)
}

func TestUpdateAccountCurrencyLockedByReferences(t *testing.T) {
	env := setupConfigured(t)
	seedBanks(t, env)
	env.login(t, "admin", "secret123")
	future := timeutil.FormatUTC(timeutil.NowUTC().AddDate(0, 1, 0))

	t.Run("transfer_account_id", func(t *testing.T) {
		// POST /transfers always writes a leg with account_id = destination, so the
		// transfer_account_id-only state is seeded directly.
		other := currencyCreateAccount(t, env, map[string]any{"name": "Перевод-источник", "type": "cash", "initial_balance": "0"})
		acc := currencyCreateAccount(t, env, map[string]any{"name": "Перевод-цель", "type": "cash", "initial_balance": "0"})
		if _, err := env.db.Exec(`
			INSERT INTO transactions (id, user_id, account_id, type, amount, transfer_group_id, transfer_account_id, transaction_date)
			SELECT 'tx-transfer-only', user_id, ?, 'transfer', 100, 'g-transfer-only', ?, '2026-01-01 00:00:00'
			FROM accounts WHERE id = ?`, other.ID, acc.ID, other.ID); err != nil {
			t.Fatal(err)
		}
		var n int
		if err := env.db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE account_id = ?`, acc.ID).Scan(&n); err != nil || n != 0 {
			t.Fatalf("account_id rows %d err %v", n, err)
		}
		assertCurrencyLocked(t, env, acc, nil)
	})

	t.Run("credit debit_account_id", func(t *testing.T) {
		acc := currencyCreateAccount(t, env, map[string]any{"name": "Кредит-счёт", "type": "cash", "initial_balance": "0"})
		createCredit(t, env, map[string]any{
			"name":                "Кредит",
			"principal_amount":    "1000.00",
			"issue_date":          timeutil.NowUTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
			"term_months":         12,
			"interest_rate":       0,
			"payment_interval":    "month",
			"debit_account_id":    acc.ID,
			"added_retroactively": false,
		})
		assertNoAccountTransactions(t, env, acc.ID)
		assertCurrencyLocked(t, env, acc, nil)
	})

	t.Run("subscription", func(t *testing.T) {
		acc := currencyCreateAccount(t, env, map[string]any{"name": "Подписка-счёт", "type": "cash", "initial_balance": "0"})
		status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/subscriptions", map[string]any{
			"name": "Музыка", "amount": "10.00", "account_id": acc.ID,
			"period": "month", "day_of_month": 1, "start_date": future, "time_local": "08:00",
		})
		if status != http.StatusCreated {
			t.Fatalf("create subscription status %d: %s", status, raw)
		}
		assertNoAccountTransactions(t, env, acc.ID)
		assertCurrencyLocked(t, env, acc, nil)
	})

	t.Run("recurring operation", func(t *testing.T) {
		acc := currencyCreateAccount(t, env, map[string]any{"name": "Плановая-счёт", "type": "cash", "initial_balance": "0"})
		status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/recurring-operations", map[string]any{
			"type": "expense", "amount": "75.00", "account_id": acc.ID,
			"category_id": getExpenseCategory(t, env), "period": "month", "day_of_month": 1,
			"start_date": future, "time_local": "08:00",
		})
		if status != http.StatusCreated {
			t.Fatalf("create recurring status %d: %s", status, raw)
		}
		assertNoAccountTransactions(t, env, acc.ID)
		assertCurrencyLocked(t, env, acc, nil)
	})

	t.Run("budget account_id", func(t *testing.T) {
		acc := currencyCreateAccount(t, env, map[string]any{"name": "Бюджет-счёт", "type": "cash", "initial_balance": "0"})
		status, raw := currencyDo(t, env, http.MethodPost, "/api/v1/budgets?month=2026-01", map[string]any{
			"name": "Всё по счёту", "scope": "all_expense", "amount": "100.00", "account_id": acc.ID,
		})
		if status != http.StatusCreated {
			t.Fatalf("create budget status %d: %s", status, raw)
		}
		assertNoAccountTransactions(t, env, acc.ID)
		assertCurrencyLocked(t, env, acc, nil)
	})

	t.Run("auto topup source", func(t *testing.T) {
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
		assertNoAccountTransactions(t, env, src.ID)
		assertCurrencyLocked(t, env, src, map[string]any{"bank_id": "tinkoff"})
	})
}

func TestCreateAccountUnsupportedProfileCurrencyFallsBack(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	// Native import can store a profile code outside the allowlist.
	if _, err := env.db.Exec(`UPDATE users SET currency = 'XYZ'`); err != nil {
		t.Fatal(err)
	}
	acc := currencyCreateAccount(t, env, map[string]any{"name": "Фолбэк", "type": "cash", "initial_balance": "0"})
	if acc.Currency != "RUB" {
		t.Fatalf("currency %q, want RUB", acc.Currency)
	}
}
