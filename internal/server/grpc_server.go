package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"database/sql"

	"github.com/jia-app/paymentservice/internal/auth"
	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/ratelimit"
	"github.com/jia-app/paymentservice/internal/server/interceptors"
	"github.com/redis/go-redis/v9"
)

// GRPCServer represents a gRPC server
type GRPCServer struct {
	server       *grpc.Server
	config       *config.Config
	logger       *zap.Logger
	healthServer *health.Server
	db           *sql.DB
	redisClient  *redis.Client
	rateLimiter  ratelimit.RateLimiter
}

// NewGRPCServer creates a new gRPC server instance with all interceptors
func NewGRPCServer(cfg *config.Config, db *sql.DB, redisClient *redis.Client, validator auth.Validator) *GRPCServer {
	// Get logger instance
	logger := log.L(context.Background())

	// Create interceptors
	authInterceptor := interceptors.NewAuthInterceptor(validator, cfg.Auth.WhitelistedMethods)
	loggingInterceptor := interceptors.NewLoggingInterceptor()
	errorHandlerInterceptor := interceptors.NewErrorHandlerInterceptor()

	// Create timeout interceptor with method-specific timeouts
	methodTimeouts := map[string]time.Duration{
		"/payment.v1.PaymentService/CreatePayment":         30 * time.Second,
		"/payment.v1.PaymentService/CreateCheckoutSession": 30 * time.Second,
		"/payment.v1.PaymentService/PaymentSuccessWebhook": 10 * time.Second,
	}
	timeoutInterceptor := interceptors.NewTimeoutInterceptor(15*time.Second, methodTimeouts)

	// Initialize rate limiter if Redis is available
	var rateLimiter ratelimit.RateLimiter
	if redisClient != nil {
		rateLimiter = ratelimit.NewRedisRateLimiter(redisClient, logger)
		logger.Info("Rate limiter initialized with Redis")
	} else {
		logger.Warn("Redis not available, rate limiting disabled")
	}

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

	// Setup TLS credentials if enabled - CRITICAL: Fail if TLS setup fails
	var creds credentials.TransportCredentials
	if cfg.GRPC.TLSEnabled {
		tlsCreds, err := setupTLS(cfg.GRPC)
		if err != nil {
			logger.Fatal("Failed to setup TLS - refusing to start server without TLS", zap.Error(err))
		}
		creds = tlsCreds
		logger.Info("TLS enabled for gRPC server",
			zap.String("cert_file", cfg.GRPC.CertFile),
			zap.String("key_file", cfg.GRPC.KeyFile))
	} else {
		logger.Warn("TLS disabled for gRPC server - not recommended for production")
	}

	// Create server with interceptor chain
	var serverOpts []grpc.ServerOption

	// Build interceptor chain
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		otelgrpc.UnaryServerInterceptor(),
		grpc_recovery.UnaryServerInterceptor(recoveryOpts...),
		grpc_zap.UnaryServerInterceptor(logger, zapOpts...),
		timeoutInterceptor.Unary(),
		authInterceptor.Unary(),
		errorHandlerInterceptor.Unary(),
		loggingInterceptor.Unary(),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		otelgrpc.StreamServerInterceptor(),
		grpc_recovery.StreamServerInterceptor(recoveryOpts...),
		grpc_zap.StreamServerInterceptor(logger, zapOpts...),
		timeoutInterceptor.Stream(),
		authInterceptor.Stream(),
		errorHandlerInterceptor.Stream(),
		loggingInterceptor.Stream(),
	}

	// Add rate limiting interceptor if available
	if rateLimiter != nil {
		rateLimitConfig := ratelimit.DefaultConfig()
		rateLimitInterceptor := ratelimit.UnaryServerInterceptor(rateLimiter, rateLimitConfig)
		unaryInterceptors = append(unaryInterceptors, rateLimitInterceptor)
		logger.Info("Rate limiting interceptor added")
	}

	serverOpts = append(serverOpts,
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
	)

	// Add TLS credentials if configured
	if creds != nil {
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	server := grpc.NewServer(serverOpts...)

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
		db:           db,
		redisClient:  redisClient,
		rateLimiter:  rateLimiter,
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

	// Check Redis health
	redisHealthy := s.checkRedis(ctx)

	// Set overall health status
	if dbHealthy && redisHealthy {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		s.logger.Debug("All dependencies healthy, setting status to SERVING")
	} else {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.logger.Warn("Dependencies unhealthy, setting status to NOT_SERVING",
			zap.Bool("db_healthy", dbHealthy),
			zap.Bool("redis_healthy", redisHealthy))
	}
}

// checkDatabase checks if the database is reachable
func (s *GRPCServer) checkDatabase(ctx context.Context) bool {
	if s.db == nil {
		return false
	}

	if err := s.db.PingContext(ctx); err != nil {
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

// setupTLS configures TLS credentials for the gRPC server
func setupTLS(cfg config.GRPCConfig) (credentials.TransportCredentials, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Validate server certificate
	if err := validateServerCertificate(cert); err != nil {
		return nil, fmt.Errorf("server certificate validation failed: %w", err)
	}

	// Create TLS config with enhanced security
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13, // Use TLS 1.3 when available
		CipherSuites: []uint16{
			// TLS 1.2 cipher suites (in order of preference)
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
		// Enable session resumption
		SessionTicketsDisabled: false,
	}

	// If client CA file is provided, enable mTLS
	if cfg.ClientCAFile != "" {
		caCert, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse client CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		// Enhanced client certificate validation
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided")
			}

			// Parse client certificate
			clientCert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse client certificate: %w", err)
			}

			// Validate client certificate
			return validateClientCertificate(clientCert)
		}
	}

	return credentials.NewTLS(tlsConfig), nil
}

// validateServerCertificate validates the server certificate
func validateServerCertificate(cert tls.Certificate) error {
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("no server certificate found")
	}

	// Parse the server certificate
	serverCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse server certificate: %w", err)
	}

	// Check certificate validity period
	now := time.Now()
	if now.After(serverCert.NotAfter) {
		return fmt.Errorf("server certificate has expired")
	}
	if now.Before(serverCert.NotBefore) {
		return fmt.Errorf("server certificate is not yet valid")
	}

	// Check key usage
	if serverCert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("server certificate does not have digital signature key usage")
	}
	if serverCert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
		return fmt.Errorf("server certificate does not have key encipherment key usage")
	}

	// Check extended key usage for server authentication
	hasServerAuth := false
	for _, extKeyUsage := range serverCert.ExtKeyUsage {
		if extKeyUsage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		return fmt.Errorf("server certificate does not have server authentication extended key usage")
	}

	return nil
}

// validateClientCertificate validates the client certificate
func validateClientCertificate(cert *x509.Certificate) error {
	// Check certificate validity period
	now := time.Now()
	if now.After(cert.NotAfter) {
		return fmt.Errorf("client certificate has expired")
	}
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("client certificate is not yet valid")
	}

	// Check key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("client certificate does not have digital signature key usage")
	}

	// Check extended key usage for client authentication
	hasClientAuth := false
	for _, extKeyUsage := range cert.ExtKeyUsage {
		if extKeyUsage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}
	if !hasClientAuth {
		return fmt.Errorf("client certificate does not have client authentication extended key usage")
	}

	return nil
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
