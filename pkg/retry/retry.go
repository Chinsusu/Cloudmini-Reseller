// Package retry provides exponential backoff retry logic for transient failures.
// Suitable for NATS publish, HTTP calls, database operations, etc.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrMaxRetries is returned when all retry attempts are exhausted.
var ErrMaxRetries = errors.New("retry: max attempts exhausted")

// Config defines retry behaviour.
type Config struct {
	// MaxAttempts is the total number of attempts (including the first).
	MaxAttempts int
	// InitialDelay is the delay before the second attempt.
	InitialDelay time.Duration
	// MaxDelay caps the exponential backoff delay.
	MaxDelay time.Duration
	// Multiplier is the backoff factor (e.g. 2.0 doubles each time).
	Multiplier float64
	// Jitter adds ±20% random jitter to avoid thundering herd.
	Jitter bool
	// IsRetryable returns true if the error should be retried.
	// If nil, all non-nil errors are retried.
	IsRetryable func(err error) bool
}

// Default returns sensible defaults for HTTP/NATS calls.
func Default() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// Aggressive returns settings for database operations where fast retry is important.
func Aggressive() Config {
	return Config{
		MaxAttempts:  5,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		Multiplier:   1.5,
		Jitter:       true,
	}
}

// Do executes fn with retry according to cfg.
// ctx cancellation aborts the retry loop immediately.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context, attempt int) error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return fmt.Errorf("retry: context cancelled: %w", ctx.Err())
		}

		lastErr = fn(ctx, attempt)
		if lastErr == nil {
			return nil // success
		}

		// Check if error is retryable
		if cfg.IsRetryable != nil && !cfg.IsRetryable(lastErr) {
			return lastErr // non-retryable — return immediately
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		// Compute next delay with exponential backoff
		sleepDur := computeDelay(delay, cfg.MaxDelay, cfg.Jitter)
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry: context cancelled during wait: %w", ctx.Err())
		case <-time.After(sleepDur):
		}
	}

	return fmt.Errorf("%w after %d attempts: %w", ErrMaxRetries, cfg.MaxAttempts, lastErr)
}

// DoSimple is a convenience wrapper that calls fn() without attempt number.
func DoSimple(ctx context.Context, cfg Config, fn func() error) error {
	return Do(ctx, cfg, func(ctx context.Context, _ int) error { return fn() })
}

func computeDelay(base, max time.Duration, jitter bool) time.Duration {
	d := base
	if jitter {
		// ±20% jitter using math (no rand import needed — deterministic enough)
		factor := 1.0 + 0.2*math.Sin(float64(time.Now().UnixNano()))
		d = time.Duration(float64(d) * factor)
	}
	if d > max {
		return max
	}
	if d < time.Millisecond {
		return time.Millisecond
	}
	return d
}

// IsTemporaryError returns true for errors that are likely transient.
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}
	type temporary interface{ Temporary() bool }
	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}
	return false
}
