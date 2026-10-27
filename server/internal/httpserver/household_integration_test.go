package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

func createHouseholdUser(t *testing.T, env *testEnv, login, password string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"login": login, "password": password, "password_confirm": password,
		"display_name": login, "is_admin": false,
	})
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create user %s: %d %s", login, resp.StatusCode, b)
	}
}

func decodeErrCode(t *testing.T, resp *http.Response) string {
	t.Helper()
	var wrap struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	b, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(b, &wrap)
	return wrap.Error.Code
}

func TestHouseholdCreateInviteAcceptKickDissolve(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "member1", "password1")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/households/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me empty status %d", resp.StatusCode)
	}
	var empty struct {
		Household any `json:"household"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&empty)
	if empty.Household != nil {
		t.Fatalf("expected null household, got %#v", empty.Household)
	}

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var created struct {
		Household *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"household"`
		Role    string `json:"role"`
		Members []struct {
			Login string `json:"login"`
			Role  string `json:"role"`
		} `json:"members"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created.Household == nil || created.Role != "owner" || len(created.Members) != 1 {
		t.Fatalf("create payload %#v", created)
	}
	if created.Household.Kind != "family" {
		t.Fatalf("default kind %q", created.Household.Kind)
	}

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusConflict || decodeErrCode(t, resp) != "ERR_HOUSEHOLD_ALREADY_MEMBER" {
		t.Fatalf("repeat create %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	resp.Body.Close()

	resp, err = env.authedRequest(http.MethodPost, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invite %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var invite struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		URL  string `json:"url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&invite)
	resp.Body.Close()
	if len(invite.Code) != 11 || invite.URL == "" {
		t.Fatalf("invite %#v", invite)
	}

	var principalLogin string
	if err := env.db.QueryRow(`SELECT login FROM users WHERE kind = 'household'`).Scan(&principalLogin); err != nil {
		t.Fatal(err)
	}
	loginBody, _ := json.Marshal(map[string]string{"login": principalLogin, "password": "secret123"})
	loginResp, err := http.Post(env.server.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	if loginResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("principal login %d", loginResp.StatusCode)
	}
	loginResp.Body.Close()

	env.cookie = adminCookie
	usersResp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	var users []struct {
		Login string `json:"login"`
	}
	_ = json.NewDecoder(usersResp.Body).Decode(&users)
	usersResp.Body.Close()
	for _, u := range users {
		if u.Login == principalLogin {
			t.Fatal("principal visible in ListUsers")
		}
	}

	env.login(t, "member1", "password1")
	invBody, _ := json.Marshal(map[string]string{"code": invite.Code})
	prev, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/preview", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if prev.StatusCode != http.StatusOK {
		t.Fatalf("preview %d %s", prev.StatusCode, decodeErrCode(t, prev))
	}
	prev.Body.Close()

	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if acc.StatusCode != http.StatusOK {
		t.Fatalf("accept %d %s", acc.StatusCode, decodeErrCode(t, acc))
	}
	acc.Body.Close()

	acc2, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if acc2.StatusCode != http.StatusConflict || decodeErrCode(t, acc2) != "ERR_HOUSEHOLD_ALREADY_MEMBER" {
		t.Fatalf("reuse accept as member %d %s", acc2.StatusCode, decodeErrCode(t, acc2))
	}
	acc2.Body.Close()

	env.cookie = adminCookie
	createHouseholdUser(t, env, "outsider1", "password1")
	env.login(t, "outsider1", "password1")
	acc3, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(invBody))
	if err != nil {
		t.Fatal(err)
	}
	if acc3.StatusCode != http.StatusBadRequest || decodeErrCode(t, acc3) != "ERR_INVITE_INVALID" {
		t.Fatalf("used code by other %d %s", acc3.StatusCode, decodeErrCode(t, acc3))
	}
	acc3.Body.Close()
	env.login(t, "member1", "password1")

	inv2, _ := json.Marshal(map[string]any{
		"login": "member1", "password": "x", "password_confirm": "x",
	})
	_ = inv2
	listInv, err := env.authedRequest(http.MethodGet, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	if listInv.StatusCode != http.StatusForbidden || decodeErrCode(t, listInv) != "ERR_HOUSEHOLD_OWNER_ONLY" {
		t.Fatalf("member invites %d %s", listInv.StatusCode, decodeErrCode(t, listInv))
	}
	listInv.Body.Close()

	me, err := env.authedRequest(http.MethodGet, "/api/v1/households/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	var meBody struct {
		Members []struct {
			UserID string `json:"user_id"`
			Login  string `json:"login"`
		} `json:"members"`
	}
	_ = json.NewDecoder(me.Body).Decode(&meBody)
	me.Body.Close()
	var memberID string
	for _, m := range meBody.Members {
		if m.Login == "member1" {
			memberID = m.UserID
		}
	}
	if memberID == "" {
		t.Fatal("member id missing")
	}

	env.cookie = adminCookie
	kick, err := env.authedRequest(http.MethodDelete, "/api/v1/households/me/members/"+memberID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if kick.StatusCode != http.StatusNoContent {
		t.Fatalf("kick %d %s", kick.StatusCode, decodeErrCode(t, kick))
	}
	kick.Body.Close()

	diss, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/dissolve", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if diss.StatusCode != http.StatusNoContent {
		t.Fatalf("dissolve %d %s", diss.StatusCode, decodeErrCode(t, diss))
	}
	diss.Body.Close()

	var n int
	if err := env.db.QueryRow(`SELECT COUNT(*) FROM users WHERE kind = 'household'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("principal left after dissolve: %d", n)
	}
}

func TestHouseholdAcceptRaceAndExpired(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	createHouseholdUser(t, env, "alice1", "password1")
	createHouseholdUser(t, env, "alice2", "password1")

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	inv, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	var invite struct {
		Code string `json:"code"`
		ID   string `json:"id"`
	}
	_ = json.NewDecoder(inv.Body).Decode(&invite)
	inv.Body.Close()

	_, err = env.db.Exec(`UPDATE household_invites SET expires_at = ? WHERE id = ?`, time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), invite.ID)
	if err != nil {
		t.Fatal(err)
	}
	env.login(t, "alice1", "password1")
	body, _ := json.Marshal(map[string]string{"code": invite.Code})
	expired, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if expired.StatusCode != http.StatusBadRequest || decodeErrCode(t, expired) != "ERR_INVITE_INVALID" {
		t.Fatalf("expired %d %s", expired.StatusCode, decodeErrCode(t, expired))
	}
	expired.Body.Close()

	env.login(t, "admin", "secret123")
	inv2, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	var invite2 struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(inv2.Body).Decode(&invite2)
	inv2.Body.Close()

	codeBody, _ := json.Marshal(map[string]string{"code": invite2.Code})
	var statuses []int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, login := range []string{"alice1", "alice2"} {
		login := login
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodPost, env.server.URL+"/api/v1/auth/login", bytes.NewReader(mustJSON(map[string]string{"login": login, "password": "password1"})))
			req.Header.Set("Content-Type", "application/json")
			lr, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			var tok struct {
				Token string `json:"token"`
			}
			_ = json.NewDecoder(lr.Body).Decode(&tok)
			lr.Body.Close()
			accReq, _ := http.NewRequest(http.MethodPost, env.server.URL+"/api/v1/household-invites/accept", bytes.NewReader(codeBody))
			accReq.Header.Set("Content-Type", "application/json")
			accReq.Header.Set("Authorization", "Bearer "+tok.Token)
			ar, err := http.DefaultClient.Do(accReq)
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			statuses = append(statuses, ar.StatusCode)
			mu.Unlock()
			ar.Body.Close()
		}()
	}
	wg.Wait()
	ok, bad := 0, 0
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusBadRequest, http.StatusConflict:
			bad++
		}
	}
	if ok != 1 || bad != 1 {
		t.Fatalf("race statuses %v", statuses)
	}
}

func TestHouseholdOwnerLeaveAndFlagOff(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	leave, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/leave", nil)
	if err != nil {
		t.Fatal(err)
	}
	if leave.StatusCode != http.StatusConflict || decodeErrCode(t, leave) != "ERR_HOUSEHOLD_OWNER_CANNOT_LEAVE" {
		t.Fatalf("owner leave %d %s", leave.StatusCode, decodeErrCode(t, leave))
	}
	leave.Body.Close()

	feat, _ := json.Marshal(map[string]bool{"household": false})
	put, err := env.authedRequest(http.MethodPut, "/api/v1/admin/features", bytes.NewReader(feat))
	if err != nil {
		t.Fatal(err)
	}
	put.Body.Close()

	me, err := env.authedRequest(http.MethodGet, "/api/v1/households/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	if me.StatusCode != http.StatusNotFound || decodeErrCode(t, me) != "FEATURE_DISABLED" {
		t.Fatalf("flag off %d %s", me.StatusCode, decodeErrCode(t, me))
	}
	me.Body.Close()
}

func TestHouseholdTransferAndMemberLeave(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	adminCookie := env.cookie
	createHouseholdUser(t, env, "member2", "password1")
	cr, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", cr.StatusCode, decodeErrCode(t, cr))
	}
	cr.Body.Close()
	inv, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/invites", nil)
	if err != nil {
		t.Fatal(err)
	}
	var invite struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(inv.Body).Decode(&invite)
	inv.Body.Close()

	env.login(t, "member2", "password1")
	memberCookie := env.cookie
	body, _ := json.Marshal(map[string]string{"code": invite.Code})
	acc, err := env.authedRequest(http.MethodPost, "/api/v1/household-invites/accept", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var me struct {
		Members []struct {
			UserID string `json:"user_id"`
			Login  string `json:"login"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	_ = json.NewDecoder(acc.Body).Decode(&me)
	acc.Body.Close()
	var memberID string
	for _, m := range me.Members {
		if m.Login == "member2" {
			memberID = m.UserID
		}
	}
	env.cookie = adminCookie
	trBody, _ := json.Marshal(map[string]string{"user_id": memberID})
	tr, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/transfer", bytes.NewReader(trBody))
	if err != nil {
		t.Fatal(err)
	}
	if tr.StatusCode != http.StatusOK {
		t.Fatalf("transfer %d %s", tr.StatusCode, decodeErrCode(t, tr))
	}
	var after struct {
		Role    string `json:"role"`
		Members []struct {
			Login string `json:"login"`
			Role  string `json:"role"`
		} `json:"members"`
	}
	_ = json.NewDecoder(tr.Body).Decode(&after)
	tr.Body.Close()
	if after.Role != "member" {
		t.Fatalf("old owner role %s", after.Role)
	}
	owners := 0
	for _, m := range after.Members {
		if m.Role == "owner" {
			owners++
			if m.Login != "member2" {
				t.Fatalf("new owner %s", m.Login)
			}
		}
	}
	if owners != 1 {
		t.Fatalf("owners=%d", owners)
	}

	leave, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/leave", nil)
	if err != nil {
		t.Fatal(err)
	}
	if leave.StatusCode != http.StatusNoContent {
		t.Fatalf("leave %d %s", leave.StatusCode, decodeErrCode(t, leave))
	}
	leave.Body.Close()

	env.cookie = memberCookie
	diss, err := env.authedRequest(http.MethodPost, "/api/v1/households/me/dissolve", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if diss.StatusCode != http.StatusNoContent {
		t.Fatalf("dissolve %d %s", diss.StatusCode, decodeErrCode(t, diss))
	}
	diss.Body.Close()
}

func TestHouseholdCreateTeamKind(t *testing.T) {
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")
	bad, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader(mustJSON(map[string]any{"kind": "clan"})))
	if err != nil {
		t.Fatal(err)
	}
	code := decodeErrCode(t, bad)
	if bad.StatusCode != http.StatusBadRequest || code != "ERR_HOUSEHOLD_KIND" {
		t.Fatalf("invalid kind %d %s", bad.StatusCode, code)
	}
	bad.Body.Close()

	resp, err := env.authedRequest(http.MethodPost, "/api/v1/households", bytes.NewReader(mustJSON(map[string]any{"kind": "team"})))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create team %d %s", resp.StatusCode, decodeErrCode(t, resp))
	}
	var created struct {
		Household *struct {
			Kind string `json:"kind"`
		} `json:"household"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created.Household == nil || created.Household.Kind != "team" {
		t.Fatalf("team payload %#v", created)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
