package coordinator

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"sync"
)

// Coordinator holds a list of node addresses.
type Coordinator struct {
	mu    sync.RWMutex
	nodes []string
}

// NewCoordinator creates a new Coordinator instance.
func NewCoordinator() *Coordinator {
	return &Coordinator{
		nodes: []string{},
	}
}

// Register a node (this would be a POST endpoint)
func (c *Coordinator) registerHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Address == "" {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	c.nodes = append(c.nodes, payload.Address)
	c.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

// Where returns the node responsible for a given key.
func (c *Coordinator) whereHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}
	node := c.where(key)
	fmt.Fprintln(w, node)
}

func (c *Coordinator) where(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.nodes) == 0 {
		return ""
	}
	// Simple hash-based routing
	h := fnv.New32a()
	h.Write([]byte(key))
	index := int(h.Sum32()) % len(c.nodes)
	return c.nodes[index]
}

func (c *Coordinator) Start(port string) {
	http.HandleFunc("/register", c.registerHandler)
	http.HandleFunc("/where", c.whereHandler)
	log.Printf("Coordinator running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
