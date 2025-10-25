package resilience

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
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

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name           string
	state          CircuitState
	failures       int
	requests       int
	lastFailTime   time.Time
	mutex          sync.RWMutex
	
	// Configuration
	maxFailures    int
	timeout        time.Duration
	resetTimeout   time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, maxFailures int, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:          name,
		state:         StateClosed,
		maxFailures:   maxFailures,
		timeout:       timeout,
		resetTimeout:  resetTimeout,
	}
}

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if we should attempt the call
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker '%s' is open", cb.name)
	}

	// Execute the function
	err := fn()
	
	// Record the result
	cb.recordResult(err == nil)
	
	return err
}

// canExecute determines if the circuit breaker should allow execution
func (cb *CircuitBreaker) canExecute() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			log.Info().
				Str("circuit_breaker", cb.name).
				Msg("Circuit breaker moving to half-open state")
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the success or failure of an operation
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.requests++
	
	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles a successful operation
func (cb *CircuitBreaker) onSuccess() {
	if cb.state == StateHalfOpen {
		// Reset the circuit breaker
		cb.reset()
		log.Info().
			Str("circuit_breaker", cb.name).
			Msg("Circuit breaker reset to closed state after successful call")
	}
	// Reset failure count on success
	cb.failures = 0
}

// onFailure handles a failed operation
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
	
	if cb.failures >= cb.maxFailures {
		cb.trip()
	}
}

// trip opens the circuit breaker
func (cb *CircuitBreaker) trip() {
	cb.state = StateOpen
	log.Warn().
		Str("circuit_breaker", cb.name).
		Int("failures", cb.failures).
		Msg("Circuit breaker tripped to open state")
}

// reset closes the circuit breaker
func (cb *CircuitBreaker) reset() {
	cb.state = StateClosed
	cb.failures = 0
	cb.requests = 0
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return map[string]interface{}{
		"name":           cb.name,
		"state":          cb.state.String(),
		"failures":       cb.failures,
		"requests":       cb.requests,
		"last_fail_time": cb.lastFailTime,
	}
}