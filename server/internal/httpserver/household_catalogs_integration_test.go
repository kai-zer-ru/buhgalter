package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestHouseholdCatalogsVisibilityABC(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "hhmemcat", "password1")
	createHouseholdUser(t, env, "hhoutcat", "password1")

	tagHH, _ := json.Marshal(map[string]string{"name": "hh-shared-tag", "scope": "household"})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader(tagHH))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict || decodeErrCode(t, resp) != "ERR_HOUSEHOLD_NOT_MEMBER" {
		t.Fatalf("household tag without group %d %s", resp.StatusCode, decodeErrCode(t, resp))
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

	env.login(t, "hhmemcat", "password1")
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
	personalTag, _ := json.Marshal(map[string]string{"name": "a-personal-tag"})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader(personalTag))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("personal tag %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var aTag struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&aTag)
	resp.Body.Close()
	if aTag.Scope != "personal" {
		t.Fatalf("personal tag scope %q", aTag.Scope)
	}

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/tags", bytes.NewReader(tagHH))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("household tag %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var hhTag struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&hhTag)
	resp.Body.Close()
	if hhTag.Scope != "household" {
		t.Fatalf("household tag scope %q", hhTag.Scope)
	}

	catBody, _ := json.Marshal(map[string]any{
		"name": "HH Extra Cat", "type": "expense", "icon": "default", "scope": "household",
	})
	resp, err = env.authedRequest(http.MethodPost, "/api/v1/categories", bytes.NewReader(catBody))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("household cat %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var hhCat struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&hhCat)
	resp.Body.Close()
	if hhCat.Scope != "household" {
		t.Fatalf("household cat scope %q", hhCat.Scope)
	}

	env.cookie = memberCookie
	list, err := env.authedRequest(http.MethodGet, "/api/v1/tags", nil)
	if err != nil {
		t.Fatal(err)
	}
	if list.StatusCode != http.StatusOK {
		t.Fatalf("member tags %d %s", list.StatusCode, decodeErrCode(t, list))
	}
	var memberTags []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(list.Body).Decode(&memberTags)
	list.Body.Close()
	if catalogHas(memberTags, aTag.ID, "personal") {
		t.Fatalf("member saw A's personal tag: %#v", memberTags)
	}
	if !catalogHas(memberTags, hhTag.ID, "household") {
		t.Fatalf("member missing household tag: %#v", memberTags)
	}

	cats, err := env.authedRequest(http.MethodGet, "/api/v1/categories?scope=household", nil)
	if err != nil {
		t.Fatal(err)
	}
	var hhCats []struct {
		ID    string `json:"id"`
		Scope string `json:"scope"`
	}
	_ = json.NewDecoder(cats.Body).Decode(&hhCats)
	cats.Body.Close()
	if !catalogHas(hhCats, hhCat.ID, "household") {
		t.Fatalf("member missing household category: %#v", hhCats)
	}
	for _, c := range hhCats {
		if c.Scope != "household" {
			t.Fatalf("scope=household leaked %q: %#v", c.Scope, c)
		}
	}

	env.login(t, "hhoutcat", "password1")
	outList, err := env.authedRequest(http.MethodGet, "/api/v1/tags", nil)
	if err != nil {
		t.Fatal(err)
	}
	var outsiderTags []struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(outList.Body).Decode(&outsiderTags)
	outList.Body.Close()
	for _, tag := range outsiderTags {
		if tag.ID == hhTag.ID || tag.ID == aTag.ID {
			t.Fatal("outsider listed household or A's personal tag")
		}
	}
}

func catalogHas(list []struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
}, id, scope string) bool {
	for _, item := range list {
		if item.ID == id && item.Scope == scope {
			return true
		}
	}
	return false
}
