package counter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockRegistry satisfies the PeerRegistry interface.
type MockRegistry struct {
	peers []string
}

func (m *MockRegistry) GetPeerAddrs() []string {
	return m.peers
}

// MockHTTPClient satisfies the HTTPClient interface.
type MockHTTPClient struct {
	PostFunc func(ctx context.Context, url string, body interface{}, responseBody interface{}) error
}

func (m *MockHTTPClient) Post(ctx context.Context, url string, body interface{}, responseBody interface{}) error {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, url, body, responseBody)
	}
	return nil
}

func TestCounter_SingleNodeIncrement(t *testing.T) {
	registry := &MockRegistry{peers: []string{}}
	client := &MockHTTPClient{}
	c := NewCounter("node1", registry, client)

	c.ApplyIncrement(Increment{ID: "inc1", NodeID: "node1"})
	assert.Equal(t, int64(1), c.Value())
}

func TestCounter_Deduplication(t *testing.T) {
	registry := &MockRegistry{}
	client := &MockHTTPClient{}
	c := NewCounter("node1", registry, client)
	inc := Increment{ID: "inc1", NodeID: "node1"}

	appliedFirst := c.ApplyIncrement(inc)
	assert.True(t, appliedFirst)
	assert.Equal(t, int64(1), c.Value())

	appliedSecond := c.ApplyIncrement(inc)
	assert.False(t, appliedSecond)
	assert.Equal(t, int64(1), c.Value(), "Counter should not be incremented twice for the same ID")
}

func TestCounter_ConcurrentIncrements(t *testing.T) {
	registry := &MockRegistry{}
	client := &MockHTTPClient{}
	c := NewCounter("node1", registry, client)
	var wg sync.WaitGroup
	numIncrements := 1000

	for i := 0; i < numIncrements; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.ApplyIncrement(Increment{ID: fmt.Sprintf("inc-%d", i), NodeID: "node1"})
		}(i)
	}
	wg.Wait()

	assert.Equal(t, int64(numIncrements), c.Value())
}

func TestCounter_IncrementAndPropagate(t *testing.T) {
	propagateCalled := make(chan Increment, 1)
	mockClient := &MockHTTPClient{
		PostFunc: func(ctx context.Context, url string, body interface{}, responseBody interface{}) error {
			assert.Equal(t, "http://peer1:8081/counter/propagate", url)
			propagateCalled <- body.(Increment)
			return nil
		},
	}

	registry := &MockRegistry{peers: []string{"peer1:8081"}}
	c := NewCounter("node1:8080", registry, mockClient)

	c.IncrementAndPropagate()

	assert.Equal(t, int64(1), c.Value(), "Local counter should be incremented")

	select {
	case inc := <-propagateCalled:
		assert.NotEmpty(t, inc.ID)
		assert.Equal(t, "node1:8080", inc.NodeID)
	case <-time.After(1 * time.Second):
		t.Fatal("Propagation was not called")
	}
}