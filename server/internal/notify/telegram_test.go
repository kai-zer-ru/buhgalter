package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	if err := n.Send(context.Background(), "", "hello"); err != nil {
		t.Fatal(err)
	}
}

func TestTelegramNotifierInvalidConfig(t *testing.T) {
	n := &TelegramNotifier{}
	if err := n.ValidateConfig(); err != ErrInvalidConfig {
		t.Fatalf("got %v", err)
	}
}
