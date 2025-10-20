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

	"github.com/jackc/pgx/v5/pgxpool"
	paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
	"github.com/jia-app/paymentservice/internal/app/server/interceptors"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/jia-app/paymentservice/internal/shared/metrics"
	"github.com/jia-app/paymentservice/internal/shared/ratelimit"
	"github.com/redis/go-redis/v9"
)

// GRPCServer represents a gRPC server
type GRPCServer struct {
	server       *grpc.Server
	config       *config.Config
	logger       *zap.Logger
	healthServer *health.Server
	dbPool       *pgxpool.Pool
	redisClient  *redis.Client
}

// NewGRPCServer creates a new gRPC server instance with all interceptors
func NewGRPCServer(cfg *config.Config, dbPool *pgxpool.Pool, redisClient *redis.Client, metricsCollector *metrics.MetricsCollector) *GRPCServer {
	// Get logger instance
	logger := log.L(context.Background())

	// Create interceptors
	authInterceptor := interceptors.NewAuthInterceptor()
	loggingInterceptor := interceptors.NewLoggingInterceptor()

	// Create rate limiting interceptor
	rateLimitConfig := ratelimit.RateLimitConfigs.Moderate
	rateLimitInterceptor := ratelimit.UnaryServerInterceptor(rateLimitConfig)
	rateLimitStreamInterceptor := ratelimit.StreamServerInterceptor(rateLimitConfig)

	// Create metrics interceptor
	metricsInterceptor := metrics.NewMetricsInterceptor(metricsCollector)
	metricsUnaryInterceptor := metricsInterceptor.UnaryServerInterceptor()
	metricsStreamInterceptor := metricsInterceptor.StreamServerInterceptor()

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
			metricsUnaryInterceptor,
			rateLimitInterceptor,
			authInterceptor.Unary(),
			loggingInterceptor.Unary(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(recoveryOpts...),
			grpc_zap.StreamServerInterceptor(logger, zapOpts...),
			metricsStreamInterceptor,
			rateLimitStreamInterceptor,
			authInterceptor.Stream(),
			loggingInterceptor.Stream(),
		)),
	)

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	// Set initial health status to NOT_SERVING until dependencies are healthy
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	// Register reflection service for non-production environments
	env := os.Getenv("ENV")
	if env != "prod" && env != "production" {
		logger.Info("Registering gRPC reflection (non-production environment)")
		reflection.Register(server)
	}

	return &GRPCServer{
		server:       server,
		config:       cfg,
		logger:       logger,
		healthServer: healthServer,
		dbPool:       dbPool,
		redisClient:  redisClient,
	}
}

// RegisterService registers a gRPC service with the server
func (s *GRPCServer) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// RegisterPaymentService registers the payment service with the gRPC server
func (s *GRPCServer) RegisterPaymentService(paymentService paymentv1.PaymentServiceServer) {
	paymentv1.RegisterPaymentServiceServer(s.server, paymentService)
	s.logger.Info("Payment service registered with gRPC server")
}

// GetServer returns the underlying gRPC server
func (s *GRPCServer) GetServer() *grpc.Server {
	return s.server
}

// StartHealthMonitoring starts background health checks for dependencies
func (s *GRPCServer) StartHealthMonitoring(ctx context.Context) {
	go s.monitorHealth(ctx)

	// Wait for initial health check to complete
	go s.waitForInitialHealth(ctx)
}

// waitForInitialHealth waits for dependencies to be healthy before setting status to SERVING
func (s *GRPCServer) waitForInitialHealth(ctx context.Context) {
	// Wait a bit for initial connections to stabilize
	time.Sleep(2 * time.Second)

	// Check dependencies immediately
	s.checkDependencies()

	s.logger.Info("Initial health check completed")
}

// monitorHealth runs background health checks for DB and Redis
func (s *GRPCServer) monitorHealth(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	s.logger.Info("Starting health monitoring for dependencies")

	// Initial health check
	s.checkDependencies()

	// Periodic health checks
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Health monitoring stopped due to context cancellation")
			return
		case <-ticker.C:
			s.checkDependencies()
		}
	}
}

// checkDependencies checks the health of DB and Redis
func (s *GRPCServer) checkDependencies() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check database health
	dbHealthy := s.checkDatabase(ctx)

	// Check Redis health (optional)
	redisHealthy := s.checkRedis(ctx)
	redisOptional := s.redisClient == nil

	// Set overall health status
	// If Redis is nil (optional), only check database
	// If Redis is configured, both must be healthy
	allHealthy := dbHealthy && (redisOptional || redisHealthy)

	if allHealthy {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		s.logger.Debug("All dependencies healthy, setting status to SERVING",
			zap.Bool("db_healthy", dbHealthy),
			zap.Bool("redis_healthy", redisHealthy),
			zap.Bool("redis_optional", redisOptional))
	} else {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.logger.Warn("Dependencies unhealthy, setting status to NOT_SERVING",
			zap.Bool("db_healthy", dbHealthy),
			zap.Bool("redis_healthy", redisHealthy),
			zap.Bool("redis_optional", redisOptional))
	}
}

// checkDatabase checks if the database is reachable
func (s *GRPCServer) checkDatabase(ctx context.Context) bool {
	if s.dbPool == nil {
		return false
	}

	if err := s.dbPool.Ping(ctx); err != nil {
		s.logger.Debug("Database health check failed", zap.Error(err))
		return false
	}

	return true
}

// checkRedis checks if Redis is reachable
func (s *GRPCServer) checkRedis(ctx context.Context) bool {
	if s.redisClient == nil {
		return false
	}

	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		s.logger.Debug("Redis health check failed", zap.Error(err))
		return false
	}

	return true
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
