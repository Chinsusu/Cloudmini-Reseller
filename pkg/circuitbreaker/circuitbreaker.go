// Package circuitbreaker provides a generic Circuit Breaker implementation
// using the three-state machine (Closed → Open → HalfOpen) pattern.
// Thread-safe; designed to wrap any fallible external call.
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit is open and requests are rejected.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // Normal operation — requests pass through
	StateOpen                  // Failure threshold breached — requests rejected immediately
	StateHalfOpen              // Probe state — one test request allowed
)

func (s State) String() string {
	return [...]string{"closed", "open", "half_open"}[s]
}

// Config holds circuit breaker configuration.
type Config struct {
	// MaxFailures is the consecutive failure count that trips the breaker.
	MaxFailures int
	// Timeout is how long the breaker stays open before moving to HalfOpen.
	Timeout time.Duration
	// MaxHalfOpenRequests is concurrent requests allowed in HalfOpen state.
	MaxHalfOpenRequests int
	// OnStateChange is called when the state transitions.
	OnStateChange func(name string, from, to State)
}

// DefaultConfig returns sensible defaults for a downstream HTTP service.
func DefaultConfig(name string) Config {
	return Config{
		MaxFailures:         5,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 1,
		OnStateChange: func(n string, from, to State) {
			// Replace with structured logger in production
			_ = n
		},
	}
}

// CircuitBreaker implements the three-state breaker pattern.
type CircuitBreaker struct {
	name    string
	cfg     Config
	mu      sync.Mutex
	state   State
	failures    int
	successes   int
	lastFailure time.Time
	halfOpen    int // concurrent half-open requests
}

// New creates a new CircuitBreaker.
func New(name string, cfg Config) *CircuitBreaker {
	return &CircuitBreaker{name: name, cfg: cfg, state: StateClosed}
}

// Execute runs fn if the circuit allows it.
// Returns ErrCircuitOpen if requests are being short-circuited.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}
	err := fn()
	cb.afterRequest(err)
	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if timeout has elapsed → move to HalfOpen
		if time.Since(cb.lastFailure) >= cb.cfg.Timeout {
			cb.toHalfOpen()
			cb.halfOpen++
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.halfOpen < cb.cfg.MaxHalfOpenRequests {
			cb.halfOpen++
			return nil
		}
		return ErrCircuitOpen
	}
	return nil
}

func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.halfOpen--
	}

	if err != nil {
		cb.failure()
	} else {
		cb.success()
	}
}

func (cb *CircuitBreaker) failure() {
	cb.failures++
	cb.lastFailure = time.Now()
	cb.successes = 0

	if cb.state == StateHalfOpen || cb.failures >= cb.cfg.MaxFailures {
		cb.toOpen()
	}
}

func (cb *CircuitBreaker) success() {
	cb.successes++
	cb.failures = 0

	if cb.state == StateHalfOpen {
		// After 1 successful probe → close the circuit
		cb.toClosed()
	}
}

func (cb *CircuitBreaker) toClosed() {
	prev := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpen = 0
	if cb.cfg.OnStateChange != nil {
		go cb.cfg.OnStateChange(cb.name, prev, StateClosed)
	}
}

func (cb *CircuitBreaker) toOpen() {
	prev := cb.state
	cb.state = StateOpen
	if cb.cfg.OnStateChange != nil {
		go cb.cfg.OnStateChange(cb.name, prev, StateOpen)
	}
}

func (cb *CircuitBreaker) toHalfOpen() {
	prev := cb.state
	cb.state = StateHalfOpen
	cb.failures = 0
	if cb.cfg.OnStateChange != nil {
		go cb.cfg.OnStateChange(cb.name, prev, StateHalfOpen)
	}
}

// State returns the current circuit state (thread-safe).
func (cb *CircuitBreaker) CurrentState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Failures returns the current consecutive failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}
