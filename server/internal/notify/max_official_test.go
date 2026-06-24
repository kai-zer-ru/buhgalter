package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxOfficialNotifierValidateConfig(t *testing.T) {
	t.Parallel()

	n := &MaxOfficialNotifier{
		Token:       "official-token",
		RecipientID: "42",
	}
	if err := n.ValidateConfig(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}

	invalid := &MaxOfficialNotifier{Token: "", RecipientID: ""}
	if err := invalid.ValidateConfig(); err != ErrInvalidConfig {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestMaxOfficialNotifierSendUserRecipient(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotBody string
	var gotPath string
	var gotQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := &MaxOfficialNotifier{
		Token:       "official-token",
		RecipientID: "1001",
		BaseURL:     server.URL,
		APIVersion:  "1.2.5",
	}
	if err := n.Send(context.Background(), "", "hello"); err != nil {
		t.Fatalf("unexpected send error: %v", err)
	}
	if gotAuth != "official-token" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotPath != "/messages" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotQuery != "user_id=1001&v=1.2.5" && gotQuery != "v=1.2.5&user_id=1001" {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if !strings.Contains(gotBody, "\"text\":\"hello\"") {
		t.Fatalf("unexpected body: %q", gotBody)
	}
}

func TestBuildMaxOfficialMessageEndpointChatRecipient(t *testing.T) {
	t.Parallel()

	endpoint, err := buildMaxOfficialMessageEndpoint("https://platform-api2.max.ru", "-123", "1.2.5")
	if err != nil {
		t.Fatalf("unexpected endpoint error: %v", err)
	}
	if endpoint != "https://platform-api2.max.ru/messages?chat_id=-123&v=1.2.5" &&
		endpoint != "https://platform-api2.max.ru/messages?v=1.2.5&chat_id=-123" {
		t.Fatalf("unexpected endpoint: %q", endpoint)
	}
}

