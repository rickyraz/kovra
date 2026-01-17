package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"kovra/internal/config"
	"kovra/internal/db"
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

	// TODO: Connect to TigerBeetle
	// TODO: Connect to Redis
	// TODO: Start HTTP server

	logger.Info("kovra ready",
		zap.Int("port", cfg.Server.Port),
	)

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down")

	return nil
}
