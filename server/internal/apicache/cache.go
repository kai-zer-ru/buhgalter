package apicache

import (
	"strings"
	"sync"
	"time"
)

// Response holds a cached HTTP response body.
type Response struct {
	Status      int
	Body        []byte
	ContentType string
	until       time.Time
}

// Cache is an in-memory store of serialized GET responses.
type Cache struct {
	mu    sync.RWMutex
	items map[string]Response
}

func New() *Cache {
	return &Cache{items: make(map[string]Response)}
}

func (c *Cache) Get(key string) (Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.until) {
		return Response{}, false
	}
	return item, true
}

func (c *Cache) Set(key string, resp Response, ttl time.Duration) {
	resp.until = time.Now().Add(ttl)
	c.mu.Lock()
	c.items[key] = resp
	c.mu.Unlock()
}

func (c *Cache) DeletePrefix(prefix string) {
	if prefix == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]Response)
	c.mu.Unlock()
}
