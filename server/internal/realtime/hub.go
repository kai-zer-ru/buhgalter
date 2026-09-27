package realtime

import (
	"encoding/json"
	"sync"
	"time"
)

const (
	maxConnsPerUser = 16
	sendBuffer      = 8
	pingInterval    = 25 * time.Second
	writeWait       = 10 * time.Second
)

// Conn is a single client subscription.
type Conn struct {
	userID    string
	send      chan Event
	closeOnce sync.Once
	closed    chan struct{}
}

func (c *Conn) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		// Do not close send: concurrent Publish may still select on it.
	})
}

// Hub fans out events to authenticated WebSocket clients by user_id.
type Hub struct {
	mu    sync.Mutex
	conns map[string]map[*Conn]struct{} // userID → set
}

func NewHub() *Hub {
	return &Hub{conns: make(map[string]map[*Conn]struct{})}
}

// Register adds a connection; excess oldest connections for the user are closed.
func (h *Hub) Register(userID string) *Conn {
	c := &Conn{
		userID: userID,
		send:   make(chan Event, sendBuffer),
		closed: make(chan struct{}),
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	set := h.conns[userID]
	if set == nil {
		set = make(map[*Conn]struct{})
		h.conns[userID] = set
	}
	for len(set) >= maxConnsPerUser {
		var oldest *Conn
		for existing := range set {
			oldest = existing
			break
		}
		if oldest == nil {
			break
		}
		delete(set, oldest)
		oldest.close()
	}
	set[c] = struct{}{}
	return c
}

// Unregister removes a connection.
func (h *Hub) Unregister(c *Conn) {
	if c == nil {
		return
	}
	h.mu.Lock()
	set := h.conns[c.userID]
	if set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, c.userID)
		}
	}
	h.mu.Unlock()
	c.close()
}

// Publish sends an event to all connections of userID (non-blocking per conn).
func (h *Hub) Publish(userID string, ev Event) {
	if h == nil || userID == "" {
		return
	}
	h.mu.Lock()
	set := h.conns[userID]
	targets := make([]*Conn, 0, len(set))
	for c := range set {
		targets = append(targets, c)
	}
	h.mu.Unlock()

	for _, c := range targets {
		select {
		case <-c.closed:
		case c.send <- ev:
		default:
			h.drop(c)
		}
	}
}

// PublishInvalidate publishes a coarse (no hint_paths) invalidate event.
func (h *Hub) PublishInvalidate(userID string) {
	h.Publish(userID, NewInvalidate(nil, nil))
}

// PublishInvalidateHints publishes a path-aware invalidate event.
func (h *Hub) PublishInvalidateHints(userID string, hintPaths, entities []string) {
	h.Publish(userID, NewInvalidate(hintPaths, entities))
}

func (h *Hub) drop(c *Conn) {
	h.mu.Lock()
	set := h.conns[c.userID]
	if set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, c.userID)
		}
	}
	h.mu.Unlock()
	c.close()
}

// CountForUser returns live connections for tests.
func (h *Hub) CountForUser(userID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns[userID])
}

func encodeEvent(ev Event) ([]byte, error) {
	return json.Marshal(ev)
}
