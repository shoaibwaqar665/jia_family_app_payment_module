package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/app/server"
	"github.com/jia-app/paymentservice/internal/payment"
	"github.com/jia-app/paymentservice/internal/payment/repo/postgres"
	"github.com/jia-app/paymentservice/internal/payment/transport"
	"github.com/jia-app/paymentservice/internal/payment/usecase"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// BootstrapAndServe initializes all dependencies and starts the gRPC server
func BootstrapAndServe(ctx context.Context) error {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		return err
	}

	// Initialize logger with configuration
	if err := log.Init(cfg.Log.Level); err != nil {
		return err
	}

	// Use structured logging from here on
	log.Info(ctx, "Payment Service starting",
		zap.String("app_name", cfg.AppName),
		zap.String("grpc_address", cfg.GRPC.Address),
		zap.Int32("postgres_max_conns", cfg.Postgres.MaxConns),
		zap.String("redis_addr", cfg.Redis.Addr),
		zap.String("billing_provider", cfg.Billing.Provider),
		zap.String("events_provider", cfg.Events.Provider))

	// Initialize database connection
	log.Info(ctx, "Connecting to database...")
	dbConfig, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		log.Error(ctx, "Failed to parse database config", zap.Error(err))
		return err
	}

	// Set maximum connections
	dbConfig.MaxConns = cfg.Postgres.MaxConns

	dbPool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		log.Error(ctx, "Failed to create database pool", zap.Error(err))
		return err
	}
	defer dbPool.Close()

	// Fast fail on missing DB connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Error(ctx, "Failed to ping database - fast fail", zap.Error(err))
		return err
	}
	log.Info(ctx, "Database connection established", zap.Int32("max_conns", cfg.Postgres.MaxConns))

	// Initialize Redis connection
	log.Info(ctx, "Connecting to Redis...")
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		DB:       cfg.Redis.DB,
		Password: cfg.Redis.Password,
	})
	defer redisClient.Close()

	// Test Redis connection (optional)
	var cacheClient *cache.Cache
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Warn(ctx, "Redis not available, running without cache", zap.Error(err))
		redisClient = nil
		cacheClient = nil
	} else {
		log.Info(ctx, "Redis connection established")

		// Initialize cache with Redis client
		log.Info(ctx, "Initializing cache...")
		cacheClient, err = cache.NewCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			log.Warn(ctx, "Failed to create cache client, running without cache", zap.Error(err))
			cacheClient = nil
		} else {
			log.Info(ctx, "Cache initialized successfully")
		}
	}

	// Initialize repositories with database pool
	log.Info(ctx, "Initializing repositories...")
	repo, err := postgres.NewStoreWithPool(dbPool)
	if err != nil {
		log.Error(ctx, "Failed to create repository", zap.Error(err))
		return err
	}
	log.Info(ctx, "Repositories initialized successfully")

	// Initialize billing provider
	log.Info(ctx, "Initializing billing provider...")
	billingProvider, err := NewBillingProvider(ctx, cfg)
	if err != nil {
		log.Error(ctx, "Failed to initialize billing provider", zap.Error(err))
		return err
	}
	defer billingProvider.Close()

	// Initialize event publisher based on configuration
	log.Info(ctx, "Initializing event publisher...")
	var entitlementPublisher events.EntitlementPublisher
	if cfg.Events.Provider == "kafka" {
		logger := log.L(ctx)
		entitlementPublisher = events.NewKafkaPublisher(cfg.Events.Topic, logger)
		log.Info(ctx, "Using Kafka event publisher", zap.String("topic", cfg.Events.Topic))
	} else {
		entitlementPublisher = events.NoopPublisher{}
		log.Info(ctx, "Using Noop event publisher")
	}

	// Initialize payment service
	log.Info(ctx, "Initializing payment service...")

	// Pricing: in-memory rule store and calculator (pluggable later)
	ruleStore := payment.NewMemoryRuleStore()
	_ = payment.NewCalculator(ruleStore) // TODO: Use for price adjustments

	// Initialize use cases
	paymentUseCase := usecase.NewPaymentUseCase(repo.Payment())
	entitlementUseCase := usecase.NewEntitlementUseCase(repo.Entitlement(), cacheClient, entitlementPublisher)
	checkoutUseCase := usecase.NewCheckoutUseCase(repo.Plan(), repo.Entitlement(), repo.PricingZone(), cacheClient, entitlementPublisher)
	pricingZoneUseCase := usecase.NewPricingZoneUseCase(repo.PricingZone())

	paymentService := transport.NewPaymentService(
		cfg,
		paymentUseCase,
		entitlementUseCase,
		checkoutUseCase,
		pricingZoneUseCase,
		cacheClient,
		entitlementPublisher,
		billingProvider,
	)
	log.Info(ctx, "Payment service initialized successfully")

	// Initialize gRPC server with dependencies
	log.Info(ctx, "Initializing gRPC server...")
	grpcServer := server.NewGRPCServer(cfg, dbPool, redisClient)

	// Register payment service with gRPC server
	grpcServer.RegisterPaymentService(paymentService)
	log.Info(ctx, "Payment service registered with gRPC server")

	log.Info(ctx, "Payment Service initialized successfully")

	// Start health monitoring
	log.Info(ctx, "Starting health monitoring...")
	grpcServer.StartHealthMonitoring(ctx)

	// Start gRPC server (blocks until shutdown)
	log.Info(ctx, "Starting gRPC server...")
	if err := grpcServer.Serve(ctx); err != nil {
		log.Error(ctx, "gRPC server error", zap.Error(err))
		return err
	}

	log.Info(ctx, "Payment Service shutdown complete")
	return nil
}
