package node

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type Server struct {
	cache *Cache
}

func NewServer() *Server {
	return &Server{
		cache: NewCache(),
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) setHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Status: "error", Message: "Invalid JSON"})
		return
	}
	s.cache.Set(payload.Key, payload.Value)
	writeJSON(w, http.StatusOK, Response{Status: "success", Message: "Key set successfully"})
}

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, Response{Status: "error", Message: "Missing key"})
		return
	}
	value, ok := s.cache.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, Response{Status: "error", Message: "Key not found or expired"})
		return
	}
	writeJSON(w, http.StatusOK, Response{Status: "success", Data: value})
}

func (s *Server) deleteHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Status: "error", Message: "Invalid JSON"})
		return
	}
	if s.cache.Delete(payload.Key) {
		writeJSON(w, http.StatusOK, Response{Status: "success", Message: "Key deleted"})
	} else {
		writeJSON(w, http.StatusNotFound, Response{Status: "error", Message: "Key not found"})
	}
}

func (s *Server) updateTTLHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Key string  `json:"key"`
		TTL float64 `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Status: "error", Message: "Invalid JSON"})
		return
	}
	newTTL := time.Duration(payload.TTL * float64(time.Second))
	if s.cache.UpdateTTL(payload.Key, newTTL) {
		writeJSON(w, http.StatusOK, Response{Status: "success", Message: "TTL updated"})
	} else {
		writeJSON(w, http.StatusNotFound, Response{Status: "error", Message: "Key not found"})
	}
}

func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.cache.Stats()
	writeJSON(w, http.StatusOK, Response{Status: "success", Data: stats})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Status: "success", Message: "OK"})
}

func (s *Server) Start(port string) {
	http.HandleFunc("/set", s.setHandler)
	http.HandleFunc("/get", s.getHandler)
	http.HandleFunc("/delete", s.deleteHandler)
	http.HandleFunc("/updateTTL", s.updateTTLHandler)
	http.HandleFunc("/stats", s.statsHandler)
	http.HandleFunc("/health", s.healthHandler)
	http.Handle("/metrics", promhttp.Handler())
	logrus.Infof("Cache node running on port %s", port)
	logrus.Fatal(http.ListenAndServe(":"+port, nil))
}