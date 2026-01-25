package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"kovra/internal/cache"
	"kovra/internal/config"
	"kovra/internal/db"
	"kovra/internal/ledger"
	"kovra/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("failed to run: %v", err)
		os.Exit(1) // explicit, defer sudah jalan
	}
}

// run initializes all dependencies, starts the HTTP server,
// blocks until shutdown signal or server failure, and
// performs graceful cleanup before exiting.
func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel() // lifecycle cleanup

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Setup logger
	var logger *zap.Logger
	if cfg.IsDevelopment() {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}
	defer logger.Sync() // lifecycle cleanup

	logger.Info("starting kovra",
		zap.String("env", cfg.Server.Env),
		zap.Int("port", cfg.Server.Port),
	)

	// Connect to PostgreSQL
	database, err := db.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer database.Close()
	logger.Info("connected to PostgreSQL")

	// Connect to TigerBeetle
	ledgerClient, err := ledger.NewClient(cfg.TigerBeetle)
	if err != nil {
		return fmt.Errorf("ledger unavailable: %w", err)
	}
	defer ledgerClient.Close()

	// Connect to Redis
	cacheClient, err := cache.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("redis unavailable: %w", err)
	}
	defer cacheClient.Close()

	// Create and start HTTP server
	srv := server.New(server.Config{
		Port:         cfg.Server.Port,
		Pool:         database.Pool(),
		LedgerClient: ledgerClient,
		CacheClient:  cacheClient,
		Logger:       logger,
	})

	// Start server in goroutine
	// Start HTTP server in a separate goroutine because ListenAndServe() is BLOCKING.
	// If run directly, it would halt the main control flow and prevent graceful shutdown.
	errChan := make(chan error, 1)
	// Buffered channel (size=1) to avoid goroutine deadlock
	// if the server fails before the receiver is ready.

	go func() {
		// Start the server; this call blocks until the server stops or fails.
		// Only unexpected errors are returned (http.ErrServerClosed is ignored).
		if err := srv.Start(); err != nil {
			// Propagate server startup/runtime errors back to the main goroutine.
			// This allows the main select loop to handle server failures
			// the same way as OS shutdown signals.
			errChan <- err
		}
	}()

	logger.Info("kovra ready",
		zap.Int("port", cfg.Server.Port),
	)

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	logger.Info("shutdown complete")
	return nil
}
