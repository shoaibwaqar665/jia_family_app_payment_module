package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jia-app/paymentservice/internal/shared/log"
	"go.uber.org/zap"
)

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *zap.Logger
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   log.L(context.Background()),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	// Create new circuit breaker with state change logging
	breaker := NewCircuitBreakerWithCallback(config, func(from, to State) {
		m.logger.Info("Circuit breaker state changed",
			zap.String("name", name),
			zap.String("from", from.String()),
			zap.String("to", to.String()))
	})

	m.breakers[name] = breaker
	return breaker
}

// Get returns an existing circuit breaker
func (m *Manager) Get(name string) (*CircuitBreaker, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	breaker, exists := m.breakers[name]
	return breaker, exists
}

// Remove removes a circuit breaker
func (m *Manager) Remove(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.breakers, name)
}

// List returns all circuit breaker names
func (m *Manager) List() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.breakers))
	for name := range m.breakers {
		names = append(names, name)
	}
	return names
}

// GetAllMetrics returns metrics for all circuit breakers
func (m *Manager) GetAllMetrics() map[string]Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metrics := make(map[string]Metrics)
	for name, breaker := range m.breakers {
		metrics[name] = breaker.GetMetrics()
	}
	return metrics
}

// ResetAll resets all circuit breakers
func (m *Manager) ResetAll() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, breaker := range m.breakers {
		breaker.Reset()
	}
}

// Reset resets a specific circuit breaker
func (m *Manager) Reset(name string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	breaker, exists := m.breakers[name]
	if !exists {
		return fmt.Errorf("circuit breaker %s not found", name)
	}

	breaker.Reset()
	return nil
}

// HealthCheck performs a health check on all circuit breakers
func (m *Manager) HealthCheck(ctx context.Context) map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	health := make(map[string]interface{})

	for name, breaker := range m.breakers {
		metrics := breaker.GetMetrics()
		health[name] = map[string]interface{}{
			"state":     metrics.State.String(),
			"failures":  metrics.Failures,
			"successes": metrics.Successes,
			"healthy":   metrics.State == StateClosed,
		}
	}

	return health
}

// Default circuit breaker configurations for common services
var (
	// StripeConfig is optimized for Stripe API calls
	StripeConfig = Config{
		MaxRequests:      5,
		Interval:         30 * time.Second,
		Timeout:          2 * time.Minute,
		MaxFailures:      3,
		SuccessThreshold: 2,
	}

	// DatabaseConfig is optimized for database operations
	DatabaseConfig = Config{
		MaxRequests:      10,
		Interval:         10 * time.Second,
		Timeout:          30 * time.Second,
		MaxFailures:      5,
		SuccessThreshold: 3,
	}

	// RedisConfig is optimized for Redis operations
	RedisConfig = Config{
		MaxRequests:      20,
		Interval:         5 * time.Second,
		Timeout:          15 * time.Second,
		MaxFailures:      3,
		SuccessThreshold: 2,
	}

	// ExternalAPIConfig is optimized for external API calls
	ExternalAPIConfig = Config{
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          5 * time.Minute,
		MaxFailures:      3,
		SuccessThreshold: 2,
	}
)

// Global circuit breaker manager instance
var globalManager = NewManager()

// GetGlobalManager returns the global circuit breaker manager
func GetGlobalManager() *Manager {
	return globalManager
}

// GetOrCreateGlobal gets or creates a circuit breaker from the global manager
func GetOrCreateGlobal(name string, config Config) *CircuitBreaker {
	return globalManager.GetOrCreate(name, config)
}

// GetGlobal gets a circuit breaker from the global manager
func GetGlobal(name string) (*CircuitBreaker, bool) {
	return globalManager.Get(name)
}
