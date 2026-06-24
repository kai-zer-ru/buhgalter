package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxA161NotifierValidateConfig(t *testing.T) {
	t.Parallel()

	n := &MaxA161Notifier{
		Token:  "1234567890123456",
		UserID: "42",
	}
	if err := n.ValidateConfig(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}

	invalid := &MaxA161Notifier{Token: "short", UserID: ""}
	if err := invalid.ValidateConfig(); err != ErrInvalidConfig {
		t.Fatalf("expected ErrInvalidConfig, got: %v", err)
	}
}

func TestMaxA161NotifierSend(t *testing.T) {
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

	n := &MaxA161Notifier{
		Token:   "1234567890123456",
		UserID:  "1001",
		BaseURL: server.URL,
	}
	if err := n.Send(context.Background(), "", "hello"); err != nil {
		t.Fatalf("unexpected send error: %v", err)
	}
	if gotAuth != "1234567890123456" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if gotPath != "/messages" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if gotQuery != "user_id=1001" {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if !strings.Contains(gotBody, "\"text\":\"hello\"") {
		t.Fatalf("unexpected body: %q", gotBody)
	}
}

func TestBuildMaxA161MessageEndpointChatRecipient(t *testing.T) {
	t.Parallel()

	endpoint, err := buildMaxA161MessageEndpoint("https://notify.a161.ru", "-1001")
	if err != nil {
		t.Fatalf("unexpected endpoint error: %v", err)
	}
	if endpoint != "https://notify.a161.ru/messages?chat_id=-1001" {
		t.Fatalf("unexpected endpoint: %q", endpoint)
	}
}

