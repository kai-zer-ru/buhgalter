package realtime

import (
	"testing"
	"time"
)

func TestHubPublishIsolatesUsers(t *testing.T) {
	h := NewHub()
	a := h.Register("user-a")
	b := h.Register("user-b")
	defer h.Unregister(a)
	defer h.Unregister(b)

	h.PublishInvalidate("user-a")

	select {
	case ev := <-a.send:
		if ev.Type != TypeInvalidate {
			t.Fatalf("type=%q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("expected event for user-a")
	}

	select {
	case ev := <-b.send:
		t.Fatalf("user-b got unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubFanoutSameUser(t *testing.T) {
	h := NewHub()
	c1 := h.Register("user-1")
	c2 := h.Register("user-1")
	defer h.Unregister(c1)
	defer h.Unregister(c2)

	h.PublishInvalidate("user-1")
	for _, c := range []*Conn{c1, c2} {
		select {
		case ev := <-c.send:
			if ev.Type != TypeInvalidate {
				t.Fatalf("type=%q", ev.Type)
			}
		case <-time.After(time.Second):
			t.Fatal("expected event")
		}
	}
}

func TestHubDropsSlowClient(t *testing.T) {
	h := NewHub()
	c := h.Register("user-1")
	// Fill buffer without reading.
	for i := 0; i < sendBuffer; i++ {
		h.PublishInvalidate("user-1")
	}
	// One more should drop the connection.
	h.PublishInvalidate("user-1")

	deadline := time.After(time.Second)
	for h.CountForUser("user-1") != 0 {

		select {
		case <-deadline:
			t.Fatalf("expected slow client dropped, count=%d", h.CountForUser("user-1"))
		case <-time.After(10 * time.Millisecond):
		}
	}
	_ = c
}

func TestHubCapsConnectionsPerUser(t *testing.T) {
	h := NewHub()
	conns := make([]*Conn, 0, maxConnsPerUser+2)
	for i := 0; i < maxConnsPerUser+2; i++ {
		conns = append(conns, h.Register("user-1"))
	}
	if got := h.CountForUser("user-1"); got != maxConnsPerUser {
		t.Fatalf("count=%d want=%d", got, maxConnsPerUser)
	}
	for _, c := range conns {
		h.Unregister(c)
	}
}
