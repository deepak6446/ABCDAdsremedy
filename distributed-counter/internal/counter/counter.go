package counter

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
)

// PeerRegistry defines the interface for getting peer addresses.
// This allows us to mock the cluster.Registry in tests.
type PeerRegistry interface {
	GetPeerAddrs() []string
}

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Post(ctx context.Context, url string, body interface{}, responseBody interface{}) error
}

type Increment struct {
	ID     string `json:"id"`
	NodeID string `json:"node_id"`
}


// Counter is a thread-safe, distributed, in-memory counter.
type Counter struct {
	mu             sync.RWMutex
	value          int64
	seenIncrements map[string]struct{}
	registry       PeerRegistry // Depend on the interface
	httpClient     HTTPClient   // Depend on the interface
	selfID         string
}

// NewCounter creates a new distributed counter.
func NewCounter(selfID string, registry PeerRegistry, client HTTPClient) *Counter {
	return &Counter{
		value:          0,
		seenIncrements: make(map[string]struct{}),
		registry:       registry,
		httpClient:     client,
		selfID:         selfID,
	}
}

// IncrementAndPropagate increments the local counter and propagates the change to peers.
func (c *Counter) IncrementAndPropagate() {
	increment := Increment{
		ID:     uuid.NewString(),
		NodeID: c.selfID,
	}

	// Apply locally first
	c.ApplyIncrement(increment)

	// Propagate to all peers
	peerAddrs := c.registry.GetPeerAddrs()
	for _, addr := range peerAddrs {
		go c.propagate(addr, increment)
	}
}

// ApplyIncrement applies a given increment if it hasn't been seen before. Returns true if applied.
func (c *Counter) ApplyIncrement(inc Increment) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, seen := c.seenIncrements[inc.ID]; seen {
		return false // Already applied
	}

	c.value++
	c.seenIncrements[inc.ID] = struct{}{}
	log.Printf("Applied increment %s from node %s. New value: %d", inc.ID, inc.NodeID, c.value)
	return true
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

func (c *Counter) propagate(peerAddr string, inc Increment) {
	url := "http://" + peerAddr + "/counter/propagate"

	op := func() error {
		err := c.httpClient.Post(context.Background(), url, inc, nil)
		if err != nil {
			log.Printf("Failed to propagate increment %s to %s. Retrying... Error: %v", inc.ID, peerAddr, err)
		}
		return err
	}

	// Retry with exponential backoff
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 10 * time.Second // Stop retrying after 10 seconds

	err := backoff.Retry(op, b)
	if err != nil {
		log.Printf("Permanently failed to propagate increment %s to %s after multiple retries", inc.ID, peerAddr)
	}
}