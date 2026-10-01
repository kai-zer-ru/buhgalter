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
	mu           sync.RWMutex
	items        map[string]Response
	epoch        uint64
	OnInvalidate func(userID string, hints InvalidateHints) // optional: realtime publish after user cache drop
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

func (c *Cache) Epoch() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.epoch
}

func (c *Cache) Set(key string, resp Response, ttl time.Duration) {
	resp.until = time.Now().Add(ttl)
	c.mu.Lock()
	c.items[key] = resp
	c.mu.Unlock()
}

// SetIfEpoch stores the response only if no invalidation happened since epoch.
// Prevents a GET that started before a write from filling the cache after InvalidateUser.
func (c *Cache) SetIfEpoch(key string, resp Response, ttl time.Duration, epoch uint64) bool {
	resp.until = time.Now().Add(ttl)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.epoch != epoch {
		return false
	}
	c.items[key] = resp
	return true
}

func (c *Cache) DeletePrefix(prefix string) {
	if prefix == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	for key := range c.items {
		if strings.HasPrefix(key, prefix) {
			delete(c.items, key)
		}
	}
}

// InvalidateUser drops the user's GET cache and publishes a coarse realtime invalidate.
func (c *Cache) InvalidateUser(userID string) {
	c.InvalidateUserHints(userID, InvalidateHints{})
}

// InvalidateUserHints drops the user's GET cache and publishes a targeted realtime invalidate.
func (c *Cache) InvalidateUserHints(userID string, hints InvalidateHints) {
	if c == nil || userID == "" {
		return
	}
	c.DeletePrefix("u:" + userID + ":")
	if c.OnInvalidate != nil {
		c.OnInvalidate(userID, hints)
	}
}

func (c *Cache) DeleteContaining(substr string) {
	if substr == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.epoch++
	for key := range c.items {
		if strings.Contains(key, substr) {
			delete(c.items, key)
		}
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.epoch++
	c.items = make(map[string]Response)
	c.mu.Unlock()
}
