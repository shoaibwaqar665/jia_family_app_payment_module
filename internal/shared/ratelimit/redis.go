package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RedisLimiter implements a Redis-based rate limiter using sliding window
type RedisLimiter struct {
	client      *redis.Client
	windowSize  time.Duration
	maxRequests int
	keyPrefix   string
}

// NewRedisLimiter creates a new Redis-based rate limiter
func NewRedisLimiter(client *redis.Client, windowSize time.Duration, maxRequests int) *RedisLimiter {
	return &RedisLimiter{
		client:      client,
		windowSize:  windowSize,
		maxRequests: maxRequests,
		keyPrefix:   "rate_limit",
	}
}

// Allow checks if a request is allowed using Redis sliding window
func (rl *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", rl.keyPrefix, key)
	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	// Use Redis pipeline for atomic operations
	pipe := rl.client.Pipeline()

	// Remove expired entries
	pipe.ZRemRangeByScore(ctx, redisKey, "0", strconv.FormatInt(windowStart.UnixNano(), 10))

	// Count current requests
	countCmd := pipe.ZCard(ctx, redisKey)

	// Add current request
	pipe.ZAdd(ctx, redisKey, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: now.UnixNano(),
	})

	// Set expiration
	pipe.Expire(ctx, redisKey, rl.windowSize)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.L(ctx).Error("Redis rate limiter error", zap.Error(err))
		return false, err
	}

	// Check count
	count, err := countCmd.Result()
	if err != nil {
		return false, err
	}

	return count < int64(rl.maxRequests), nil
}

// Reset clears the rate limit for a key
func (rl *RedisLimiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s:%s", rl.keyPrefix, key)
	return rl.client.Del(ctx, redisKey).Err()
}

// RedisTokenBucket implements a Redis-based token bucket rate limiter
type RedisTokenBucket struct {
	client     *redis.Client
	capacity   int
	refillRate int
	keyPrefix  string
}

// NewRedisTokenBucket creates a new Redis-based token bucket
func NewRedisTokenBucket(client *redis.Client, capacity, refillRate int) *RedisTokenBucket {
	return &RedisTokenBucket{
		client:     client,
		capacity:   capacity,
		refillRate: refillRate,
		keyPrefix:  "token_bucket",
	}
}

// Allow checks if a request is allowed using Redis token bucket
func (rtb *RedisTokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("%s:%s", rtb.keyPrefix, key)
	now := time.Now()

	// Lua script for atomic token bucket operations
	script := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add
		local elapsed = now - last_refill
		local tokens_to_add = math.floor(elapsed * refill_rate)
		
		-- Refill tokens
		tokens = math.min(capacity, tokens + tokens_to_add)
		
		-- Check if we can consume a token
		if tokens > 0 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 3600) -- Expire after 1 hour
			return 1
		else
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 3600)
			return 0
		end
	`

	result, err := rtb.client.Eval(ctx, script, []string{redisKey},
		rtb.capacity, rtb.refillRate, now.Unix()).Result()
	if err != nil {
		log.L(ctx).Error("Redis token bucket error", zap.Error(err))
		return false, err
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from Redis script")
	}

	return allowed == 1, nil
}

// Reset resets the token bucket for a key
func (rtb *RedisTokenBucket) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("%s:%s", rtb.keyPrefix, key)
	return rtb.client.Del(ctx, redisKey).Err()
}

// RedisFixedWindow implements a Redis-based fixed window rate limiter
type RedisFixedWindow struct {
	client      *redis.Client
	windowSize  time.Duration
	maxRequests int
	keyPrefix   string
}

// NewRedisFixedWindow creates a new Redis-based fixed window rate limiter
func NewRedisFixedWindow(client *redis.Client, windowSize time.Duration, maxRequests int) *RedisFixedWindow {
	return &RedisFixedWindow{
		client:      client,
		windowSize:  windowSize,
		maxRequests: maxRequests,
		keyPrefix:   "fixed_window",
	}
}

// Allow checks if a request is allowed using Redis fixed window
func (rfw *RedisFixedWindow) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	windowStart := now.Truncate(rfw.windowSize)
	redisKey := fmt.Sprintf("%s:%s:%d", rfw.keyPrefix, key, windowStart.Unix())

	// Lua script for atomic fixed window operations
	script := `
		local key = KEYS[1]
		local max_requests = tonumber(ARGV[1])
		local window_size = tonumber(ARGV[2])
		
		local current = redis.call('GET', key)
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end
		
		if current < max_requests then
			redis.call('INCR', key)
			redis.call('EXPIRE', key, window_size)
			return 1
		else
			return 0
		end
	`

	result, err := rfw.client.Eval(ctx, script, []string{redisKey},
		rfw.maxRequests, int(rfw.windowSize.Seconds())).Result()
	if err != nil {
		log.L(ctx).Error("Redis fixed window error", zap.Error(err))
		return false, err
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from Redis script")
	}

	return allowed == 1, nil
}

// Reset resets the fixed window for a key
func (rfw *RedisFixedWindow) Reset(ctx context.Context, key string) error {
	now := time.Now()
	windowStart := now.Truncate(rfw.windowSize)
	redisKey := fmt.Sprintf("%s:%s:%d", rfw.keyPrefix, key, windowStart.Unix())
	return rfw.client.Del(ctx, redisKey).Err()
}

// RateLimitConfigs provides predefined rate limit configurations
var RateLimitConfigs = struct {
	// Strict limits for sensitive operations
	Strict RateLimitConfig

	// Moderate limits for normal operations
	Moderate RateLimitConfig

	// Lenient limits for read operations
	Lenient RateLimitConfig

	// Bulk operation limits
	Bulk RateLimitConfig
}{
	Strict: RateLimitConfig{
		Limiter:      NewTokenBucket(10, 1), // 10 tokens, refill 1 per second
		KeyExtractor: UserIDKeyExtractor,
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.ResourceExhausted, "strict rate limit exceeded")
		},
	},

	Moderate: RateLimitConfig{
		Limiter:      NewTokenBucket(100, 10), // 100 tokens, refill 10 per second
		KeyExtractor: UserIDKeyExtractor,
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		},
	},

	Lenient: RateLimitConfig{
		Limiter:      NewTokenBucket(1000, 100), // 1000 tokens, refill 100 per second
		KeyExtractor: UserIDKeyExtractor,
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		},
	},

	Bulk: RateLimitConfig{
		Limiter:      NewTokenBucket(5, 1), // 5 tokens, refill 1 per second
		KeyExtractor: UserIDKeyExtractor,
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.ResourceExhausted, "bulk operation rate limit exceeded")
		},
	},
}
