package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/features"
)

func TestFeatureFlagsBudgetGate(t *testing.T) {
	features.Invalidate()
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/budgets", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("budgets enabled status = %d", resp.StatusCode)
	}

	body, _ := json.Marshal(map[string]bool{"budget": false})
	resp, err = env.authedRequest(http.MethodPut, "/api/v1/admin/features", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put features status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = env.authedRequest(http.MethodGet, "/api/v1/budgets", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("budgets disabled status = %d, want 404", resp.StatusCode)
	}
	if code := apiErrorCode(t, resp); code != apperror.FeatureDisabled {
		t.Fatalf("code = %s, want %s", code, apperror.FeatureDisabled)
	}
}

func TestAdminSettingsWithoutRegistrationField(t *testing.T) {
	features.Invalidate()
	env := setupConfigured(t)
	env.login(t, "admin", "secret123")

	resp, err := env.authedRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var settings map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if _, ok := settings["registration_enabled"]; ok {
		t.Fatal("registration_enabled must not be in admin settings")
	}

	body, _ := json.Marshal(map[string]any{"external_url": ""})
	resp, err = env.authedRequest(http.MethodPut, "/api/v1/admin/settings", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put settings status = %d", resp.StatusCode)
	}
}
