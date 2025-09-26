package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRun_GracefulShutdown tests the full lifecycle: startup and graceful shutdown.
func TestRun_GracefulShutdown(t *testing.T) {
	// Create a context that we can cancel to simulate a shutdown signal.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	var runErr error
	go func() {
		defer wg.Done()
		// Run the application with a high, non-standard port to avoid conflicts.
		args := []string{"-port=8099"}
		runErr = run(ctx, args)
	}()

	// Poll the server to wait for it to be ready.
	serverIsReady := false
	for i := 0; i < 20; i++ { // Poll for up to 2 seconds
		// The /count endpoint should be available.
		_, err := http.Get("http://localhost:8099/count")
		if err == nil {
			serverIsReady = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	require.True(t, serverIsReady, "Server did not start within the expected time")

	// Trigger the graceful shutdown by canceling the context.
	cancel()

	// Wait for the run function to exit.
	wg.Wait()

	// A clean shutdown should result in no error.
	assert.NoError(t, runErr, "Expected a clean shutdown")
}

// TestRun_PortInUse tests the failure case where a server cannot start.
func TestRun_PortInUse(t *testing.T) {
	// Occupy a port to force a startup failure.
	port := "8098"
	listener, err := net.Listen("tcp", ":"+port)
	require.NoError(t, err, "Should be able to listen on the test port")
	defer listener.Close()

	// Run the application, which should fail immediately because the port is in use.
	// We use a background context because it won't be canceled.
	args := []string{"-port=" + port}
	err = run(context.Background(), args)

	// We expect an error related to the address being in use.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address already in use", "Expected error for port in use")
}