package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

func TestHandlerSendsHelloAndInvalidate(t *testing.T) {
	hub := NewHub()
	h := &Handler{Hub: hub}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), auth.AuthContextKey, auth.AuthInfo{
			User: auth.User{ID: "user-1"},
		})
		h.ServeHTTP(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read hello: %v", err)
	}
	var hello Event
	if err := json.Unmarshal(data, &hello); err != nil {
		t.Fatalf("unmarshal hello: %v", err)
	}
	if hello.Type != TypeHello {
		t.Fatalf("hello type=%q", hello.Type)
	}

	hub.PublishInvalidate("user-1")

	_, data, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read invalidate: %v", err)
	}
	var inv Event
	if err := json.Unmarshal(data, &inv); err != nil {
		t.Fatalf("unmarshal invalidate: %v", err)
	}
	if inv.Type != TypeInvalidate {
		t.Fatalf("invalidate type=%q", inv.Type)
	}
}
