package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHouseholdUnshareAndDissolveABC(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhmemexit", "password1")

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

	env.login(t, "hhmemexit", "password1")
	invBody, _ := json.Marshal(map[string]string{"code": invite.Code})
	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if acc.StatusCode != http.StatusOK {
		t.Fatalf("accept %d", acc.StatusCode)
	}
	acc.Body.Close()
	memberCookie := env.cookie

	env.cookie = adminCookie
	createdHH, _ := json.Marshal(map[string]any{"name": "Общая карта", "type": "cash", "shared": true, "initial_balance": "0.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(createdHH))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shared %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var familyAcc struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&familyAcc)
	resp.Body.Close()

	unshareNone, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+familyAcc.ID+"/unshare", nil)
	if err != nil {
		t.Fatal(err)
	}
	if unshareNone.StatusCode != http.StatusConflict || decodeErrCode(t, unshareNone) != "ERR_UNSHARE_NO_ORIGINAL_OWNER" {
		t.Fatalf("unshare no original %d %s", unshareNone.StatusCode, decodeErrCode(t, unshareNone))
	}

	env.cookie = memberCookie
	personalBody, _ := json.Marshal(map[string]any{"name": "Кошелёк B", "type": "cash", "initial_balance": "10.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(personalBody))
	if err != nil {
		t.Fatal(err)
	}
	var walletB struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&walletB)
	resp.Body.Close()
	share, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+walletB.ID+"/share", nil)
	if err != nil {
		t.Fatal(err)
	}
	if share.StatusCode != http.StatusOK {
		t.Fatalf("share B %d %s", share.StatusCode, decodeErrCode(t, share))
	}
	var shareOut struct {
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(share.Body).Decode(&shareOut)
	share.Body.Close()
	if shareOut.Scope != "household" {
		t.Fatalf("share B scope %q", shareOut.Scope)
	}

	memberUnshare, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+walletB.ID+"/unshare", nil)
	if err != nil {
		t.Fatal(err)
	}
	if memberUnshare.StatusCode != http.StatusForbidden || decodeErrCode(t, memberUnshare) != "ERR_HOUSEHOLD_OWNER_ONLY" {
		t.Fatalf("member unshare %d %s", memberUnshare.StatusCode, decodeErrCode(t, memberUnshare))
	}

	env.login(t, "admin", "secret123")
	unshare, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+walletB.ID+"/unshare", nil)
	if err != nil {
		t.Fatal(err)
	}
	if unshare.StatusCode != http.StatusOK {
		t.Fatalf("unshare %d %s", unshare.StatusCode, decodeErrCode(t, unshare))
	}
	var back struct {
		Scope         string `json:"scope"`
		OriginalOwner *struct {
			UserID string `json:"user_id"`
		} `json:"original_owner"`
	}
	_ = json.NewDecoder(unshare.Body).Decode(&back)
	unshare.Body.Close()
	if back.Scope != "personal" || back.OriginalOwner != nil {
		t.Fatalf("unshare payload %#v", back)
	}

	env.cookie = memberCookie
	share2, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+walletB.ID+"/share", nil)
	if err != nil {
		t.Fatal(err)
	}
	if share2.StatusCode != http.StatusOK {
		t.Fatalf("re-share %d %s", share2.StatusCode, decodeErrCode(t, share2))
	}
	share2.Body.Close()
	env.login(t, "admin", "secret123")

	noConfirm, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/dissolve", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if noConfirm.StatusCode != http.StatusConflict || decodeErrCode(t, noConfirm) != "ERR_HOUSEHOLD_HAS_MEMBERS" {
		t.Fatalf("dissolve no confirm %d %s", noConfirm.StatusCode, decodeErrCode(t, noConfirm))
	}

	body, _ := json.Marshal(map[string]any{"confirm_members": true})
	diss, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/dissolve", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if diss.StatusCode != http.StatusNoContent {
		t.Fatalf("dissolve %d %s", diss.StatusCode, decodeErrCode(t, diss))
	}

	env.cookie = memberCookie
	gotB, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+walletB.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.StatusCode != http.StatusOK {
		t.Fatalf("B wallet after dissolve %d", gotB.StatusCode)
	}
	gotB.Body.Close()

	env.cookie = adminCookie
	gotFam, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+familyAcc.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotFam.StatusCode != http.StatusOK {
		t.Fatalf("A family acc after dissolve %d", gotFam.StatusCode)
	}
	var fam struct {
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(gotFam.Body).Decode(&fam)
	gotFam.Body.Close()
	if fam.Scope != "personal" {
		t.Fatalf("family acc scope %q", fam.Scope)
	}

	env.cookie = memberCookie
	miss, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+familyAcc.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("B still sees family acc %d", miss.StatusCode)
	}
	miss.Body.Close()
}

func TestHouseholdLeaveIsolationAndAdminOwnerDelete(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhleavemem", "password1")

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
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

	env.login(t, "hhleavemem", "password1")
	invBody, _ := json.Marshal(map[string]string{"code": invite.Code})
	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	acc.Body.Close()
	memberCookie := env.cookie

	env.cookie = memberCookie
	personalBody, _ := json.Marshal(map[string]any{"name": "Кошелёк leave", "type": "cash", "initial_balance": "5.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(personalBody))
	if err != nil {
		t.Fatal(err)
	}
	var wallet struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&wallet)
	resp.Body.Close()
	share, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+wallet.ID+"/share", nil)
	if err != nil {
		t.Fatal(err)
	}
	if share.StatusCode != http.StatusOK {
		t.Fatalf("share %d %s", share.StatusCode, decodeErrCode(t, share))
	}
	share.Body.Close()

	env.cookie = adminCookie
	sharedBody, _ := json.Marshal(map[string]any{"name": "Семейный leave", "type": "cash", "shared": true, "initial_balance": "20.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(sharedBody))
	if err != nil {
		t.Fatal(err)
	}
	var family struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&family)
	resp.Body.Close()

	txBody, _ := json.Marshal(map[string]any{
		"account_id": family.ID, "type": "expense", "amount": "1.00",
		"transaction_date": "2026-01-10 11:00:00",
	})
	env.cookie = memberCookie
	txResp, err := env.authedRequest(http.MethodPost, "/api/v1/transactions", bytes.NewReader(txBody))
	if err != nil {
		t.Fatal(err)
	}
	if txResp.StatusCode != http.StatusCreated {
		t.Fatalf("tx %d %s", txResp.StatusCode, decodeErrCode(t, txResp))
	}
	var createdTx struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(txResp.Body).Decode(&createdTx)
	txResp.Body.Close()

	leave, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/leave", nil)
	if err != nil {
		t.Fatal(err)
	}
	if leave.StatusCode != http.StatusNoContent {
		t.Fatalf("leave %d %s", leave.StatusCode, decodeErrCode(t, leave))
	}

	for _, path := range []string{
		"/api/v1/accounts/" + family.ID,
		"/api/v1/accounts/" + wallet.ID,
		"/api/v1/transactions/" + createdTx.ID,
	} {
		got, err := env.authedRequest(http.MethodGet, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.StatusCode != http.StatusNotFound {
			t.Fatalf("after leave %s → %d", path, got.StatusCode)
		}
		got.Body.Close()
	}

	for _, path := range []string{
		"/api/v1/accounts",
		"/api/v1/transactions",
		"/api/v1/budgets",
		"/api/v1/credits",
		"/api/v1/categories",
		"/api/v1/tags",
		"/api/v1/merchants",
		"/api/v1/debtors",
		"/api/v1/subscriptions",
		"/api/v1/recurring-operations",
		"/api/v1/transaction-templates",
		"/api/v1/category-rules",
		"/api/v1/debts",
		"/api/v1/sync/transaction-changes?since_id=0",
		"/api/v1/dashboard?scope=household",
		"/api/v1/stats/summary?scope=household",
	} {
		got, err := env.authedRequest(http.MethodGet, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(got.Body)
		got.Body.Close()
		if got.StatusCode != http.StatusOK {
			t.Fatalf("%s %d %s", path, got.StatusCode, b)
		}
		if strings.Contains(string(b), family.ID) || strings.Contains(string(b), wallet.ID) || strings.Contains(string(b), createdTx.ID) {
			t.Fatalf("%s still contains household ids: %s", path, b)
		}
	}

	env.cookie = adminCookie
	gotFam, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+wallet.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotFam.StatusCode != http.StatusOK {
		t.Fatalf("A missing B wallet %d", gotFam.StatusCode)
	}
	gotFam.Body.Close()
	gotTx, err := env.authedRequest(http.MethodGet, "/api/v1/transactions/"+createdTx.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotTx.StatusCode != http.StatusOK {
		t.Fatalf("A missing B tx %d", gotTx.StatusCode)
	}
	gotTx.Body.Close()

}

func TestAdminCannotDeleteHouseholdOwnerWithMembers(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	createHouseholdUser(t, env, "hhownerd", "password1")
	createHouseholdUser(t, env, "hhmemd", "password1")

	env.login(t, "hhownerd", "password1")
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("household %d", resp.StatusCode)
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

	env.login(t, "hhmemd", "password1")
	invBody, _ := json.Marshal(map[string]string{"code": invite.Code})
	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	acc.Body.Close()

	env.login(t, "admin", "secret123")
	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	var users []struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	}
	_ = json.NewDecoder(usersResp.Body).Decode(&users)
	usersResp.Body.Close()
	var ownerID, memID string
	for _, u := range users {
		switch u.Login {
		case "hhownerd":
			ownerID = u.ID
		case "hhmemd":
			memID = u.ID
		}
	}
	if ownerID == "" {
		t.Fatal("no owner id")
	}
	delOwner, err := env.authedRequest(http.MethodDelete, "/api/v1/admin/users/"+ownerID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if delOwner.StatusCode != http.StatusConflict || decodeErrCode(t, delOwner) != "ERR_HOUSEHOLD_OWNER_DELETE" {
		t.Fatalf("delete owner %d %s", delOwner.StatusCode, decodeErrCode(t, delOwner))
	}

	delMem, err := env.authedRequest(http.MethodDelete, "/api/v1/admin/users/"+memID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if delMem.StatusCode != http.StatusNoContent {
		t.Fatalf("delete member %d %s", delMem.StatusCode, decodeErrCode(t, delMem))
	}
}
