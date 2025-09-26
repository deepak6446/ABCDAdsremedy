package transport

import (
	"bytes"
	"distributed-counter/internal/cluster"
	"distributed-counter/internal/counter"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer now correctly initializes the registry's state
func setupTestServer() *Server {
	registry := cluster.NewRegistry("self:8080", nil)
	// **THIS IS THE FIX**: Manually add self, simulating what Start() does.
	registry.HandleHeartbeat("self:8080") 

	// The counter needs a registry that implements its interface.
	// The cluster.Registry works perfectly for this.
	cntr := counter.NewCounter("self:8080", registry, nil)
	return NewServer(registry, cntr)
}

func TestHandleIncrementAndGetCount(t *testing.T) {
	s := setupTestServer()
	
	// Test Increment
	req := httptest.NewRequest(http.MethodPost, "/increment", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Test GetCount
	req = httptest.NewRequest(http.MethodGet, "/count", nil)
	rr = httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var resp map[string]int64
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(1), resp["count"])
}

func TestHandleClusterJoin(t *testing.T) {
	s := setupTestServer()
	body, _ := json.Marshal(map[string]string{"id": "peer1:8081"})
	
	req := httptest.NewRequest(http.MethodPost, "/cluster/join", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	
	assert.Equal(t, http.StatusOK, rr.Code)
	
	var peerList []cluster.Peer
	err := json.Unmarshal(rr.Body.Bytes(), &peerList)
	require.NoError(t, err)

	// The assertion will now correctly pass
	assert.Len(t, peerList, 2, "Should contain self and the new peer")
}

func TestHandleCounterPropagate(t *testing.T) {
	s := setupTestServer()
	inc := counter.Increment{ID: "inc-123", NodeID: "peer1:8081"}
	body, _ := json.Marshal(inc)

	req := httptest.NewRequest(http.MethodPost, "/counter/propagate", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, int64(1), s.counter.Value())
}

func TestInvalidJSONRequests(t *testing.T) {
	s := setupTestServer()
	
	endpoints := []string{"/cluster/join", "/cluster/heartbeat", "/counter/propagate"}
	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader([]byte("{invalid json")))
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}