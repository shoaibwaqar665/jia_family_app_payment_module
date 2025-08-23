package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/redis/go-redis/v9"
)

// Cache represents a Redis cache implementation
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Redis cache instance
func NewCache(addr, password string, db int) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &Cache{client: client}, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	return c.client.Close()
}

// Set sets a key-value pair in the cache
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value from the cache
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

// Delete removes a key from the cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in the cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	return result > 0, nil
}

// SetNX sets a key-value pair only if the key doesn't exist
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	result, err := c.client.SetNX(ctx, key, data, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set key: %w", err)
	}

	return result, nil
}

// Incr increments a counter
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	result, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key: %w", err)
	}

	return result, nil
}

// Expire sets the expiration time for a key
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// GetEntitlement retrieves an entitlement from cache
func (c *Cache) GetEntitlement(ctx context.Context, userID, featureCode string) (*domain.Entitlement, bool, error) {
	key := fmt.Sprintf("entl:%s:%s", userID, featureCode)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Cache miss - not found
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get entitlement from cache: %w", err)
	}

	var entitlement domain.Entitlement
	if err := json.Unmarshal(data, &entitlement); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal entitlement: %w", err)
	}

	return &entitlement, true, nil
}

// SetEntitlement stores an entitlement in cache
func (c *Cache) SetEntitlement(ctx context.Context, ent domain.Entitlement, ttl time.Duration) error {
	key := fmt.Sprintf("entl:%s:%s", ent.UserID, ent.FeatureCode)

	// Default TTL to 2 minutes if not specified
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}

	data, err := json.Marshal(ent)
	if err != nil {
		return fmt.Errorf("failed to marshal entitlement: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set entitlement in cache: %w", err)
	}

	return nil
}

// SetEntitlementNotFound caches a negative result for an entitlement
// Uses a shorter TTL (10 seconds max) to avoid caching stale negative results
func (c *Cache) SetEntitlementNotFound(ctx context.Context, userID, featureCode string) error {
	key := fmt.Sprintf("entl:%s:%s", userID, featureCode)

	// Cache negative result for 10 seconds maximum
	ttl := 10 * time.Second

	// Use a special marker for negative results
	negativeResult := map[string]interface{}{
		"not_found": true,
		"cached_at": time.Now().Unix(),
	}

	data, err := json.Marshal(negativeResult)
	if err != nil {
		return fmt.Errorf("failed to marshal negative result: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set negative result in cache: %w", err)
	}

	return nil
}

// IsEntitlementNotFound checks if the cached value represents a negative result
func (c *Cache) IsEntitlementNotFound(ctx context.Context, userID, featureCode string) (bool, error) {
	key := fmt.Sprintf("entl:%s:%s", userID, featureCode)

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil // No cache entry
		}
		return false, fmt.Errorf("failed to check negative cache: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// If unmarshaling fails, it might be a valid entitlement, not a negative result
		return false, nil
	}

	// Check if this is a negative result marker
	if notFound, ok := result["not_found"].(bool); ok && notFound {
		return true, nil
	}

	return false, nil
}

// DeleteEntitlement removes an entitlement from cache
func (c *Cache) DeleteEntitlement(ctx context.Context, userID, featureCode string) error {
	key := fmt.Sprintf("entl:%s:%s", userID, featureCode)
	return c.client.Del(ctx, key).Err()
}
