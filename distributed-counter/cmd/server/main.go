package main

import (
	"context"
	"distributed-counter/internal/cluster"
	"distributed-counter/internal/counter"
	"distributed-counter/internal/httpclient"
	"distributed-counter/internal/transport"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	// 1. Create a context that is canceled on a SIGINT or SIGTERM signal.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Run the application, passing in the context and command-line arguments.
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Printf("error: application failed: %v", err)
		os.Exit(1)
	}
}

// run sets up and executes the application. It is designed to be testable.
func run(ctx context.Context, args []string) error {
	// Use a custom flag set to avoid interfering with the global one during tests.
	fs := flag.NewFlagSet("node", flag.ExitOnError)
	port := fs.String("port", "8080", "Port for the node to listen on")
	peers := fs.String("peers", "", "Comma-separated list of initial peers (e.g., localhost:8081,localhost:8082)")
	
	// Parse the provided arguments.
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	selfID := "localhost:" + *port
	initialPeers := parsePeers(*peers)

	// --- Dependency Injection ---
	client := httpclient.New()
	registry := cluster.NewRegistry(selfID, client)
	cntr := counter.NewCounter(selfID, registry, client)
	httpServer := transport.NewServer(registry, cntr)

	// Start service discovery
	registry.Start(initialPeers)

	// --- Server Setup and Graceful Shutdown ---
	server := &http.Server{
		Addr:    ":" + *port,
		Handler: httpServer,
	}

	// Channel to receive errors from the server goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Node %s starting on port %s", selfID, *port)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Block until the context is canceled (e.g., by a signal) or the server fails.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received")
	}

	// Gracefully shut down the server.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server gracefully stopped")
	return nil
}

func parsePeers(peerString string) []string {
	if peerString == "" {
		return nil
	}
	return strings.Split(peerString, ",")
}