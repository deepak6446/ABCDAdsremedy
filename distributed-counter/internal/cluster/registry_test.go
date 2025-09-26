package cluster

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockClient now correctly implements the HTTPClient interface
type mockClient struct {
	postFunc func(ctx context.Context, url string, body interface{}, responseBody interface{}) error
}

func (m *mockClient) Post(ctx context.Context, url string, body interface{}, responseBody interface{}) error {
	return m.postFunc(ctx, url, body, responseBody)
}

func TestRegistry_StartAndAnnounce(t *testing.T) {
	var announceCalled bool
	var mu sync.Mutex

	mockHTTPClient := &mockClient{
		postFunc: func(ctx context.Context, url string, body interface{}, responseBody interface{}) error {
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, "http://peer1:8081/cluster/join", url)
			announceCalled = true
			return nil
		},
	}

	// This now compiles correctly
	r := NewRegistry("self:8080", mockHTTPClient)
	r.Start([]string{"peer1:8081"})

	time.Sleep(100 * time.Millisecond) // Allow announce goroutine to run

	mu.Lock()
	assert.True(t, announceCalled)
	mu.Unlock()

	assert.Contains(t, r.peers, "self:8080")
}
// ... Rest of registry_test.go is now correct and will compile ...
func TestRegistry_HandleJoinRequest(t *testing.T) {
	r := NewRegistry("self:8080", nil) // Pass nil for client as it's not used in this function
	r.addPeer(Peer{ID: "self:8080", Addr: "self:8080"})

	peerList := r.HandleJoinRequest("new-peer:8081")

	assert.Len(t, peerList, 2)
	assert.Contains(t, r.peers, "new-peer:8081")
}

func TestRegistry_HandleHeartbeat(t *testing.T) {
	r := NewRegistry("self:8080", nil)

	// Heartbeat from unknown peer
	r.HandleHeartbeat("peer1:8081")
	assert.Contains(t, r.peers, "peer1:8081")

	// Heartbeat from known peer
	p := r.peers["peer1:8081"]
	time.Sleep(10 * time.Millisecond)
	r.HandleHeartbeat("peer1:8081")
	assert.True(t, r.peers["peer1:8081"].LastSeen.After(p.LastSeen))
}

func TestRegistry_RemoveExpiredPeers(t *testing.T) {
	r := NewRegistry("self:8080", nil)
	r.addPeer(Peer{ID: "self:8080", Addr: "self:8080", LastSeen: time.Now()})
	r.addPeer(Peer{ID: "expired-peer:8081", Addr: "expired-peer:8081", LastSeen: time.Now().Add(-20 * time.Second)})
	r.addPeer(Peer{ID: "active-peer:8082", Addr: "active-peer:8082", LastSeen: time.Now()})

	r.removeExpiredPeers()

	assert.NotContains(t, r.peers, "expired-peer:8081")
	assert.Contains(t, r.peers, "active-peer:8082")
	assert.Len(t, r.peers, 2)
}