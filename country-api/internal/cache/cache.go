package cache

import "sync"

// Cache is the interface for our in-memory cache.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
}

// InMemoryCache is a thread-safe in-memory cache.
type InMemoryCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

// NewInMemoryCache creates a new instance of InMemoryCache.
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		items: make(map[string]interface{}),
	}
}

// Get retrieves a value from the cache.
func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.items[key]
	return item, found
}

// Set adds a value to the cache, overwriting an existing one if present.
func (c *InMemoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}