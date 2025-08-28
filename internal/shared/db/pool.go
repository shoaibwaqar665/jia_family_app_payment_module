package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// Pool represents a PostgreSQL connection pool
type Pool struct {
	*pgxpool.Pool
	config *Config
}

// Config represents database pool configuration
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultConfig returns a default database configuration
func DefaultConfig() *Config {
	return &Config{
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: time.Minute * 30,
		HealthCheckPeriod: time.Minute,
	}
}

// NewPool creates a new PostgreSQL connection pool
func NewPool(ctx context.Context, config *Config) (*Pool, error) {
	// Parse connection string
	poolConfig, err := pgxpool.ParseConfig(config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = config.HealthCheckPeriod

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info(ctx, "Database pool created successfully",
		zap.Int32("max_conns", config.MaxConns),
		zap.Int32("min_conns", config.MinConns),
		zap.Duration("max_conn_lifetime", config.MaxConnLifetime),
		zap.Duration("max_conn_idle_time", config.MaxConnIdleTime))

	return &Pool{
		Pool:   pool,
		config: config,
	}, nil
}

// Health checks if the database pool is healthy
func (p *Pool) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return p.Ping(ctx)
}

// Stats returns pool statistics
func (p *Pool) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}

// Close closes the database pool
func (p *Pool) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
