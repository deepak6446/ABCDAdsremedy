package main

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_GracefulShutdown(t *testing.T) {
	// Create a context that we can cancel to simulate a shutdown signal
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called even if the test panics

	var wg sync.WaitGroup
	wg.Add(1)

	var runErr error
	go func() {
		defer wg.Done()
		runErr = run(ctx)
	}()

	// Give the server a moment to start.
	// In a real-world scenario, you'd poll an endpoint to check readiness.
	// For this test, a short sleep is sufficient and simpler.
	serverIsReady := false
	for i := 0; i < 10; i++ {
		// Attempt to connect to a known endpoint. We expect a 400 because 'name' is missing,
		// but getting any response means the server is up.
		_, err := http.Get("http://localhost:8000/api/countries/search")
		if err == nil {
			serverIsReady = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.True(t, serverIsReady, "Server did not start in time")

	// Trigger the graceful shutdown by canceling the context
	cancel()

	// Wait for the run function to exit
	wg.Wait()

	// Assert that the run function returned no error, indicating a clean shutdown
	assert.NoError(t, runErr, "Expected a clean shutdown")
}