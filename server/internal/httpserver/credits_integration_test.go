package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/credit"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

func createCredit(t *testing.T, env *testEnv, body map[string]any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("create credit status %d: %v", resp.StatusCode, errBody)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func TestCreateCreditWithSchedule(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Ипотека-счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Ипотека",
		"principal_amount":    "100000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         12,
		"interest_rate":       0,
		"payment_interval":    "month",
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	if credit["id"] == nil {
		t.Fatal("expected credit id")
	}
	schedule, ok := credit["schedule"].([]any)
	if !ok || len(schedule) != 12 {
		t.Fatalf("expected 12 schedule entries, got %v", credit["schedule"])
	}
	if remaining, ok := credit["remaining_amount"].(float64); !ok || int64(remaining) != 10000000 {
		t.Fatalf("remaining %v", credit["remaining_amount"])
	}
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["transaction_id"] != nil {
			t.Fatal("scheduled payment must not have precreated transaction")
		}
	}
}

func TestCreateCreditRejectsTooLowMonthlyPayment(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	body, _ := json.Marshal(map[string]any{
		"name":             "Потреб",
		"principal_amount": "100000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      12,
		"interest_rate":    12.0,
		"payment_interval": "month",
		"monthly_payment":  "100.00",
		"debit_account_id": accID,
	})

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 400 for too low monthly payment, got %d: %v", resp.StatusCode, errBody)
	}
}

func TestCreateCreditRejectsTooHighMonthlyPaymentForTerm(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	body, _ := json.Marshal(map[string]any{
		"name":             "Кредит",
		"credit_kind":      "consumer",
		"principal_amount": "36800000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      360,
		"interest_rate":    40.0,
		"payment_interval": "month",
		"monthly_payment":  "3771000.00",
		"debit_account_id": accID,
	})

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 400 for too high monthly payment, got %d: %v", resp.StatusCode, errBody)
	}
}

func TestCreateCreditWithoutAutoTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"principal_amount":    "10000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         3,
		"interest_rate":       0,
		"payment_interval":    "month",
		"debit_account_id":    accID,
		"create_transactions": false,
	})
	for _, item := range credit["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["transaction_id"] != nil {
			t.Fatal("expected no transaction_id when create_transactions is false")
		}
	}
}

func TestCreateCreditRetroactiveWithAutoTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Ретро-счёт")

	issueDate := time.Now().UTC().AddDate(0, -3, 0).Format("2006-01-02 00:00:00")
	credit := createCredit(t, env, map[string]any{
		"principal_amount":    "12000.00",
		"issue_date":          issueDate,
		"term_months":         12,
		"interest_rate":       0,
		"payment_interval":    "month",
		"debit_account_id":    accID,
		"added_retroactively": true,
		"create_transactions": true,
	})
	var retroWithoutTx, pendingWithoutTx int
	for _, item := range credit["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "retroactive" {
			if row["transaction_id"] != nil {
				t.Fatal("retroactive payment must not have transaction")
			}
			retroWithoutTx++
		}
		if row["kind"] == "scheduled" && row["is_applied"] != true {
			if row["transaction_id"] != nil {
				t.Fatal("pending scheduled payment must not have precreated transaction")
			}
			pendingWithoutTx++
		}
	}
	if retroWithoutTx == 0 {
		t.Fatal("expected retroactive payments")
	}
	if pendingWithoutTx == 0 {
		t.Fatal("expected pending scheduled payments")
	}
}

func TestRepairEmptyCreditSchedule(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"principal_amount":    "12000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         6,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"create_transactions": false,
	})
	creditID := credit["id"].(string)

	if _, err := env.db.Exec("DELETE FROM credit_payments WHERE credit_id = ?", creditID); err != nil {
		t.Fatal(err)
	}

	listResp, err := env.authedRequest(http.MethodGet, "/api/v1/credits?status=active", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var list []map[string]any
	_ = json.NewDecoder(listResp.Body).Decode(&list)
	var found map[string]any
	for _, item := range list {
		if item["id"] == creditID {
			found = item
			break
		}
	}
	if found == nil {
		t.Fatal("credit not in list")
	}
	if found["next_payment_date"] == nil {
		t.Fatal("expected next_payment_date after schedule repair")
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	var detail map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&detail)
	schedule, ok := detail["schedule"].([]any)
	if !ok || len(schedule) != 6 {
		t.Fatalf("expected repaired schedule with 6 rows, got %v", detail["schedule"])
	}
}

func TestPayScheduledPaymentCreatesFutureTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Потреб",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)

	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")

	body, _ := json.Marshal(map[string]any{
		"amount":       "5000.00",
		"payment_date": payDate,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", resp.StatusCode)
	}
	var updated map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&updated)
	paid := int64(updated["paid_amount"].(float64))
	if paid != 500000 {
		t.Fatalf("paid_amount %d", paid)
	}

	schedule := updated["schedule"].([]any)
	found := false
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] == true && row["transaction_id"] != nil {
			if row["payment_date"] != payDate {
				t.Fatalf("expected payment date %q, got %v", payDate, row["payment_date"])
			}
			txID := row["transaction_id"].(string)
			txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
			defer txResp.Body.Close()
			var tx map[string]any
			_ = json.NewDecoder(txResp.Body).Decode(&tx)
			if tx["kind"] != "manual" {
				t.Fatalf("expected manual tx, got %v", tx["kind"])
			}
			found = true
		}
	}
	if !found {
		t.Fatal("scheduled payment with future transaction not found")
	}

	var sameDayCount int
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["payment_date"] == payDate {
			sameDayCount++
		}
	}
	if sameDayCount != 1 {
		t.Fatalf("expected 1 payment on first date, got %d", sameDayCount)
	}

	var pendingScheduled int
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] != true {
			pendingScheduled++
		}
	}
	if pendingScheduled != 4 {
		t.Fatalf("expected 4 pending scheduled payments, got %d", pendingScheduled)
	}
}

func TestDeleteCreditPaymentTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Потреб",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)

	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")

	payBody, _ := json.Marshal(map[string]any{
		"amount":       "10000.00",
		"payment_date": payDate,
	})
	payResp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(payBody))
	if err != nil {
		t.Fatal(err)
	}
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", payResp.StatusCode)
	}
	var paid map[string]any
	_ = json.NewDecoder(payResp.Body).Decode(&paid)
	payResp.Body.Close()

	var txID string
	for _, item := range paid["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] == true && row["transaction_id"] != nil {
			txID = row["transaction_id"].(string)
			break
		}
	}
	if txID == "" {
		t.Fatal("paid scheduled payment without transaction")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/transactions/"+txID, nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete transaction status %d", delResp.StatusCode)
	}

	getResp, err := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	var updated map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&updated)

	if int64(updated["paid_amount"].(float64)) != 0 {
		t.Fatalf("paid_amount should be 0, got %v", updated["paid_amount"])
	}

	var reverted bool
	for _, item := range updated["schedule"].([]any) {
		row := item.(map[string]any)
		if row["transaction_id"] == nil && row["kind"] == "scheduled" && row["is_applied"] != true {
			if row["is_applied"] == true || row["transaction_id"] != nil {
				t.Fatalf("payment should be reverted, got %+v", row)
			}
			reverted = true
		}
	}
	if !reverted {
		t.Fatal("first scheduled payment not found after revert")
	}

	txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusNotFound {
		t.Fatalf("transaction should be deleted, status %d", txResp.StatusCode)
	}
}

func TestDeleteCreditPaymentRow(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Потреб",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)

	var pendingID string
	for _, item := range credit["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] != true {
			pendingID = row["id"].(string)
			break
		}
	}
	if pendingID == "" {
		t.Fatal("no pending payment")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/credits/"+creditID+"/payments/"+pendingID, nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete payment status %d", delResp.StatusCode)
	}
}

func TestDeleteAppliedScheduledPaymentRestoresRow(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Потреб",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().AddDate(0, -1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)
	beforeLen := len(credit["schedule"].([]any))

	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")
	payBody, _ := json.Marshal(map[string]any{
		"amount":       "10000.00",
		"payment_date": payDate,
	})
	payResp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(payBody))
	if err != nil {
		t.Fatal(err)
	}
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", payResp.StatusCode)
	}
	var paid map[string]any
	_ = json.NewDecoder(payResp.Body).Decode(&paid)
	payResp.Body.Close()

	var paymentID string
	for _, item := range paid["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] == true && row["transaction_id"] != nil {
			paymentID = row["id"].(string)
			break
		}
	}
	if paymentID == "" {
		t.Fatal("no applied scheduled payment")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/credits/"+creditID+"/payments/"+paymentID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete payment status %d", delResp.StatusCode)
	}
	delResp.Body.Close()

	getResp, _ := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	defer getResp.Body.Close()
	var updated map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&updated)
	schedule := updated["schedule"].([]any)
	if len(schedule) != beforeLen {
		t.Fatalf("expected %d schedule rows, got %d", beforeLen, len(schedule))
	}
	if int64(updated["paid_amount"].(float64)) != 0 {
		t.Fatalf("paid_amount should be 0, got %v", updated["paid_amount"])
	}

	var restored map[string]any
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["id"] == paymentID {
			restored = row
			break
		}
	}
	if restored == nil {
		t.Fatal("restored payment row not found")
	}
	if restored["kind"] != "scheduled" {
		t.Fatalf("expected kind scheduled, got %v", restored["kind"])
	}
	if restored["is_applied"] == true {
		t.Fatal("restored payment must be unpaid")
	}
	if restored["transaction_id"] != nil {
		t.Fatal("restored payment must not have transaction_id")
	}
}

func TestFutureCreditPaymentAffectsDashboardForecast(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "WB")

	credit := createCredit(t, env, map[string]any{
		"name":                "Потреб",
		"principal_amount":    "50000.00",
		"issue_date":          time.Now().UTC().Format("2006-01-02 00:00:00"),
		"term_months":         5,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)

	payDate := time.Now().UTC().Add(2 * time.Hour).Format("2006-01-02 15:04:05")
	payBody, _ := json.Marshal(map[string]any{
		"amount":       "5000.00",
		"payment_date": payDate,
	})
	payResp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(payBody))
	if err != nil {
		t.Fatal(err)
	}
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", payResp.StatusCode)
	}
	var paid map[string]any
	_ = json.NewDecoder(payResp.Body).Decode(&paid)
	payResp.Body.Close()
	var txID string
	for _, item := range paid["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] == true && row["transaction_id"] != nil {
			txID = row["transaction_id"].(string)
			break
		}
	}
	if txID == "" {
		t.Fatal("applied scheduled payment with transaction_id not found")
	}
	txResp, err := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer txResp.Body.Close()
	var tx map[string]any
	_ = json.NewDecoder(txResp.Body).Decode(&tx)
	if tx["kind"] != "future" {
		t.Fatalf("expected future tx kind, got %v", tx["kind"])
	}
	if tx["account_id"] != accID {
		t.Fatalf("expected tx account_id %s, got %v", accID, tx["account_id"])
	}
	if int64(tx["amount"].(float64)) != 500000 {
		t.Fatalf("expected tx amount 500000, got %v", tx["amount"])
	}
	txDateRaw, _ := tx["transaction_date"].(string)
	if txDateRaw == "" {
		t.Fatal("transaction_date is empty")
	}
	tz := "Europe/Moscow"
	monthStart, monthEnd, err := timeutil.MonthBoundsUTC(tz, timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	txDate, err := timeutil.ParseUTC(txDateRaw)
	if err != nil {
		t.Fatal(err)
	}
	monthStartTime, err := timeutil.ParseUTC(monthStart)
	if err != nil {
		t.Fatal(err)
	}
	monthEndTime, err := timeutil.ParseUTC(monthEnd)
	if err != nil {
		t.Fatal(err)
	}
	if txDate.Before(monthStartTime) || txDate.After(monthEndTime) {
		t.Fatalf("tx date %s outside month [%s..%s]", txDateRaw, monthStart, monthEnd)
	}
	var affects int
	if err := env.db.QueryRow("SELECT affects_balance FROM transactions WHERE id = ?", txID).Scan(&affects); err != nil {
		t.Fatal(err)
	}
	if affects != 1 {
		t.Fatalf("expected affects_balance=1, got %d", affects)
	}

	dashResp, err := env.authedRequest(http.MethodGet, "/api/v1/dashboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dashResp.Body.Close()
	if dashResp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard status %d", dashResp.StatusCode)
	}
	var dash map[string]any
	_ = json.NewDecoder(dashResp.Body).Decode(&dash)
	accounts, ok := dash["accounts"].([]any)
	if !ok {
		t.Fatalf("dashboard accounts missing: %v", dash["accounts"])
	}
	var target map[string]any
	for _, item := range accounts {
		row := item.(map[string]any)
		if row["id"] == accID {
			target = row
			break
		}
	}
	if target == nil {
		t.Fatalf("account %s not found in dashboard", accID)
	}
	balance := int64(target["balance"].(float64))
	forecast := int64(target["forecast_balance"].(float64))
	if balance == forecast {
		t.Fatalf("expected forecast != balance for account %s, got %d", accID, balance)
	}
	if target["has_future_this_month"] != true {
		t.Fatalf("expected has_future_this_month=true, got %v", target["has_future_this_month"])
	}
}

func TestChangeDebitAccount(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	acc1 := createTestAccount(t, env, "Счёт 1")
	acc2 := createTestAccount(t, env, "Счёт 2")

	credit := createCredit(t, env, map[string]any{
		"name":             "Тест",
		"principal_amount": "30000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      3,
		"interest_rate":    0,
		"debit_account_id": acc1,
	})
	creditID := credit["id"].(string)

	body, _ := json.Marshal(map[string]any{"debit_account_id": acc2})
	resp, err := env.authedRequest(http.MethodPut, "/api/v1/credits/"+creditID, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status %d", resp.StatusCode)
	}
	var updated map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&updated)
	if updated["debit_account_id"] != acc2 {
		t.Fatalf("debit account not updated")
	}
}

func TestDeleteCreditKeepTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":                "Удаляемый",
		"principal_amount":    "20000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         4,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": false,
	})
	creditID := credit["id"].(string)

	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")
	payBody, _ := json.Marshal(map[string]any{"amount": "2000.00", "payment_date": payDate})
	payResp, _ := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(payBody))
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", payResp.StatusCode)
	}
	payResp.Body.Close()

	var txID string
	getResp, _ := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	var detail map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&detail)
	getResp.Body.Close()
	for _, item := range detail["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "scheduled" && row["is_applied"] == true && row["transaction_id"] != nil {
			txID = row["transaction_id"].(string)
			break
		}
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/credits/"+creditID+"?mode=keep_transactions", nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status %d", delResp.StatusCode)
	}

	txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusOK {
		t.Fatal("transaction should remain")
	}
	var tx map[string]any
	_ = json.NewDecoder(txResp.Body).Decode(&tx)
	desc := tx["description"].(string)
	if desc == "" || len(desc) < 10 {
		t.Fatalf("description should contain suffix, got %q", desc)
	}
}

func TestDeleteCreditCascade(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"principal_amount":    "30000.00",
		"issue_date":          time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":         6,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"create_transactions": true,
	})
	creditID := credit["id"].(string)

	payDate := time.Now().UTC().Format("2006-01-02 15:04:05")

	payBody, _ := json.Marshal(map[string]any{
		"amount":       "1000.00",
		"payment_date": payDate,
	})
	payResp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/payments", bytes.NewReader(payBody))
	if err != nil {
		t.Fatal(err)
	}
	if payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay payment status %d", payResp.StatusCode)
	}
	var paid map[string]any
	_ = json.NewDecoder(payResp.Body).Decode(&paid)
	payResp.Body.Close()

	var txID string
	for _, item := range paid["schedule"].([]any) {
		row := item.(map[string]any)
		if row["transaction_id"] != nil {
			txID = row["transaction_id"].(string)
			break
		}
	}
	if txID == "" {
		t.Fatal("expected payment transaction")
	}

	delResp, err := env.authedRequest(http.MethodDelete, "/api/v1/credits/"+creditID+"?mode=cascade", nil)
	if err != nil {
		t.Fatal(err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete cascade status %d", delResp.StatusCode)
	}

	txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusNotFound {
		t.Fatalf("transaction should be deleted, status %d", txResp.StatusCode)
	}

	getResp, _ := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("credit should be deleted, status %d", getResp.StatusCode)
	}
}

func TestRetroactiveCreditNoTransactions(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	balBefore, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	var bal1 struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balBefore.Body).Decode(&bal1)
	balBefore.Body.Close()

	credit := createCredit(t, env, map[string]any{
		"name":                "Старый кредит",
		"principal_amount":    "60000.00",
		"issue_date":          "2022-01-01 00:00:00",
		"term_months":         6,
		"interest_rate":       0,
		"debit_account_id":    accID,
		"added_retroactively": true,
	})
	if credit["added_retroactively"] != true {
		t.Fatal("expected added_retroactively")
	}
	schedule := credit["schedule"].([]any)
	retroCount := 0
	for _, item := range schedule {
		row := item.(map[string]any)
		if row["kind"] == "retroactive" {
			retroCount++
			if row["transaction_id"] != nil {
				t.Fatal("retroactive payment should not have transaction")
			}
			if row["is_applied"] != true {
				t.Fatal("retroactive should be applied")
			}
		}
	}
	if retroCount == 0 {
		t.Fatal("expected retroactive payments")
	}

	balAfter, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	var bal2 struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balAfter.Body).Decode(&bal2)
	balAfter.Body.Close()
	if bal1.Balance != bal2.Balance {
		t.Fatalf("balance changed: before %d after %d", bal1.Balance, bal2.Balance)
	}
}

func TestCompleteWithoutBalanceCreatesTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	balBefore, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	var bal1 struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balBefore.Body).Decode(&bal1)
	balBefore.Body.Close()

	credit := createCredit(t, env, map[string]any{
		"name":             "Завершение без баланса",
		"principal_amount": "10000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      2,
		"interest_rate":    0,
		"debit_account_id": accID,
	})
	creditID := credit["id"].(string)

	closeBody, _ := json.Marshal(map[string]any{
		"affects_balance": false,
		"payment_date":    time.Now().UTC().Format("2006-01-02 15:04:05"),
	})
	closeResp, _ := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/close", bytes.NewReader(closeBody))
	closeResp.Body.Close()
	if closeResp.StatusCode != http.StatusOK {
		t.Fatalf("close status %d", closeResp.StatusCode)
	}

	getResp, _ := env.authedRequest(http.MethodGet, "/api/v1/credits/"+creditID, nil)
	var closed map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&closed)
	getResp.Body.Close()
	if closed["remaining_amount"].(float64) != 0 {
		t.Fatalf("expected remaining 0, got %v", closed["remaining_amount"])
	}
	if closed["paid_amount"].(float64) != closed["principal_amount"].(float64) {
		t.Fatalf("expected fully paid")
	}

	var txID string
	for _, item := range closed["schedule"].([]any) {
		row := item.(map[string]any)
		if row["kind"] == "auto" && row["is_applied"] == true && row["transaction_id"] != nil {
			txID = row["transaction_id"].(string)
			if row["exclude_from_stats"] == true {
				t.Fatal("closure payment should be included in stats")
			}
			break
		}
	}
	if txID == "" {
		t.Fatal("expected closure payment with transaction")
	}

	txResp, _ := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+txID, nil)
	defer txResp.Body.Close()
	if txResp.StatusCode != http.StatusOK {
		t.Fatal("closure transaction should exist")
	}

	balAfter, _ := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+accID+"/balance", nil)
	var bal2 struct {
		Balance int64 `json:"balance"`
	}
	_ = json.NewDecoder(balAfter.Body).Decode(&bal2)
	balAfter.Body.Close()
	if bal1.Balance != bal2.Balance {
		t.Fatalf("balance changed: before %d after %d", bal1.Balance, bal2.Balance)
	}
}

func TestCloseCreditHidesFromActive(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	credit := createCredit(t, env, map[string]any{
		"name":             "Закрываемый",
		"principal_amount": "10000.00",
		"issue_date":       time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months":      2,
		"interest_rate":    0,
		"debit_account_id": accID,
	})
	creditID := credit["id"].(string)

	closeBody, _ := json.Marshal(map[string]any{
		"affects_balance": false,
		"payment_date":    time.Now().UTC().Format("2006-01-02 15:04:05"),
	})
	closeResp, _ := env.authedRequest(http.MethodPost, "/api/v1/credits/"+creditID+"/close", bytes.NewReader(closeBody))
	closeResp.Body.Close()
	if closeResp.StatusCode != http.StatusOK {
		t.Fatalf("close status %d", closeResp.StatusCode)
	}

	listResp, _ := env.authedRequest(http.MethodGet, "/api/v1/credits?status=active", nil)
	defer listResp.Body.Close()
	var active []map[string]any
	_ = json.NewDecoder(listResp.Body).Decode(&active)
	for _, c := range active {
		if c["id"] == creditID {
			t.Fatal("closed credit should not appear in active list")
		}
	}
}

func TestSchedulePreview(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	body, _ := json.Marshal(map[string]any{
		"principal":        "120000.00",
		"term":             12,
		"interest_rate":    0,
		"payment_interval": "month",
		"issue_date":       "2024-01-15 00:00:00",
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits/schedule/preview", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("preview status %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	preview := result["schedule_preview"].([]any)
	if len(preview) != 12 {
		t.Fatalf("expected 12 preview rows, got %d", len(preview))
	}
}

func TestCreateCreditManualSchedule(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	d1 := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 15:04:05")
	d2 := time.Now().UTC().AddDate(0, 2, 0).Format("2006-01-02 15:04:05")
	d3 := time.Now().UTC().AddDate(0, 3, 0).Format("2006-01-02 15:04:05")

	credit := createCredit(t, env, map[string]any{
		"name":             "Ручной график",
		"principal_amount": "30000.00",
		"issue_date":       time.Now().UTC().Format("2006-01-02 00:00:00"),
		"term_months":      3,
		"interest_rate":    0,
		"payment_interval": "manual",
		"debit_account_id": accID,
		"schedule_seed": []map[string]any{
			{"payment_date": d1, "amount": "10000.00"},
			{"payment_date": d2, "amount": "10000.00"},
			{"payment_date": d3, "amount": "10000.00"},
		},
	})
	if credit["payment_interval"] != "manual" {
		t.Fatalf("expected manual interval, got %v", credit["payment_interval"])
	}
	schedule := credit["schedule"].([]any)
	if len(schedule) != 3 {
		t.Fatalf("expected 3 payments, got %d", len(schedule))
	}
}

func TestCreateCreditRejectsNegativeScheduleSeedAmount(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	accID := createTestAccount(t, env, "Счёт")

	d1 := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 15:04:05")
	d2 := time.Now().UTC().AddDate(0, 2, 0).Format("2006-01-02 15:04:05")
	d3 := time.Now().UTC().AddDate(0, 3, 0).Format("2006-01-02 15:04:05")

	body, _ := json.Marshal(map[string]any{
		"name":             "Ручной график",
		"principal_amount": "30000.00",
		"issue_date":       time.Now().UTC().Format("2006-01-02 00:00:00"),
		"term_months":      3,
		"interest_rate":    0,
		"payment_interval": "manual",
		"debit_account_id": accID,
		"schedule_seed": []map[string]any{
			{"payment_date": d1, "amount": "10000.00"},
			{"payment_date": d2, "amount": "-10000.00"},
			{"payment_date": d3, "amount": "10000.00"},
		},
	})

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/credits", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 400 for negative schedule amount, got %d: %v", resp.StatusCode, errBody)
	}
}

func TestCreditSystemCategoryExists(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/categories", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cats []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&cats)
	found := false
	for _, c := range cats {
		if c["name"] == "Кредиты" && c["type"] == "expense" && c["is_system"] == true {
			found = true
		}
	}
	if !found {
		t.Fatal("system category Кредиты not found")
	}
}

func TestCreditApplyDueUsesDebitTimeForTransaction(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	ctx := context.Background()
	accID := createTestAccount(t, env, "Автосписание кредита")

	issueDate := timeutil.FormatUTC(timeutil.NowUTC().AddDate(0, -1, 0))
	creditRow := createCredit(t, env, map[string]any{
		"principal_amount": "60000.00",
		"issue_date":       issueDate,
		"term_months":      6,
		"interest_rate":    0,
		"payment_interval": "month",
		"debit_account_id": accID,
		"debit_time_local": "10:00",
	})
	creditID := creditRow["id"].(string)

	_, err := env.db.ExecContext(ctx, `
		UPDATE credit_payments SET payment_date = datetime('now', '-1 day')
		WHERE id = (
			SELECT id FROM credit_payments
			WHERE credit_id = ? AND is_applied = 0 AND kind = 'scheduled' LIMIT 1
		)`, creditID)
	if err != nil {
		t.Fatal(err)
	}

	cutoff, err := credit.TodayCutoffUTC("Europe/Moscow", timeutil.NowUTC())
	if err != nil {
		t.Fatal(err)
	}
	applied, err := credit.ApplyDuePayments(ctx, env.db, mustAdminUserID(t, env), cutoff, "10:00")
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("expected 1 applied payment, got %d", applied)
	}

	var txDate string
	err = env.db.QueryRowContext(ctx, `
		SELECT t.transaction_date FROM transactions t
		JOIN credit_payments cp ON cp.transaction_id = t.id
		WHERE cp.credit_id = ? AND cp.kind = 'auto' LIMIT 1`, creditID).Scan(&txDate)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := timeutil.ParseUTC(txDate)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	if parsed.In(loc).Format("15:04") != "10:00" {
		t.Fatalf("expected transaction at 10:00 Europe/Moscow, got %s", txDate)
	}
}
