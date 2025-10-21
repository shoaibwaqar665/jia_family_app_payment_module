package retry

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts   int           // Maximum number of attempts
	InitialDelay  time.Duration // Initial delay between retries
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Backoff multiplier
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// Do executes a function with retry logic
func Do(ctx context.Context, config Config, logger *zap.Logger, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after retry",
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", config.MaxAttempts))
			}
			return nil
		}

		lastErr = err
		logger.Warn("Operation failed, will retry",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", config.MaxAttempts))

		// Don't sleep after the last attempt
		if attempt < config.MaxAttempts {
			delay := calculateDelay(config, attempt)

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled: %w", ctx.Err())
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(config Config, attempt int) time.Duration {
	// Calculate exponential backoff delay
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to determine if an error is retryable
	// For example, network errors, timeouts, 5xx errors, etc.
	// This is a simple implementation - you may want to expand this
	errStr := err.Error()

	// Check for common retryable error patterns
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"503",
		"502",
		"504",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) && containsIgnoreCaseHelper(s, substr)))
}

func containsIgnoreCaseHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
