package server

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/server/interceptors"
)

// GRPCServer represents a gRPC server
type GRPCServer struct {
	server *grpc.Server
	config *config.Config
	logger *zap.Logger
}

// NewGRPCServer creates a new gRPC server instance with all interceptors
func NewGRPCServer(cfg *config.Config) *GRPCServer {
	// Get logger instance
	logger := log.L(context.Background())

	// Create interceptors
	authInterceptor := interceptors.NewAuthInterceptor()
	loggingInterceptor := interceptors.NewLoggingInterceptor()

	// Recovery options
	recoveryOpts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandler(func(p interface{}) (err error) {
			logger.Error("gRPC panic recovered", zap.Any("panic", p))
			return status.Errorf(codes.Internal, "internal server error")
		}),
	}

	// Zap logging options
	zapOpts := []grpc_zap.Option{
		grpc_zap.WithLevels(grpc_zap.DefaultCodeToLevel),
	}

	// Create server with interceptor chain
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(recoveryOpts...),
			grpc_zap.UnaryServerInterceptor(logger, zapOpts...),
			authInterceptor.Unary(),
			loggingInterceptor.Unary(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(recoveryOpts...),
			grpc_zap.StreamServerInterceptor(logger, zapOpts...),
			authInterceptor.Stream(),
			loggingInterceptor.Stream(),
		)),
	)

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	// Set initial health status
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// Register reflection service for non-production environments
	env := os.Getenv("ENV")
	if env != "prod" && env != "production" {
		logger.Info("Registering gRPC reflection (non-production environment)")
		reflection.Register(server)
	}

	return &GRPCServer{
		server: server,
		config: cfg,
		logger: logger,
	}
}

// RegisterService registers a gRPC service with the server
func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// GetServer returns the underlying gRPC server
func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}

// Serve starts the gRPC server and handles graceful shutdown
func (s *GRPCServer) Serve(ctx context.Context) error {
	// Create listener
	listener, err := net.Listen("tcp", s.config.GRPC.Address)
	if err != nil {
		return err
	}

	s.logger.Info("gRPC server starting",
		zap.String("address", s.config.GRPC.Address))

	// Channel to receive server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		serverErr <- s.server.Serve(listener)
	}()

	// Channel to receive shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either server error, context cancellation, or shutdown signal
	select {
	case err := <-serverErr:
		return err

	case <-ctx.Done():
		s.logger.Info("gRPC server shutting down due to context cancellation")

	case sig := <-shutdown:
		s.logger.Info("gRPC server shutting down due to signal",
			zap.String("signal", sig.String()))
	}

	// Graceful shutdown with timeout
	s.logger.Info("Starting graceful shutdown...")

	// Create a context with timeout for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Channel to signal when graceful stop is complete
	gracefulStop := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(gracefulStop)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-gracefulStop:
		s.logger.Info("gRPC server stopped gracefully")
		return nil

	case <-shutdownCtx.Done():
		s.logger.Warn("Graceful shutdown timeout, forcing stop")
		s.server.Stop()
		return nil
	}
}
