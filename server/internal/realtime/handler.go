package realtime

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/kai-zer-ru/buhgalter/internal/apperror"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
)

// Handler serves GET /api/v1/realtime WebSocket upgrades.
type Handler struct {
	Hub *Hub
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Hub == nil {
		apperror.WriteR(w, r, http.StatusServiceUnavailable, apperror.ServiceUnavailable)
		return
	}
	info, ok := auth.FromContext(r.Context())
	if !ok {
		apperror.WriteR(w, r, http.StatusUnauthorized, apperror.Unauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Same-origin browser UI; empty patterns allow the request host.
		// Cross-origin native clients (Android later) will need OriginPatterns.
		OriginPatterns: []string{},
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(4096)

	c := h.Hub.Register(info.User.ID)
	defer h.Hub.Unregister(c)

	ctx := r.Context()
	writeCtx, cancelWrite := context.WithTimeout(ctx, writeWait)
	_ = writeJSON(writeCtx, conn, newHello())
	cancelWrite()

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				return
			}
			// Client messages (pong / subscribe) are ignored in MVP.
		}
	}()

	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case <-readDone:
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return
		case <-c.closed:
			_ = conn.Close(websocket.StatusGoingAway, "replaced")
			return
		case <-pingTicker.C:
			wctx, cancel := context.WithTimeout(ctx, writeWait)
			err := writeJSON(wctx, conn, newPing())
			cancel()
			if err != nil {
				_ = conn.Close(websocket.StatusGoingAway, "ping failed")
				return
			}
		case ev := <-c.send:
			wctx, cancel := context.WithTimeout(ctx, writeWait)
			err := writeJSON(wctx, conn, ev)
			cancel()
			if err != nil {
				_ = conn.Close(websocket.StatusGoingAway, "write failed")
				return
			}
		}
	}
}

func writeJSON(ctx context.Context, conn *websocket.Conn, ev Event) error {
	payload, err := encodeEvent(ev)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, payload)
}
