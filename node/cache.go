package node

import (
	"container/list"
	"encoding/json"
	"os"
	"log"
	"sync"
	"time"
)

type Item struct {
	Value      string    `json:"value"`
	ExpiresAt  time.Time `json:"expires_at"`
	Hits       int       `json:"hits"`
	lruElement *list.Element
}

type Cache struct {
	mu              sync.RWMutex
	store           map[string]*Item
	baseTTL         time.Duration
	hitThreshold    int
	ttlIncrement    time.Duration
	cleanupInterval time.Duration
	evictionPolicy  string 
	capacity        int
	evictionList    *list.List 
	persistenceFile string
}

func NewCache() *Cache {
	c := &Cache{
		store:           make(map[string]*Item),
		baseTTL:         10 * time.Second,
		hitThreshold:    5,
		ttlIncrement:    5 * time.Second,
		cleanupInterval: 5 * time.Second,
		evictionPolicy:  "LRU",  
		capacity:        100,    
		evictionList:    list.New(),
		persistenceFile: "",     
	}
	if c.persistenceFile != "" {
		c.loadFromDisk()
	}
	go c.cleanupExpiredItems()
	return c
}

func (c *Cache) Set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if item, exists := c.store[key]; exists {
		item.Value = value
		item.ExpiresAt = now.Add(c.baseTTL)
		item.Hits = 0
		if c.evictionPolicy == "LRU" && item.lruElement != nil {
			c.evictionList.MoveToFront(item.lruElement)
		}
	} else {
		item := &Item{
			Value:     value,
			ExpiresAt: now.Add(c.baseTTL),
			Hits:      0,
		}
		c.store[key] = item
		if c.evictionPolicy == "LRU" {
			elem := c.evictionList.PushFront(key)
			item.lruElement = elem
			if c.evictionList.Len() > c.capacity {
				c.evictLRU()
			}
		}
	}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	item, exists := c.store[key]
	c.mu.RUnlock()
	if !exists || time.Now().After(item.ExpiresAt) {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item.Hits++
	if item.Hits%c.hitThreshold == 0 {
		item.ExpiresAt = item.ExpiresAt.Add(c.ttlIncrement)
	}
	if c.evictionPolicy == "LRU" && item.lruElement != nil {
		c.evictionList.MoveToFront(item.lruElement)
	}
	return item.Value, true
}

func (c *Cache) evictLRU() {
	if c.evictionList.Len() == 0 {
		return
	}
	elem := c.evictionList.Back()
	key := elem.Value.(string)
	delete(c.store, key)
	c.evictionList.Remove(elem)
	log.Printf("Evicted key %s due to LRU policy", key)
}

func (c *Cache) cleanupExpiredItems() {
	ticker := time.NewTicker(c.cleanupInterval)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for key, item := range c.store {
			if now.After(item.ExpiresAt) {
				delete(c.store, key)
				if c.evictionPolicy == "LRU" && item.lruElement != nil {
					c.evictionList.Remove(item.lruElement)
				}
			}
		}
		if c.persistenceFile != "" {
			c.persistToDisk()
		}
		c.mu.Unlock()
	}
}

func (c *Cache) persistToDisk() {
	data, err := json.Marshal(c.store)
	if err != nil {
		log.Println("Error marshalling cache for persistence:", err)
		return
	}
	err = os.WriteFile(c.persistenceFile, data, 0644)
	if err != nil {
		log.Println("Error writing cache to disk:", err)
	}
}

func (c *Cache) loadFromDisk() {
	data, err := os.ReadFile(c.persistenceFile)
	if err != nil {
		log.Println("No persistence file found, starting fresh.")
		return
	}
	err = json.Unmarshal(data, &c.store)
	if err != nil {
		log.Println("Error unmarshalling persistence file:", err)
	}
	if c.evictionPolicy == "LRU" {
		c.evictionList = list.New()
		for key := range c.store {
			elem := c.evictionList.PushFront(key)
			c.store[key].lruElement = elem
		}
	}
}

