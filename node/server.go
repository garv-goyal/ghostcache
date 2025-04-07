package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Server struct {
	cache *Cache
	// You can add configuration details, such as port number, etc.
}

func NewServer() *Server {
	return &Server{
		cache: NewCache(),
	}
}

func (s *Server) setHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	s.cache.Set(payload.Key, payload.Value)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Missing key", http.StatusBadRequest)
		return
	}
	value, ok := s.cache.Get(key)
	if !ok {
		http.Error(w, "Key not found or expired", http.StatusNotFound)
		return
	}
	fmt.Fprintln(w, value)
}

func (s *Server) Start(port string) {
	http.HandleFunc("/set", s.setHandler)
	http.HandleFunc("/get", s.getHandler)
	log.Printf("Cache node running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
