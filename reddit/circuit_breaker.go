package reddit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// CircuitClosed - normal operation, requests are allowed
	CircuitClosed CircuitState = iota
	// CircuitOpen - circuit is open, requests fail fast
	CircuitOpen
	// CircuitHalfOpen - circuit is testing if service has recovered
	CircuitHalfOpen
)

// String returns a string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures that will trigger the circuit to open
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes in half-open state needed to close the circuit
	SuccessThreshold int

	// Timeout is the duration the circuit stays open before transitioning to half-open
	Timeout time.Duration

	// MaxRequests is the maximum number of requests allowed in half-open state
	// If 0, only one request is allowed at a time in half-open state
	MaxRequests int

	// ShouldTrip is a function that determines if an error should count as a failure
	// If nil, all errors count as failures
	ShouldTrip func(error) bool

	// OnStateChange is called when the circuit state changes
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a sensible default configuration
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          30 * time.Second,
		MaxRequests:      5,
		ShouldTrip: func(err error) bool {
			// Only trip on server errors and timeouts, not client errors
			return IsServerError(err) || IsTemporaryError(err) || errors.Is(err, context.DeadlineExceeded)
		},
		OnStateChange: func(from, to CircuitState) {
			slog.Info("circuit breaker state changed",
				"from", from.String(),
				"to", to.String())
		},
	}
}

// CircuitBreaker implements the circuit breaker pattern for API resilience
type CircuitBreaker struct {
	config *CircuitBreakerConfig

	mu               sync.RWMutex
	state            CircuitState
	failureCount     int
	successCount     int
	lastFailureTime  time.Time
	halfOpenRequests int
}

// CircuitBreakerError represents an error when the circuit breaker is open
type CircuitBreakerError struct {
	State CircuitState
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker is %s", e.State.String())
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	// Set defaults for zero values
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 3
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRequests < 0 {
		config.MaxRequests = 5
	}
	if config.ShouldTrip == nil {
		config.ShouldTrip = func(err error) bool {
			return IsServerError(err) || IsTemporaryError(err) || errors.Is(err, context.DeadlineExceeded)
		}
	}

	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Counts returns the current failure and success counts
func (cb *CircuitBreaker) Counts() (failures, successes int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount, cb.successCount
}

// canRequest determines if a request can be made based on the current state
func (cb *CircuitBreaker) canRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return nil
	case CircuitOpen:
		// Check if enough time has passed to transition to half-open
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenRequests = 0
			return nil
		}
		return &CircuitBreakerError{State: CircuitOpen}
	case CircuitHalfOpen:
		// Check if we can allow more requests in half-open state
		if cb.config.MaxRequests == 0 {
			// Only one request at a time
			if cb.halfOpenRequests > 0 {
				return &CircuitBreakerError{State: CircuitHalfOpen}
			}
		} else if cb.halfOpenRequests >= cb.config.MaxRequests {
			return &CircuitBreakerError{State: CircuitHalfOpen}
		}
		cb.halfOpenRequests++
		return nil
	default:
		return &CircuitBreakerError{State: cb.state}
	}
}

// onSuccess records a successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Reset failure count on success
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.halfOpenRequests-- // Decrement the counter when request completes
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitClosed)
			cb.failureCount = 0
			cb.successCount = 0
			cb.halfOpenRequests = 0 // Reset half-open request counter
		}
	}
}

// onFailure records a failed request
func (cb *CircuitBreaker) onFailure(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Always decrement half-open requests counter if we're in half-open state
	if cb.state == CircuitHalfOpen {
		cb.halfOpenRequests--
	}

	// Check if this error should trip the circuit
	if !cb.config.ShouldTrip(err) {
		return
	}

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		// Any failure in half-open state should open the circuit
		cb.transitionTo(CircuitOpen)
		cb.successCount = 0
		cb.halfOpenRequests = 0 // Reset half-open request counter
	}
}

// transitionTo changes the circuit state and calls the state change callback
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	oldState := cb.state
	cb.state = newState

	slog.Debug("circuit breaker state transition",
		"from", oldState.String(),
		"to", newState.String(),
		"failure_count", cb.failureCount,
		"success_count", cb.successCount)

	if cb.config.OnStateChange != nil {
		// Call the callback without holding the lock to prevent deadlocks
		go cb.config.OnStateChange(oldState, newState)
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if we can make the request
	if err := cb.canRequest(); err != nil {
		return err
	}

	// Execute the function
	err := fn()

	// Record the result
	if err != nil {
		cb.onFailure(err)
		return err
	}

	cb.onSuccess()
	return nil
}

// String returns a string representation of the circuit breaker
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return fmt.Sprintf("CircuitBreaker{state: %s, failures: %d, successes: %d, threshold: %d, timeout: %v}",
		cb.state.String(),
		cb.failureCount,
		cb.successCount,
		cb.config.FailureThreshold,
		cb.config.Timeout)
}
