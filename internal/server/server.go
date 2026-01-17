package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"kovra/internal/cache"
	"kovra/internal/handler"
	"kovra/internal/ledger"
	"kovra/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Server represents the HTTP server.
type Server struct {
	httpServer   *http.Server
	logger       *zap.Logger
	pool         *pgxpool.Pool
	ledgerClient *ledger.Client
	cacheClient  *cache.Client
}

// Config holds server configuration.
type Config struct {
	Port         int
	Pool         *pgxpool.Pool
	LedgerClient *ledger.Client
	CacheClient  *cache.Client
	Logger       *zap.Logger
}

// New creates a new HTTP server.
func New(cfg Config) *Server {
	s := &Server{
		logger:       cfg.Logger,
		pool:         cfg.Pool,
		ledgerClient: cfg.LedgerClient,
		cacheClient:  cfg.CacheClient,
	}

	// Create repositories
	legalEntityRepo := repository.NewLegalEntityRepository(cfg.Pool)
	tenantRepo := repository.NewTenantRepository(cfg.Pool)
	walletRepo := repository.NewWalletRepository(cfg.Pool)
	transferRepo := repository.NewTransferRepository(cfg.Pool)

	// Create handlers
	legalEntityHandler := handler.NewLegalEntityHandler(legalEntityRepo)
	tenantHandler := handler.NewTenantHandler(tenantRepo)
	walletHandler := handler.NewWalletHandler(walletRepo, cfg.LedgerClient)
	transferHandler := handler.NewTransferHandler(transferRepo, walletRepo)

	// Setup chi router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.zapLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health check endpoints
	r.Get("/health", s.healthCheck)
	r.Get("/ready", s.readyCheck)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Legal Entities (read-only)
		r.Get("/legal-entities", legalEntityHandler.List)
		r.Get("/legal-entities/{id}", legalEntityHandler.Get)
		r.Get("/legal-entities/code/{code}", legalEntityHandler.GetByCode)
		r.Get("/legal-entities/{id}/tenants", tenantHandler.ListByLegalEntity)

		// Tenants
		r.Post("/tenants", tenantHandler.Create)
		r.Get("/tenants/{id}", tenantHandler.Get)
		r.Patch("/tenants/{id}", tenantHandler.Update)
		r.Get("/tenants/{id}/wallets", walletHandler.ListByTenant)
		r.Get("/tenants/{id}/transfers", transferHandler.ListByTenant)

		// Wallets
		r.Post("/wallets", walletHandler.Create)
		r.Get("/wallets/{id}", walletHandler.Get)
		r.Get("/wallets/{id}/balance", walletHandler.GetBalance)

		// Transfers
		r.Post("/transfers", transferHandler.Create)
		r.Get("/transfers/{id}", transferHandler.Get)
	})

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// healthCheck returns basic health status.
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// readyCheck returns readiness status (all dependencies available).
func (s *Server) readyCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check PostgreSQL
	if err := s.pool.Ping(ctx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not ready","reason":"database unavailable"}`))
		return
	}

	// Check Redis
	if s.cacheClient != nil {
		if err := s.cacheClient.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready","reason":"cache unavailable"}`))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// zapLogger is a middleware that logs requests using zap.
func (s *Server) zapLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.logger.Info("request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", ww.Status()),
			zap.Int("bytes", ww.BytesWritten()),
			zap.Duration("duration", time.Since(start)),
			zap.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}
