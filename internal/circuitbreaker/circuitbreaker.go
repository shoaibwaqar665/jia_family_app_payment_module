package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrInvalidState indicates an invalid circuit breaker state
	ErrInvalidState = errors.New("invalid circuit breaker state")
)

// State represents the state of a circuit breaker
type State int

const (
	// StateClosed means the circuit is closed and requests are allowed
	StateClosed State = iota
	// StateOpen means the circuit is open and requests are blocked
	StateOpen
	// StateHalfOpen means the circuit is half-open and testing if the service is back
	StateHalfOpen
)

// String returns a string representation of the state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	// MaxFailures is the maximum number of failures before opening the circuit
	MaxFailures int
	// Timeout is the duration to wait before transitioning from open to half-open
	Timeout time.Duration
	// SuccessThreshold is the number of successful requests needed to close the circuit
	SuccessThreshold int
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxFailures:      5,
		Timeout:          30 * time.Second,
		SuccessThreshold: 3,
	}
}

// CircuitBreaker implements a circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           State
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	config          Config
	logger          *zap.Logger
	name            string
}

// New creates a new circuit breaker
func New(name string, config Config, logger *zap.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: config,
		logger: logger,
		name:   name,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if we should allow the request
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Update circuit breaker state based on result
	cb.afterRequest(err)

	return err
}

// beforeRequest checks if a request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		// Allow all requests when closed
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			if cb.state == StateOpen && time.Since(cb.lastFailureTime) >= cb.config.Timeout {
				cb.state = StateHalfOpen
				cb.successCount = 0
				cb.logger.Info("Circuit breaker transitioning to half-open",
					zap.String("name", cb.name))
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Allow a limited number of requests when half-open
		return nil

	default:
		return ErrInvalidState
	}
}

// afterRequest updates the circuit breaker state based on the result
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		if err != nil {
			cb.failureCount++
			cb.lastFailureTime = time.Now()
			cb.logger.Warn("Circuit breaker failure",
				zap.String("name", cb.name),
				zap.Int("failure_count", cb.failureCount),
				zap.Error(err))

			if cb.failureCount >= cb.config.MaxFailures {
				cb.state = StateOpen
				cb.logger.Error("Circuit breaker opened",
					zap.String("name", cb.name),
					zap.Int("failure_count", cb.failureCount))
			}
		} else {
			// Reset failure count on success
			cb.failureCount = 0
		}

	case StateOpen:
		// Do nothing in open state

	case StateHalfOpen:
		if err != nil {
			// Return to open state on failure
			cb.state = StateOpen
			cb.failureCount = cb.config.MaxFailures
			cb.lastFailureTime = time.Now()
			cb.logger.Error("Circuit breaker re-opened after half-open failure",
				zap.String("name", cb.name),
				zap.Error(err))
		} else {
			cb.successCount++
			cb.logger.Info("Circuit breaker half-open success",
				zap.String("name", cb.name),
				zap.Int("success_count", cb.successCount))

			if cb.successCount >= cb.config.SuccessThreshold {
				cb.state = StateClosed
				cb.failureCount = 0
				cb.successCount = 0
				cb.logger.Info("Circuit breaker closed after successful recovery",
					zap.String("name", cb.name))
			}
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}

	cb.logger.Info("Circuit breaker reset",
		zap.String("name", cb.name))
}

// Stats returns statistics about the circuit breaker
type Stats struct {
	State         State
	FailureCount  int
	SuccessCount  int
	LastFailureAt time.Time
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:         cb.state,
		FailureCount:  cb.failureCount,
		SuccessCount:  cb.successCount,
		LastFailureAt: cb.lastFailureTime,
	}
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   *zap.Logger
}

// NewManager creates a new circuit breaker manager
func NewManager(logger *zap.Logger) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if exists {
		return breaker
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	breaker = New(name, config, m.logger)
	m.breakers[name] = breaker

	return breaker
}

// Get returns a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	breaker, exists := m.breakers[name]
	if !exists {
		return nil, fmt.Errorf("circuit breaker %s not found", name)
	}

	return breaker, nil
}

// GetAllStats returns statistics for all circuit breakers
func (m *Manager) GetAllStats() map[string]Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]Stats)
	for name, breaker := range m.breakers {
		stats[name] = breaker.GetStats()
	}

	return stats
}
