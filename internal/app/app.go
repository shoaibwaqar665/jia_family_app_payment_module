package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/auth"
	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/cache"
	"github.com/jia-app/paymentservice/internal/circuitbreaker"
	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/events"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/metrics"
	"github.com/jia-app/paymentservice/internal/outbox"
	"github.com/jia-app/paymentservice/internal/repository/postgres"
	"github.com/jia-app/paymentservice/internal/server"
	"github.com/jia-app/paymentservice/internal/service"
	"github.com/jia-app/paymentservice/internal/tracing"
	paymentv1 "github.com/jia-app/paymentservice/proto/payment/v1"
	_ "github.com/lib/pq"
)

// App represents the application
type App struct {
	config         *config.Config
	logger         *zap.Logger
	db             *sql.DB
	redisClient    *redis.Client
	grpcServer     *server.GRPCServer
	metricsServer  *metrics.Server
	outboxWorker   *outbox.Worker
	tracingCleanup func()
}

// New creates a new application instance
func New(cfg *config.Config) (*App, error) {
	// Initialize logger
	if err := log.Init(cfg.Log.Level); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	logger := log.L(context.Background())

	logger.Info("Initializing payment service application",
		zap.String("app_name", cfg.AppName),
		zap.String("grpc_address", cfg.GRPC.Address))

	// Initialize tracing if enabled
	var tracingCleanup func()
	if cfg.Tracing.Enabled {
		tracingConfig := tracing.Config{
			ServiceName:    cfg.Tracing.ServiceName,
			ServiceVersion: cfg.Tracing.ServiceVersion,
			Environment:    cfg.Tracing.Environment,
			JaegerEndpoint: cfg.Tracing.JaegerEndpoint,
			SamplingRatio:  cfg.Tracing.SamplingRatio,
		}
		cleanup, err := tracing.Init(tracingConfig, logger)
		if err != nil {
			logger.Error("Failed to initialize tracing, continuing without tracing", zap.Error(err))
		} else {
			tracingCleanup = cleanup
			logger.Info("Tracing initialized successfully")
		}
	} else {
		logger.Info("Tracing disabled")
	}

	// Initialize database connection
	db, err := initializeDatabase(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis client (optional)
	redisClient, err := initializeRedis(cfg)
	if err != nil {
		logger.Warn("Redis initialization failed, continuing without Redis",
			zap.Error(err),
			zap.String("redis_addr", cfg.Redis.Addr))
		redisClient = nil
	}

	// Initialize repositories
	store, err := postgres.NewStoreWithDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	// Initialize cache
	var cacheClient *cache.Cache
	if redisClient != nil {
		cacheClient, err = cache.NewCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			logger.Warn("Failed to initialize cache, continuing without cache",
				zap.Error(err))
			cacheClient = nil
		}
	}

	// Initialize authentication validators
	var jwtValidator auth.Validator
	var spiffeValidator *auth.SPIFFEValidator
	var validator auth.Validator

	// Initialize JWT validator
	if cfg.Auth.PublicKeyPEM != "" {
		jwtVal, err := auth.NewJWTValidator(cfg.Auth.PublicKeyPEM)
		if err != nil {
			logger.Error("Failed to create JWT validator", zap.Error(err))
			return nil, fmt.Errorf("failed to create JWT validator: %w", err)
		}
		jwtValidator = jwtVal
		logger.Info("JWT validator initialized")
	} else {
		// Check if we're in production environment
		env := os.Getenv("ENV")
		if env == "prod" || env == "production" {
			logger.Error("JWT public key is required in production environment")
			return nil, fmt.Errorf("JWT public key is required in production environment")
		}
		logger.Warn("No JWT public key provided, using mock validator (not recommended for production)")
		jwtValidator = auth.NewMockValidator()
	}

	// Initialize SPIFFE validator if enabled
	if cfg.Auth.SPIFFEEnabled {
		spiffeVal, err := auth.NewSPIFFEValidator(cfg.Auth.SPIFFETrustDomain, cfg.Auth.SPIFFECACertPEM)
		if err != nil {
			logger.Error("Failed to create SPIFFE validator", zap.Error(err))
			return nil, fmt.Errorf("failed to create SPIFFE validator: %w", err)
		}
		spiffeValidator = spiffeVal
		logger.Info("SPIFFE validator initialized",
			zap.String("trust_domain", cfg.Auth.SPIFFETrustDomain),
			zap.Bool("ca_cert_provided", cfg.Auth.SPIFFECACertPEM != ""))
	}

	// Create hybrid validator that supports both JWT and SPIFFE
	validator = auth.NewHybridValidator(jwtValidator, spiffeValidator, cfg.Auth.SPIFFEEnabled)
	logger.Info("Hybrid authentication validator initialized",
		zap.Bool("spiffe_enabled", cfg.Auth.SPIFFEEnabled),
		zap.Bool("jwt_enabled", jwtValidator != nil))

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(cfg, db, redisClient, validator)

	// Initialize circuit breaker manager
	circuitBreakerManager := circuitbreaker.NewManager(logger)
	logger.Info("Circuit breaker manager initialized")

	// Initialize event publisher (using Kafka publisher if configured)
	var eventPublisher events.EntitlementPublisher
	if cfg.Events.Provider == "kafka" && len(cfg.Events.Brokers) > 0 {
		kafkaPublisher, err := events.NewKafkaPublisher(cfg.Events.Topic, cfg.Events.Brokers, logger)
		if err != nil {
			logger.Error("Failed to create Kafka publisher, falling back to NoopPublisher",
				zap.Error(err),
				zap.String("topic", cfg.Events.Topic),
				zap.Strings("brokers", cfg.Events.Brokers))
			eventPublisher = events.NoopPublisher{}
		} else {
			eventPublisher = kafkaPublisher
			logger.Info("Kafka event publisher initialized",
				zap.String("topic", cfg.Events.Topic),
				zap.Strings("brokers", cfg.Events.Brokers))
		}
	} else {
		eventPublisher = events.NoopPublisher{}
		logger.Info("Using NoopPublisher for events (no event broker configured)")
	}

	// Initialize billing provider
	stripeProvider := billing.NewStripeProvider(
		cfg.Billing.StripeSecret,
		cfg.Billing.StripePublishable,
		cfg.Billing.StripeWebhookSecret,
		"https://your-app.com/success",
		"https://your-app.com/cancel",
		logger,
	)

	// Initialize payment service
	paymentService := service.NewEnhancedPaymentService(
		cfg,
		store.Payment(),
		store.Plan(),
		store.Entitlement(),
		store.WebhookEvents(),
		store.Outbox(),
		cacheClient,
		eventPublisher,
		stripeProvider,
		circuitBreakerManager,
	)

	// Create gRPC handler wrapper
	grpcHandler := service.NewPaymentGRPCHandler(paymentService)

	// Register gRPC service
	paymentv1.RegisterPaymentServiceServer(grpcServer.GetServer(), grpcHandler)

	logger.Info("Enhanced payment service initialized and registered",
		zap.String("validator_type", fmt.Sprintf("%T", jwtValidator)),
		zap.Bool("circuit_breaker_enabled", true),
		zap.String("event_publisher", fmt.Sprintf("%T", eventPublisher)))

	// Initialize Prometheus metrics server
	metricsServer := metrics.NewServer(":9090", logger)

	// Initialize outbox worker if event publisher is not NoopPublisher
	var outboxWorker *outbox.Worker
	if _, ok := eventPublisher.(events.NoopPublisher); !ok {
		// Check if the event publisher also implements the general Publisher interface
		if publisher, ok := eventPublisher.(events.Publisher); ok {
			outboxWorker = outbox.NewWorker(
				store.Outbox(),
				publisher,
				logger,
				outbox.DefaultConfig(),
			)
			logger.Info("Outbox worker initialized")
		} else {
			logger.Warn("Event publisher does not implement Publisher interface, outbox worker not started")
		}
	}

	logger.Info("Payment service registered successfully")

	return &App{
		config:         cfg,
		logger:         logger,
		db:             db,
		redisClient:    redisClient,
		grpcServer:     grpcServer,
		metricsServer:  metricsServer,
		outboxWorker:   outboxWorker,
		tracingCleanup: tracingCleanup,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting payment service application")

	// Start Prometheus metrics server
	go func() {
		if err := a.metricsServer.Start(ctx); err != nil {
			a.logger.Error("Metrics server error", zap.Error(err))
		}
	}()

	// Start outbox worker if initialized
	if a.outboxWorker != nil {
		go func() {
			if err := a.outboxWorker.Start(ctx); err != nil {
				a.logger.Error("Outbox worker error", zap.Error(err))
			}
		}()
	}

	// Start health monitoring
	a.grpcServer.StartHealthMonitoring(ctx)

	// Start gRPC server
	if err := a.grpcServer.Serve(ctx); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down payment service application")

	// Shutdown tracing
	if a.tracingCleanup != nil {
		a.tracingCleanup()
		a.logger.Info("Tracing shutdown complete")
	}

	// Shutdown outbox worker
	if a.outboxWorker != nil {
		if err := a.outboxWorker.Stop(ctx); err != nil {
			a.logger.Error("Failed to stop outbox worker", zap.Error(err))
		}
	}

	// Shutdown metrics server
	if a.metricsServer != nil {
		if err := a.metricsServer.Shutdown(ctx); err != nil {
			a.logger.Error("Failed to shutdown metrics server", zap.Error(err))
		}
	}

	// Close Redis client
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			a.logger.Error("Failed to close redis client", zap.Error(err))
		}
	}

	// Close database connection
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error("Failed to close database connection", zap.Error(err))
		}
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

// initializeDatabase initializes the database connection
func initializeDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// initializeRedis initializes the Redis client
func initializeRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		DB:       cfg.Redis.DB,
		Password: cfg.Redis.Password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
