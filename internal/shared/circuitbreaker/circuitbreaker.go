package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed   State = iota // Normal operation
	StateOpen                  // Circuit is open, failing fast
	StateHalfOpen              // Testing if service is back
)

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
	MaxRequests      uint32        // Maximum requests in half-open state
	Interval         time.Duration // Time window for counting failures
	Timeout          time.Duration // Timeout for circuit breaker to stay open
	MaxFailures      uint32        // Maximum failures before opening circuit
	SuccessThreshold uint32        // Successes needed to close circuit from half-open
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		MaxRequests:      3,
		Interval:         10 * time.Second,
		Timeout:          60 * time.Second,
		MaxFailures:      5,
		SuccessThreshold: 2,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config        Config
	state         State
	failures      uint32
	successes     uint32
	lastFailTime  time.Time
	nextAttempt   time.Time
	mutex         sync.RWMutex
	onStateChange func(from, to State)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config:      config,
		state:       StateClosed,
		nextAttempt: time.Now(),
	}
}

// NewCircuitBreakerWithCallback creates a new circuit breaker with state change callback
func NewCircuitBreakerWithCallback(config Config, onStateChange func(from, to State)) *CircuitBreaker {
	return &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		nextAttempt:   time.Now(),
		onStateChange: onStateChange,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Check if we should allow the request
	if !cb.canExecute() {
		return nil, fmt.Errorf("circuit breaker is %s", cb.state.String())
	}

	// Execute the function
	result, err := fn()

	// Record the result
	cb.recordResult(err)

	return result, err
}

// ExecuteAsync executes a function asynchronously with circuit breaker protection
func (cb *CircuitBreaker) ExecuteAsync(ctx context.Context, fn func() (interface{}, error)) <-chan Result {
	resultChan := make(chan Result, 1)

	go func() {
		defer close(resultChan)

		// Check if we should allow the request
		if !cb.canExecute() {
			resultChan <- Result{
				Value: nil,
				Error: fmt.Errorf("circuit breaker is %s", cb.state.String()),
			}
			return
		}

		// Execute the function
		result, err := fn()

		// Record the result
		cb.recordResult(err)

		resultChan <- Result{
			Value: result,
			Error: err,
		}
	}()

	return resultChan
}

// State returns the current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() uint32 {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failures
}

// Successes returns the current success count
func (cb *CircuitBreaker) Successes() uint32 {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.successes
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.nextAttempt = time.Now()

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, StateClosed)
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.After(cb.nextAttempt) {
			// Time to try half-open
			cb.state = StateHalfOpen
			cb.successes = 0
			if cb.onStateChange != nil {
				cb.onStateChange(StateOpen, StateHalfOpen)
			}
			return true
		}
		return false

	case StateHalfOpen:
		return true

	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	// Reset counters if interval has passed
	if now.Sub(cb.lastFailTime) > cb.config.Interval {
		cb.failures = 0
		cb.successes = 0
	}

	if err != nil {
		// Record failure
		cb.failures++
		cb.lastFailTime = now

		switch cb.state {
		case StateClosed:
			if cb.failures >= cb.config.MaxFailures {
				// Open the circuit
				cb.state = StateOpen
				cb.nextAttempt = now.Add(cb.config.Timeout)
				if cb.onStateChange != nil {
					cb.onStateChange(StateClosed, StateOpen)
				}
			}

		case StateHalfOpen:
			// Any failure in half-open state opens the circuit
			cb.state = StateOpen
			cb.nextAttempt = now.Add(cb.config.Timeout)
			if cb.onStateChange != nil {
				cb.onStateChange(StateHalfOpen, StateOpen)
			}
		}
	} else {
		// Record success
		cb.successes++

		switch cb.state {
		case StateClosed:
			// Reset failure count on success
			cb.failures = 0

		case StateHalfOpen:
			if cb.successes >= cb.config.SuccessThreshold {
				// Close the circuit
				cb.state = StateClosed
				cb.failures = 0
				cb.successes = 0
				if cb.onStateChange != nil {
					cb.onStateChange(StateHalfOpen, StateClosed)
				}
			}
		}
	}
}

// Result represents the result of an async execution
type Result struct {
	Value interface{}
	Error error
}

// Metrics holds circuit breaker metrics
type Metrics struct {
	State          State     `json:"state"`
	Failures       uint32    `json:"failures"`
	Successes      uint32    `json:"successes"`
	TotalRequests  uint64    `json:"total_requests"`
	TotalFailures  uint64    `json:"total_failures"`
	TotalSuccesses uint64    `json:"total_successes"`
	LastFailTime   time.Time `json:"last_fail_time"`
	NextAttempt    time.Time `json:"next_attempt"`
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return Metrics{
		State:        cb.state,
		Failures:     cb.failures,
		Successes:    cb.successes,
		LastFailTime: cb.lastFailTime,
		NextAttempt:  cb.nextAttempt,
	}
}
