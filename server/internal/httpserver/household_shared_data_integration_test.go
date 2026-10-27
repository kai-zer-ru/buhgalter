package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestHouseholdBudgetCreditShareAndStatsABC(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhmemdata", "password1")
	createHouseholdUser(t, env, "hhoutdata", "password1")

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("household %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	resp.Body.Close()

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	var invite struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&invite)
	resp.Body.Close()

	env.login(t, "hhmemdata", "password1")
	invBody, _ := json.Marshal(map[string]string{"code": invite.Code})
	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if acc.StatusCode != http.StatusOK {
		t.Fatalf("accept %d %s", acc.StatusCode, decodeErrCode(t, acc))
	}
	acc.Body.Close()
	memberCookie := env.cookie

	env.cookie = adminCookie
	sharedBody, _ := json.Marshal(map[string]any{"name": "Семейный бюджет-счёт", "type": "cash", "shared": true, "initial_balance": "500.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(sharedBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shared create %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var shared struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&shared)
	resp.Body.Close()

	personalBody, _ := json.Marshal(map[string]any{"name": "Личный share-счёт", "type": "cash", "initial_balance": "10.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(personalBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("personal %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var personal struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&personal)
	resp.Body.Close()

	budgetBody, _ := json.Marshal(map[string]any{
		"name": "Семейные продукты", "scope": "all_expense", "amount": "1000.00",
		"account_id": shared.ID, "is_active": true,
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/budgets", bytes.NewReader(budgetBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("budget %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var budget struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&budget)
	resp.Body.Close()

	creditBody, _ := json.Marshal(map[string]any{
		"name": "Семейный кредит", "principal_amount": "10000.00",
		"issue_date":  time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02 00:00:00"),
		"term_months": 6, "interest_rate": 0, "payment_interval": "month",
		"debit_account_id": shared.ID, "added_retroactively": false,
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/credits", bytes.NewReader(creditBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("credit %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var cr struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&cr)
	resp.Body.Close()
	if cr.Scope != "household" {
		t.Fatalf("credit scope %q", cr.Scope)
	}

	env.cookie = memberCookie
	listB, err := env.authedRequest(http.MethodGet, "/api/v1/budgets", nil)
	if err != nil {
		t.Fatal(err)
	}
	var budgets []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(listB.Body).Decode(&budgets)
	listB.Body.Close()
	foundBudget := false
	for _, b := range budgets {
		if b.ID == budget.ID {
			foundBudget = true
		}
	}
	if !foundBudget {
		t.Fatalf("member missing household budget: %#v", budgets)
	}

	listC, err := env.authedRequest(http.MethodGet, "/api/v1/credits", nil)
	if err != nil {
		t.Fatal(err)
	}
	var credits []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(listC.Body).Decode(&credits)
	listC.Body.Close()
	foundCredit := false
	for _, c := range credits {
		if c.ID == cr.ID && c.Scope == "household" {
			foundCredit = true
		}
	}
	if !foundCredit {
		t.Fatalf("member missing household credit: %#v", credits)
	}

	dashHH, err := env.authedRequest(http.MethodGet, "/api/v1/dashboard?scope=household", nil)
	if err != nil {
		t.Fatal(err)
	}
	if dashHH.StatusCode != http.StatusOK {
		t.Fatalf("dashboard household %d %s", dashHH.StatusCode, decodeErrCode(t, dashHH))
	}
	dashHH.Body.Close()

	catID := getExpenseCategoryScoped(t, env, "household")
	txBody, _ := json.Marshal(map[string]any{
		"account_id": shared.ID, "type": "expense", "amount": "12.00",
		"category_id": catID, "transaction_date": time.Now().UTC().Format("2006-01-02 15:04:05"),
	})
	env.cookie = adminCookie
	txResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	if err != nil {
		t.Fatal(err)
	}
	if txResp.StatusCode != http.StatusCreated {
		t.Fatalf("household tx %d %s", txResp.StatusCode, decodeErrCode(t, txResp))
	}
	txResp.Body.Close()
	env.cookie = memberCookie
	listHH, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?scope=household", nil)
	if err != nil {
		t.Fatal(err)
	}
	if listHH.StatusCode != http.StatusOK {
		t.Fatalf("list household %d %s", listHH.StatusCode, decodeErrCode(t, listHH))
	}
	var hhPage struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(listHH.Body).Decode(&hhPage)
	listHH.Body.Close()
	if hhPage.Meta.Total < 1 {
		t.Fatalf("household list total %d", hhPage.Meta.Total)
	}
	metaResp, err := env.authedRequest(http.MethodGet, "/api/v1/ui/meta", nil)
	if err != nil {
		t.Fatal(err)
	}
	if metaResp.StatusCode != http.StatusOK {
		t.Fatalf("ui/meta %d %s", metaResp.StatusCode, decodeErrCode(t, metaResp))
	}
	var metaBody struct {
		Accounts []struct {
			ID    string `json:"id"`
			Scope string `json:"scope"`
		} `json:"accounts"`
	}
	_ = json.NewDecoder(metaResp.Body).Decode(&metaBody)
	metaResp.Body.Close()
	foundShared := false
	for _, acc := range metaBody.Accounts {
		if acc.ID == shared.ID {
			foundShared = true
			if acc.Scope != "household" {
				t.Fatalf("ui/meta shared scope %q", acc.Scope)
			}
		}
	}
	if !foundShared {
		t.Fatal("ui/meta missing household account for member")
	}
	listP, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?scope=personal", nil)
	if err != nil {
		t.Fatal(err)
	}
	if listP.StatusCode != http.StatusOK {
		t.Fatalf("list personal %d %s", listP.StatusCode, decodeErrCode(t, listP))
	}
	var pPage struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.NewDecoder(listP.Body).Decode(&pPage)
	listP.Body.Close()
	if pPage.Meta.Total != 0 {
		t.Fatalf("personal list leaked household txs: %d", pPage.Meta.Total)
	}

	statsHH, err := env.authedRequest(http.MethodGet, "/api/v1/stats/summary?scope=household", nil)
	if err != nil {
		t.Fatal(err)
	}
	if statsHH.StatusCode != http.StatusOK {
		t.Fatalf("stats household %d %s", statsHH.StatusCode, decodeErrCode(t, statsHH))
	}
	statsHH.Body.Close()

	env.cookie = adminCookie
	share, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+personal.ID+"/share", nil)
	if err != nil {
		t.Fatal(err)
	}
	if share.StatusCode != http.StatusOK {
		t.Fatalf("share %d %s", share.StatusCode, decodeErrCode(t, share))
	}
	var sharedOut struct {
		ID      string `json:"id"`
		Scope   string `json:"scope"`
		Details struct {
			Moved struct {
				Transactions int64 `json:"transactions"`
			} `json:"moved"`
		} `json:"details"`
	}
	_ = json.NewDecoder(share.Body).Decode(&sharedOut)
	share.Body.Close()
	if sharedOut.Scope != "household" {
		t.Fatalf("shared scope %q", sharedOut.Scope)
	}

	env.cookie = memberCookie
	got, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+personal.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusOK {
		t.Fatalf("member get shared personal %d %s", got.StatusCode, decodeErrCode(t, got))
	}
	got.Body.Close()

	env.login(t, "hhoutdata", "password1")
	outGet, err := env.authedRequest(http.MethodGet, "/api/v1/credits/"+cr.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outGet.StatusCode != http.StatusNotFound {
		t.Fatalf("outsider credit %d", outGet.StatusCode)
	}
	outGet.Body.Close()
}
