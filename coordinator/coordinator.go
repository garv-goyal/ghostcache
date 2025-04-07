package coordinator

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type Node struct {
	Address string `json:"address"`
}

type Coordinator struct {
	mu            sync.RWMutex
	nodes         map[uint32]Node 
	hashRing      []uint32        
	checkInterval time.Duration
}

func NewCoordinator() *Coordinator {
	c := &Coordinator{
		nodes:         make(map[uint32]Node),
		checkInterval: 10 * time.Second,
	}
	go c.healthCheckLoop()
	return c
}

func hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func (c *Coordinator) addNode(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	address = strings.TrimSpace(address)
	h := hashKey(address)
	c.nodes[h] = Node{Address: address}
	c.rebuildHashRing()
	log.Printf("Added node: %s (hash: %d)", address, h)
}

func (c *Coordinator) removeNode(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	address = strings.TrimSpace(address)
	h := hashKey(address)
	delete(c.nodes, h)
	c.rebuildHashRing()
	log.Printf("Removed node: %s", address)
}

func (c *Coordinator) rebuildHashRing() {
	ring := make([]uint32, 0, len(c.nodes))
	for h := range c.nodes {
		ring = append(ring, h)
	}
	sort.Slice(ring, func(i, j int) bool { return ring[i] < ring[j] })
	c.hashRing = ring
}

func (c *Coordinator) getNode(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.hashRing) == 0 {
		return "", fmt.Errorf("no nodes available")
	}
	h := hashKey(key)
	// find the first node with hash >= h
	idx := sort.Search(len(c.hashRing), func(i int) bool { return c.hashRing[i] >= h })
	if idx == len(c.hashRing) {
		idx = 0
	}
	node := c.nodes[c.hashRing[idx]]
	return node.Address, nil
}

func (c *Coordinator) healthCheckLoop() {
	ticker := time.NewTicker(c.checkInterval)
	for range ticker.C {
		c.mu.RLock()
		nodesCopy := make([]Node, 0, len(c.nodes))
		for _, n := range c.nodes {
			nodesCopy = append(nodesCopy, n)
		}
		c.mu.RUnlock()
		for _, node := range nodesCopy {
			url := node.Address + "/health"
			client := http.Client{
				Timeout: 2 * time.Second,
			}
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				log.Printf("Health check failed for node %s, removing it", node.Address)
				c.removeNode(node.Address)
			} else {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	}
}

func (c *Coordinator) registerHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Address == "" {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	c.addNode(payload.Address)
	w.WriteHeader(http.StatusOK)
}

func (c *Coordinator) whereHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}
	address, err := c.getNode(key)
	if err != nil {
		http.Error(w, "No nodes available", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, strings.TrimSpace(address))
}

func (c *Coordinator) Start(port string) {
	http.HandleFunc("/register", c.registerHandler)
	http.HandleFunc("/where", c.whereHandler)
	log.Printf("Coordinator running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
