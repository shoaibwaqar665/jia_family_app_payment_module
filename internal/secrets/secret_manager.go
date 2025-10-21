package secrets

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// SecretManager defines the interface for secret management
type SecretManager interface {
	GetSecret(ctx context.Context, key string) (string, error)
	GetSecretWithDefault(ctx context.Context, key, defaultValue string) (string, error)
	RefreshSecret(ctx context.Context, key string) error
	SetSecret(ctx context.Context, key, value string) error
}

// EnvironmentSecretManager implements secret management using environment variables
type EnvironmentSecretManager struct {
	prefix string
}

// NewEnvironmentSecretManager creates a new environment-based secret manager
func NewEnvironmentSecretManager(prefix string) *EnvironmentSecretManager {
	return &EnvironmentSecretManager{
		prefix: prefix,
	}
}

// GetSecret retrieves a secret from environment variables
func (e *EnvironmentSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	envKey := e.getEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return "", fmt.Errorf("secret not found: %s", key)
	}
	return value, nil
}

// GetSecretWithDefault retrieves a secret with a default value
func (e *EnvironmentSecretManager) GetSecretWithDefault(ctx context.Context, key, defaultValue string) (string, error) {
	envKey := e.getEnvKey(key)
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

// RefreshSecret refreshes a secret (no-op for environment variables)
func (e *EnvironmentSecretManager) RefreshSecret(ctx context.Context, key string) error {
	// Environment variables don't need refreshing
	return nil
}

// SetSecret sets a secret (no-op for environment variables)
func (e *EnvironmentSecretManager) SetSecret(ctx context.Context, key, value string) error {
	// Cannot set environment variables at runtime
	return fmt.Errorf("cannot set environment variable at runtime")
}

// getEnvKey converts a secret key to an environment variable name
func (e *EnvironmentSecretManager) getEnvKey(key string) string {
	if e.prefix == "" {
		return strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	}
	return strings.ToUpper(e.prefix + "_" + strings.ReplaceAll(key, ".", "_"))
}

// FileSecretManager implements secret management using files
type FileSecretManager struct {
	basePath string
}

// NewFileSecretManager creates a new file-based secret manager
func NewFileSecretManager(basePath string) *FileSecretManager {
	return &FileSecretManager{
		basePath: basePath,
	}
}

// GetSecret retrieves a secret from a file
func (f *FileSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	filePath := f.getFilePath(key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %s: %w", key, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// GetSecretWithDefault retrieves a secret with a default value
func (f *FileSecretManager) GetSecretWithDefault(ctx context.Context, key, defaultValue string) (string, error) {
	value, err := f.GetSecret(ctx, key)
	if err != nil {
		return defaultValue, nil
	}
	return value, nil
}

// RefreshSecret refreshes a secret (no-op for files)
func (f *FileSecretManager) RefreshSecret(ctx context.Context, key string) error {
	// Files don't need refreshing
	return nil
}

// SetSecret sets a secret to a file
func (f *FileSecretManager) SetSecret(ctx context.Context, key, value string) error {
	filePath := f.getFilePath(key)
	err := os.WriteFile(filePath, []byte(value), 0600) // Read/write for owner only
	if err != nil {
		return fmt.Errorf("failed to write secret file %s: %w", key, err)
	}
	return nil
}

// getFilePath converts a secret key to a file path
func (f *FileSecretManager) getFilePath(key string) string {
	return f.basePath + "/" + key
}

// CachedSecretManager implements secret management with caching
type CachedSecretManager struct {
	underlying SecretManager
	cache      map[string]cachedSecret
	ttl        time.Duration
}

type cachedSecret struct {
	value     string
	expiresAt time.Time
}

// NewCachedSecretManager creates a new cached secret manager
func NewCachedSecretManager(underlying SecretManager, ttl time.Duration) *CachedSecretManager {
	return &CachedSecretManager{
		underlying: underlying,
		cache:      make(map[string]cachedSecret),
		ttl:        ttl,
	}
}

// GetSecret retrieves a cached secret
func (c *CachedSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	// Check cache first
	if cached, exists := c.cache[key]; exists {
		if time.Now().Before(cached.expiresAt) {
			return cached.value, nil
		}
		// Cache expired, remove it
		delete(c.cache, key)
	}

	// Get from underlying manager
	value, err := c.underlying.GetSecret(ctx, key)
	if err != nil {
		return "", err
	}

	// Cache the value
	c.cache[key] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}

	return value, nil
}

// GetSecretWithDefault retrieves a cached secret with a default value
func (c *CachedSecretManager) GetSecretWithDefault(ctx context.Context, key, defaultValue string) (string, error) {
	// Check cache first
	if cached, exists := c.cache[key]; exists {
		if time.Now().Before(cached.expiresAt) {
			return cached.value, nil
		}
		// Cache expired, remove it
		delete(c.cache, key)
	}

	// Get from underlying manager
	value, err := c.underlying.GetSecretWithDefault(ctx, key, defaultValue)
	if err != nil {
		return "", err
	}

	// Cache the value
	c.cache[key] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}

	return value, nil
}

// RefreshSecret refreshes a secret in the cache
func (c *CachedSecretManager) RefreshSecret(ctx context.Context, key string) error {
	// Remove from cache
	delete(c.cache, key)

	// Refresh from underlying manager
	return c.underlying.RefreshSecret(ctx, key)
}

// SetSecret sets a secret and updates the cache
func (c *CachedSecretManager) SetSecret(ctx context.Context, key, value string) error {
	// Set in underlying manager
	err := c.underlying.SetSecret(ctx, key, value)
	if err != nil {
		return err
	}

	// Update cache
	c.cache[key] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}

	return nil
}

// CompositeSecretManager tries multiple secret managers in order
type CompositeSecretManager struct {
	managers []SecretManager
}

// NewCompositeSecretManager creates a new composite secret manager
func NewCompositeSecretManager(managers ...SecretManager) *CompositeSecretManager {
	return &CompositeSecretManager{
		managers: managers,
	}
}

// GetSecret tries to get a secret from each manager in order
func (c *CompositeSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	var lastErr error
	for _, manager := range c.managers {
		value, err := manager.GetSecret(ctx, key)
		if err == nil {
			return value, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("secret not found in any manager: %w", lastErr)
}

// GetSecretWithDefault tries to get a secret with a default value
func (c *CompositeSecretManager) GetSecretWithDefault(ctx context.Context, key, defaultValue string) (string, error) {
	value, err := c.GetSecret(ctx, key)
	if err != nil {
		return defaultValue, nil
	}
	return value, nil
}

// RefreshSecret refreshes a secret in all managers
func (c *CompositeSecretManager) RefreshSecret(ctx context.Context, key string) error {
	var lastErr error
	for _, manager := range c.managers {
		if err := manager.RefreshSecret(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// SetSecret sets a secret in all managers
func (c *CompositeSecretManager) SetSecret(ctx context.Context, key, value string) error {
	var lastErr error
	for _, manager := range c.managers {
		if err := manager.SetSecret(ctx, key, value); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
