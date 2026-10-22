package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTelegramNotifierSend(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	n := &TelegramNotifier{Token: "tok", ChatID: "123", Client: mock.Client()}
	if err := n.ValidateConfig(); err != nil {
		t.Fatal(err)
	}
	if err := n.Send(context.Background(), "", "hello", nil); err != nil {
		t.Fatal(err)
	}
}

func TestTelegramNotifierSendWithKeyboard(t *testing.T) {
	var gotBody string
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer mock.Close()
	t.Setenv("BUHGALTER_TELEGRAM_BASE_URL", mock.URL)

	kb := &InlineKeyboard{Rows: [][]KeyboardButton{
		{{Type: "url", Text: "Open", URL: "https://example.com"}},
	}}
	n := &TelegramNotifier{Token: "tok", ChatID: "123", Client: mock.Client()}
	if err := n.Send(context.Background(), "", "hello", kb); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotBody, `"reply_markup"`) || !strings.Contains(gotBody, `"inline_keyboard"`) {
		t.Fatalf("body missing reply_markup: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"url":"https://example.com"`) {
		t.Fatalf("body missing url: %s", gotBody)
	}
}

func TestTelegramNotifierInvalidConfig(t *testing.T) {
	n := &TelegramNotifier{}
	if err := n.ValidateConfig(); err != ErrInvalidConfig {
		t.Fatalf("got %v", err)
	}
}
