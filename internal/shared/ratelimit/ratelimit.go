package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Reset(ctx context.Context, key string) error
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity   int          // Maximum number of tokens
	tokens     int          // Current number of tokens
	refillRate int          // Tokens added per second
	lastRefill time.Time    // Last time tokens were refilled
	mutex      sync.RWMutex // Protects the bucket state
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	now := time.Now()

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate

	// Add tokens up to capacity
	tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true, nil
	}

	return false, nil
}

// Reset resets the token bucket to full capacity
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	tb.tokens = tb.capacity
	tb.lastRefill = time.Now()
	return nil
}

// SlidingWindow implements a sliding window rate limiter
type SlidingWindow struct {
	windowSize  time.Duration // Size of the time window
	maxRequests int           // Maximum requests allowed in the window
	requests    []time.Time   // Timestamps of recent requests
	mutex       sync.RWMutex  // Protects the window state
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(windowSize time.Duration, maxRequests int) *SlidingWindow {
	return &SlidingWindow{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0, maxRequests),
	}
}

// Allow checks if a request is allowed within the sliding window
func (sw *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-sw.windowSize)

	// Remove old requests outside the window
	var validRequests []time.Time
	for _, reqTime := range sw.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	sw.requests = validRequests

	// Check if we're under the limit
	if len(sw.requests) < sw.maxRequests {
		sw.requests = append(sw.requests, now)
		return true, nil
	}

	return false, nil
}

// Reset clears all requests from the sliding window
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	sw.requests = sw.requests[:0]
	return nil
}

// FixedWindow implements a fixed window rate limiter
type FixedWindow struct {
	windowSize    time.Duration // Size of the time window
	maxRequests   int           // Maximum requests allowed in the window
	currentWindow time.Time     // Start of current window
	requestCount  int           // Number of requests in current window
	mutex         sync.RWMutex  // Protects the window state
}

// NewFixedWindow creates a new fixed window rate limiter
func NewFixedWindow(windowSize time.Duration, maxRequests int) *FixedWindow {
	return &FixedWindow{
		windowSize:    windowSize,
		maxRequests:   maxRequests,
		currentWindow: time.Now().Truncate(windowSize),
		requestCount:  0,
	}
}

// Allow checks if a request is allowed within the current fixed window
func (fw *FixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	now := time.Now()
	currentWindowStart := now.Truncate(fw.windowSize)

	// If we're in a new window, reset the counter
	if currentWindowStart.After(fw.currentWindow) {
		fw.currentWindow = currentWindowStart
		fw.requestCount = 0
	}

	// Check if we're under the limit
	if fw.requestCount < fw.maxRequests {
		fw.requestCount++
		return true, nil
	}

	return false, nil
}

// Reset resets the fixed window counter
func (fw *FixedWindow) Reset(ctx context.Context, key string) error {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	fw.requestCount = 0
	fw.currentWindow = time.Now().Truncate(fw.windowSize)
	return nil
}

// KeyExtractor extracts a key from the context for rate limiting
type KeyExtractor func(ctx context.Context) (string, error)

// UserIDKeyExtractor extracts user ID from context
func UserIDKeyExtractor(ctx context.Context) (string, error) {
	// This would typically extract user ID from JWT token or metadata
	// For now, return a placeholder
	return "default_user", nil
}

// IPKeyExtractor extracts IP address from context
func IPKeyExtractor(ctx context.Context) (string, error) {
	// This would typically extract IP from gRPC metadata
	// For now, return a placeholder
	return "default_ip", nil
}

// CompositeKeyExtractor combines multiple key extractors
func CompositeKeyExtractor(extractors ...KeyExtractor) KeyExtractor {
	return func(ctx context.Context) (string, error) {
		var keys []string
		for _, extractor := range extractors {
			key, err := extractor(ctx)
			if err != nil {
				return "", err
			}
			keys = append(keys, key)
		}
		return fmt.Sprintf("%v", keys), nil
	}
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	Limiter      Limiter
	KeyExtractor KeyExtractor
	ErrorHandler func(ctx context.Context, err error) error
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limiter:      NewTokenBucket(100, 10), // 100 tokens, refill 10 per second
		KeyExtractor: UserIDKeyExtractor,
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		},
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for rate limiting
func UnaryServerInterceptor(config RateLimitConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract key for rate limiting
		key, err := config.KeyExtractor(ctx)
		if err != nil {
			log.L(ctx).Error("Failed to extract rate limit key", zap.Error(err))
			return handler(ctx, req) // Continue without rate limiting
		}

		// Check rate limit
		allowed, err := config.Limiter.Allow(ctx, key)
		if err != nil {
			log.L(ctx).Error("Rate limiter error", zap.Error(err))
			return handler(ctx, req) // Continue on limiter error
		}

		if !allowed {
			log.L(ctx).Warn("Rate limit exceeded",
				zap.String("key", key),
				zap.String("method", info.FullMethod))
			return nil, config.ErrorHandler(ctx, fmt.Errorf("rate limit exceeded"))
		}

		// Proceed with the request
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for rate limiting
func StreamServerInterceptor(config RateLimitConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// Extract key for rate limiting
		key, err := config.KeyExtractor(ctx)
		if err != nil {
			log.L(ctx).Error("Failed to extract rate limit key", zap.Error(err))
			return handler(srv, ss) // Continue without rate limiting
		}

		// Check rate limit
		allowed, err := config.Limiter.Allow(ctx, key)
		if err != nil {
			log.L(ctx).Error("Rate limiter error", zap.Error(err))
			return handler(srv, ss) // Continue on limiter error
		}

		if !allowed {
			log.L(ctx).Warn("Rate limit exceeded",
				zap.String("key", key),
				zap.String("method", info.FullMethod))
			return config.ErrorHandler(ctx, fmt.Errorf("rate limit exceeded"))
		}

		// Proceed with the stream
		return handler(srv, ss)
	}
}

// MultiLimiter implements multiple rate limiters with different rules
type MultiLimiter struct {
	limiters []Limiter
}

// NewMultiLimiter creates a new multi-limiter
func NewMultiLimiter(limiters ...Limiter) *MultiLimiter {
	return &MultiLimiter{
		limiters: limiters,
	}
}

// Allow checks if a request is allowed by all limiters
func (ml *MultiLimiter) Allow(ctx context.Context, key string) (bool, error) {
	for _, limiter := range ml.limiters {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

// Reset resets all limiters
func (ml *MultiLimiter) Reset(ctx context.Context, key string) error {
	for _, limiter := range ml.limiters {
		if err := limiter.Reset(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
