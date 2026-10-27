package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestHouseholdTransferMaskAndScopeMismatch(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhmemtx", "password1")

	personalBody, _ := json.Marshal(map[string]any{"name": "Личный A", "type": "cash", "initial_balance": "500.00"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(personalBody))
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

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
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

	env.login(t, "hhmemtx", "password1")
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
	sharedBody, _ := json.Marshal(map[string]any{"name": "Общий", "type": "cash", "shared": true, "initial_balance": "0.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(sharedBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shared %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var shared struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&shared)
	resp.Body.Close()

	personalCat := getExpenseCategoryScoped(t, env, "personal")
	mismatch, _ := json.Marshal(map[string]any{
		"account_id": shared.ID, "type": "expense", "amount": "10.00",
		"category_id": personalCat, "transaction_date": "2026-01-10 11:00:00",
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(mismatch))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest || decodeErrCode(t, resp) != "ERR_SCOPE_MISMATCH" {
		t.Fatalf("mismatch %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}

	okTx, _ := json.Marshal(map[string]any{
		"account_id": shared.ID, "type": "expense", "amount": "5.00",
		"transaction_date": "2026-01-10 11:00:00",
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(okTx))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shared expense %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var created struct {
		Scope     string `json:"scope"`
		CreatedBy *struct {
			UserID string `json:"user_id"`
		} `json:"created_by"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created.Scope != "household" || created.CreatedBy == nil {
		t.Fatalf("created payload %#v", created)
	}

	trBody, _ := json.Marshal(map[string]any{
		"from_account_id": personal.ID, "to_account_id": shared.ID,
		"amount": "20.00", "transaction_date": "2026-01-10 12:00:00",
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(trBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("transfer %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var tr struct {
		GroupID string `json:"group_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&tr)
	resp.Body.Close()

	env.cookie = memberCookie
	list, err := env.authedRequest(http.MethodGet, "/api/v1/transactions?account_id="+shared.ID+"&type=transfer", nil)
	if err != nil {
		t.Fatal(err)
	}
	var page struct {
		Data []struct {
			ID                    string  `json:"id"`
			TransferAccountID     *string `json:"transfer_account_id"`
			TransferAccountMasked bool    `json:"transfer_account_masked"`
			TransferAccountLabel  string  `json:"transfer_account_label"`
			TransferGroupID       *string `json:"transfer_group_id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(list.Body).Decode(&page)
	list.Body.Close()
	if len(page.Data) == 0 {
		t.Fatal("member saw no transfer leg")
	}
	leg := page.Data[0]
	if !leg.TransferAccountMasked || leg.TransferAccountID != nil {
		t.Fatalf("mask %#v", leg)
	}
	if leg.TransferAccountLabel == "" {
		t.Fatal("empty mask label")
	}

	groupID := tr.GroupID
	if leg.TransferGroupID != nil && *leg.TransferGroupID != "" {
		groupID = *leg.TransferGroupID
	}
	upd, _ := json.Marshal(map[string]any{
		"from_account_id": personal.ID, "to_account_id": shared.ID,
		"amount": "25.00", "transaction_date": "2026-01-10 12:00:00",
	})
	put, err := env.authedRequest(http.MethodPut, "/api/v1/transfers/"+groupID, bytes.NewReader(upd))
	if err != nil {
		t.Fatal(err)
	}
	if put.StatusCode != http.StatusForbidden || decodeErrCode(t, put) != "ERR_TRANSFER_FOREIGN_ACCOUNT" {
		t.Fatalf("member update %d", put.StatusCode)
	}

	del, err := env.authedRequest(http.MethodDelete, "/api/v1/transfers/"+groupID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if del.StatusCode != http.StatusForbidden || decodeErrCode(t, del) != "ERR_TRANSFER_FOREIGN_ACCOUNT" {
		t.Fatalf("member delete %d", del.StatusCode)
	}
	del.Body.Close()
}
