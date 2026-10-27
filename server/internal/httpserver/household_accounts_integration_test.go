package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHouseholdSharedAccountVisibilityABC(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhmem2", "password1")
	createHouseholdUser(t, env, "hhout2", "password1")

	body, _ := json.Marshal(map[string]any{"name": "Наличные A", "type": "cash", "initial_balance": "100.00"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("personal create %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var personal struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&personal)
	resp.Body.Close()
	if personal.Scope != "personal" {
		t.Fatalf("solo/personal scope %q", personal.Scope)
	}

	sharedBody, _ := json.Marshal(map[string]any{"name": "Общая карта", "type": "cash", "shared": true, "initial_balance": "0.00"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(sharedBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict || decodeErrCode(t, resp) != "ERR_HOUSEHOLD_NOT_MEMBER" {
		t.Fatalf("shared without household %d", resp.StatusCode)
	}
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

	env.login(t, "hhmem2", "password1")
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
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/accounts", bytes.NewReader(sharedBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shared create %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var shared struct {
		ID            string `json:"id"`
		Scope         string `json:"scope"`
		IsPrimary     bool   `json:"is_primary"`
		OriginalOwner *struct {
			UserID string `json:"user_id"`
		} `json:"original_owner"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&shared)
	resp.Body.Close()
	if shared.Scope != "household" || shared.IsPrimary || shared.OriginalOwner != nil {
		t.Fatalf("shared payload %#v", shared)
	}

	prim, err := env.authedRequest(http.MethodPost, "/api/v1/accounts/"+shared.ID+"/primary", nil)
	if err != nil {
		t.Fatal(err)
	}
	if prim.StatusCode != http.StatusBadRequest || decodeErrCode(t, prim) != "ERR_ACCOUNT_SHARED_PRIMARY" {
		t.Fatalf("primary %d", prim.StatusCode)
	}
	prim.Body.Close()

	list, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	var adminList []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(list.Body).Decode(&adminList)
	list.Body.Close()
	if !accountListHas(adminList, personal.ID, "personal") || !accountListHas(adminList, shared.ID, "household") {
		t.Fatalf("admin list %#v", adminList)
	}

	env.cookie = memberCookie
	list, err = env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	var memberList []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(list.Body).Decode(&memberList)
	list.Body.Close()
	if accountListHas(memberList, personal.ID, "personal") {
		t.Fatalf("member saw A's personal: %#v", memberList)
	}
	if !accountListHas(memberList, shared.ID, "household") {
		t.Fatalf("member missing shared: %#v", memberList)
	}
	got, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+shared.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != http.StatusOK {
		t.Fatalf("member get shared %d", got.StatusCode)
	}
	got.Body.Close()
	bal, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+shared.ID+"/balance", nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.StatusCode != http.StatusOK {
		t.Fatalf("member shared balance %d %s", bal.StatusCode, decodeErrCode(t, bal))
	}
	bal.Body.Close()

	env.cookie = adminCookie
	bal, err = env.authedRequest(http.MethodGet, "/api/v1/accounts/"+shared.ID+"/balance", nil)
	if err != nil {
		t.Fatal(err)
	}
	if bal.StatusCode != http.StatusOK {
		t.Fatalf("admin shared balance %d %s", bal.StatusCode, decodeErrCode(t, bal))
	}
	bal.Body.Close()

	env.cookie = memberCookie
	hidden, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+personal.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if hidden.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(hidden.Body)
		t.Fatalf("member get personal %d %s", hidden.StatusCode, b)
	}
	hidden.Body.Close()

	env.login(t, "hhout2", "password1")
	outGet, err := env.authedRequest(http.MethodGet, "/api/v1/accounts/"+shared.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outGet.StatusCode != http.StatusNotFound {
		t.Fatalf("outsider get %d", outGet.StatusCode)
	}
	outGet.Body.Close()
	outList, err := env.authedRequest(http.MethodGet, "/api/v1/accounts", nil)
	if err != nil {
		t.Fatal(err)
	}
	var outsiderList []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(outList.Body).Decode(&outsiderList)
	outList.Body.Close()
	for _, a := range outsiderList {
		if a.ID == shared.ID {
			t.Fatal("outsider listed household account")
		}
	}
}

func accountListHas(list []struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
}, id, scope string) bool {
	for _, a := range list {
		if a.ID == id && a.Scope == scope {
			return true
		}
	}
	return false
}
