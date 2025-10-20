// Package app provides the main application structure with service mesh integration
// This is an example integration file showing how to use the service mesh components
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/app/server"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/jia-app/paymentservice/internal/shared/metrics"
	"github.com/jia-app/paymentservice/internal/shared/services"
)

// App represents the application
type App struct {
	config           *config.Config
	logger           *zap.Logger
	dbPool           *pgxpool.Pool
	redisClient      *redis.Client
	grpcServer       *server.GRPCServer
	metricsCollector *metrics.MetricsCollector
	serviceManager   *services.ServiceManager
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

	// Initialize database pool
	dbPool, err := initializeDatabase(cfg)
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

	// Initialize metrics collector
	metricsCollector := metrics.NewMetricsCollector()

	// Initialize service discovery and service manager if enabled
	var serviceManager *services.ServiceManager
	if cfg.ServiceMesh.Enabled {
		logger.Info("Service mesh enabled, initializing service discovery",
			zap.String("spiffe_id", cfg.ServiceMesh.SpiffeID))

		serviceManager, err = services.NewServiceManager(cfg, logger)
		if err != nil {
			logger.Warn("Failed to initialize service manager, continuing without external services",
				zap.Error(err))
		} else {
			// Initialize service clients
			if err := serviceManager.Initialize(context.Background()); err != nil {
				logger.Warn("Failed to initialize service clients, continuing without external services",
					zap.Error(err))
			} else {
				logger.Info("Service manager initialized successfully")
			}
		}
	}

	// NOTE: This is an example integration file
	// In production, you would initialize your repositories, use cases, and services here
	// For now, we'll just initialize the gRPC server and service manager

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(cfg, dbPool, redisClient, metricsCollector)

	// Log service mesh configuration
	if cfg.ServiceMesh.Enabled {
		allowedSpiffeIDs := []string{
			cfg.ExternalServices.ContactService.SpiffeID,
			cfg.ExternalServices.FamilyService.SpiffeID,
			cfg.ExternalServices.DocumentService.SpiffeID,
		}
		logger.Info("Service mesh enabled with spiffe authentication",
			zap.Strings("allowed_spiffe_ids", allowedSpiffeIDs))
	} else {
		logger.Info("Service mesh disabled, using standard JWT authentication")
	}

	// TODO: Initialize and register payment service
	// paymentService := transport.NewPaymentService(...)
	// grpcServer.RegisterPaymentService(paymentService)

	return &App{
		config:           cfg,
		logger:           logger,
		dbPool:           dbPool,
		redisClient:      redisClient,
		grpcServer:       grpcServer,
		metricsCollector: metricsCollector,
		serviceManager:   serviceManager,
	}, nil
}

// Run starts the application
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting payment service application")

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

	// Close service manager
	if a.serviceManager != nil {
		if err := a.serviceManager.Close(); err != nil {
			a.logger.Error("Failed to close service manager", zap.Error(err))
		}
	}

	// Close Redis client
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			a.logger.Error("Failed to close redis client", zap.Error(err))
		}
	}

	// Close database pool
	if a.dbPool != nil {
		a.dbPool.Close()
	}

	a.logger.Info("Application shutdown complete")
	return nil
}

// initializeDatabase initializes the database connection pool
func initializeDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
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

// GetServiceManager returns the service manager
func (a *App) GetServiceManager() *services.ServiceManager {
	return a.serviceManager
}

// GetMetricsCollector returns the metrics collector
func (a *App) GetMetricsCollector() *metrics.MetricsCollector {
	return a.metricsCollector
}
