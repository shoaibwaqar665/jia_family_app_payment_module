package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/log"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	redis  RedisClient
	logger *zap.Logger
}

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(redis RedisClient, logger *zap.Logger) *RedisRateLimiter {
	return &RedisRateLimiter{
		redis:  redis,
		logger: logger,
	}
}

// Allow checks if a request is allowed based on the rate limit
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	// Get current count
	count, err := r.redis.Incr(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to increment rate limit counter",
			zap.Error(err),
			zap.String("key", key))
		return false, fmt.Errorf("rate limit error: %w", err)
	}

	// Set expiration on first request
	if count == 1 {
		if err := r.redis.Expire(ctx, key, time.Minute).Err(); err != nil {
			r.logger.Error("Failed to set rate limit expiration",
				zap.Error(err),
				zap.String("key", key))
		}
	}

	// Check if limit exceeded (assuming 100 requests per minute)
	limit := 100
	return count <= int64(limit), nil
}

// Config holds rate limiting configuration
type Config struct {
	// Requests per minute per user
	RequestsPerMinute int
	// Requests per minute per endpoint
	RequestsPerEndpoint int
	// Enable rate limiting
	Enabled bool
}

// DefaultConfig returns a default rate limiting configuration
func DefaultConfig() Config {
	return Config{
		RequestsPerMinute:   100,
		RequestsPerEndpoint: 1000,
		Enabled:             true,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for rate limiting
func UnaryServerInterceptor(limiter RateLimiter, config Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !config.Enabled {
			return handler(ctx, req)
		}

		// Extract user ID from context (set by auth interceptor)
		userID := getUserIDFromContext(ctx)
		if userID == "" {
			// No user ID means unauthenticated request - allow it
			return handler(ctx, req)
		}

		// Create rate limit key based on user and endpoint
		key := fmt.Sprintf("ratelimit:%s:%s", userID, info.FullMethod)

		// Check rate limit
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			log.Warn(ctx, "Rate limit check failed, allowing request",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("method", info.FullMethod))
			return handler(ctx, req)
		}

		if !allowed {
			log.Warn(ctx, "Rate limit exceeded",
				zap.String("user_id", userID),
				zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded for user %s on %s", userID, info.FullMethod)
		}

		return handler(ctx, req)
	}
}

// getUserIDFromContext extracts the user ID from the context
func getUserIDFromContext(ctx context.Context) string {
	// Check if user ID is in context metadata (set by auth interceptor)
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		return userID
	}

	// Check if user ID is in the log context key
	if userID, ok := ctx.Value("userID").(string); ok && userID != "" {
		return userID
	}

	// Check if user ID is in gRPC metadata
	if userID, ok := ctx.Value("grpc.metadata.user_id").(string); ok && userID != "" {
		return userID
	}

	// Check if user ID is in gRPC metadata under "x-user-id" header
	if userID, ok := ctx.Value("grpc.metadata.x-user-id").(string); ok && userID != "" {
		return userID
	}

	// No user ID found - this means unauthenticated request
	return ""
}
