package transport

import (
	"distributed-counter/internal/cluster"
	"distributed-counter/internal/counter"
	"encoding/json"
	"net/http"
)

// Server encapsulates all HTTP handling logic.
type Server struct {
	registry *cluster.Registry
	counter  *counter.Counter
	router   *http.ServeMux
}

func NewServer(registry *cluster.Registry, counter *counter.Counter) *Server {
	s := &Server{
		registry: registry,
		counter:  counter,
		router:   http.NewServeMux(),
	}
	s.registerHandlers()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) registerHandlers() {
	// Public API
	s.router.HandleFunc("POST /increment", s.handleIncrement)
	s.router.HandleFunc("GET /count", s.handleGetCount)

	// Internal Cluster API
	s.router.HandleFunc("POST /cluster/join", s.handleClusterJoin)
	s.router.HandleFunc("POST /cluster/heartbeat", s.handleClusterHeartbeat)

	// Internal Counter API
	s.router.HandleFunc("POST /counter/propagate", s.handleCounterPropagate)
}

// --- Public Handlers ---

func (s *Server) handleIncrement(w http.ResponseWriter, r *http.Request) {
	s.counter.IncrementAndPropagate()
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetCount(w http.ResponseWriter, r *http.Request) {
	value := s.counter.Value()
	response := map[string]int64{"count": value}
	s.respondJSON(w, http.StatusOK, response)
}

// --- Internal Handlers ---

func (s *Server) handleClusterJoin(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	peerID := body["id"]
	if peerID == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}
	peerList := s.registry.HandleJoinRequest(peerID)
	s.respondJSON(w, http.StatusOK, peerList)
}

func (s *Server) handleClusterHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	peerID := body["id"]
	if peerID == "" {
		http.Error(w, "Peer ID is required", http.StatusBadRequest)
		return
	}
	s.registry.HandleHeartbeat(peerID)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCounterPropagate(w http.ResponseWriter, r *http.Request) {
	var inc counter.Increment
	if err := json.NewDecoder(r.Body).Decode(&inc); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	s.counter.ApplyIncrement(inc)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(response)
}