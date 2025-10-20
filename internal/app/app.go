package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/server"
)

// App represents the application
type App struct {
	config      *config.Config
	logger      *zap.Logger
	dbPool      *pgxpool.Pool
	redisClient *redis.Client
	grpcServer  *server.GRPCServer
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

	// Initialize gRPC server
	grpcServer := server.NewGRPCServer(cfg, dbPool, redisClient)

	return &App{
		config:      cfg,
		logger:      logger,
		dbPool:      dbPool,
		redisClient: redisClient,
		grpcServer:  grpcServer,
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
