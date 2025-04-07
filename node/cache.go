package node

import (
	"sync"
	"time"
)

const (
	baseTTL       = 10 * time.Second // Base TTL of 10 seconds
	hitsThreshold = 5                // Increase TTL every 5 hits
	ttlIncrement  = 5 * time.Second  // Increase TTL by 5 seconds
)

type Item struct {
	Value     string
	ExpiresAt time.Time
	Hits      int
}

type Cache struct {
	mu    sync.RWMutex
	store map[string]*Item
}

// NewCache creates a new Cache instance.
func NewCache() *Cache {
	c := &Cache{
		store: make(map[string]*Item),
	}
	// Start the cleanup goroutine
	go c.cleanupExpiredItems()
	return c
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &Item{
		Value:     value,
		ExpiresAt: time.Now().Add(baseTTL),
		Hits:      0,
	}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	item, exists := c.store[key]
	c.mu.RUnlock()
	if !exists || time.Now().After(item.ExpiresAt) {
		return "", false
	}

	// Increase the hit count and adjust TTL if needed
	c.mu.Lock()
	item.Hits++
	if item.Hits%hitsThreshold == 0 {
		item.ExpiresAt = item.ExpiresAt.Add(ttlIncrement)
	}
	c.mu.Unlock()

	return item.Value, true
}

// Periodically remove expired items from the cache.
func (c *Cache) cleanupExpiredItems() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for key, item := range c.store {
			if now.After(item.ExpiresAt) {
				delete(c.store, key)
			}
		}
		c.mu.Unlock()
	}
}
