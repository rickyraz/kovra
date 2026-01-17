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
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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
	defer logger.Sync()

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
		logger.Warn("failed to connect to TigerBeetle, running without ledger",
			zap.Error(err),
		)
		// Continue without TigerBeetle - some endpoints will fail
	} else {
		defer ledgerClient.Close()
		logger.Info("connected to TigerBeetle",
			zap.Strings("addresses", cfg.TigerBeetle.Addresses),
		)
	}

	// Connect to Redis
	cacheClient, err := cache.NewClient(ctx, cfg.Redis.URL)
	if err != nil {
		logger.Warn("failed to connect to Redis, running without cache",
			zap.Error(err),
		)
		// Continue without Redis - rate limiting and caching disabled
	} else {
		defer cacheClient.Close()
		logger.Info("connected to Redis")
	}

	// Create and start HTTP server
	srv := server.New(server.Config{
		Port:         cfg.Server.Port,
		Pool:         database.Pool(),
		LedgerClient: ledgerClient,
		CacheClient:  cacheClient,
		Logger:       logger,
	})

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
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
