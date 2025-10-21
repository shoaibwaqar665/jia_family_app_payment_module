package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Server represents a Prometheus metrics server
type Server struct {
	server *http.Server
	logger *zap.Logger
}

// NewServer creates a new Prometheus metrics server
func NewServer(addr string, logger *zap.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		server: server,
		logger: logger,
	}
}

// Start starts the metrics server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting Prometheus metrics server",
		zap.String("addr", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down Prometheus metrics server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("metrics server shutdown error: %w", err)
	}

	return nil
}
