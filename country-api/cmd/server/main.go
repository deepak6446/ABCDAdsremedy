package main

import (
	"context"
	"errors"
	"fmt"
	"country-api/internal/api"
	"country-api/internal/cache"
	"country-api/internal/client"
	"country-api/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// The main function is now just a simple wrapper around run()
	if err := run(context.Background()); err != nil {
		log.Printf("error: server failed: %v", err)
		os.Exit(1)
	}
}

// run sets up and runs the application.
func run(ctx context.Context) error {
	// 1. Create a new context that is canceled when an interrupt or SIGTERM signal is received.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize dependencies
	inMemoryCache := cache.NewInMemoryCache()
	restCountriesClient := client.NewRestCountriesClient()
	countryService := service.NewCountryService(inMemoryCache, restCountriesClient)
	countryHandler := api.NewCountryHandler(countryService)
	router := api.NewRouter(countryHandler)

	// 3. Configure the server
	server := &http.Server{
		Addr:         ":8000",
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 4. Run the server in a separate goroutine so that it doesn't block.
	serverErrors := make(chan error, 1)
	go func() {
		log.Println("Server starting on port 8000...")
		// ListenAndServe always returns a non-nil error.
		// We filter for ErrServerClosed which is the expected error on graceful shutdown.
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 5. Block until a signal is received or an error occurs.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("error starting server: %w", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received")
	}

	// 6. Gracefully shut down the server.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server gracefully stopped")
	return nil
}