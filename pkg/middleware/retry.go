package middleware

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds the exponential backoff configuration.
type RetryConfig struct {
	Enabled            bool          // Enable retry mechanism
	MaxAttempts        int           // Maximum retry attempts (default: 5)
	InitialDelay       time.Duration // Initial delay before first retry (default: 100ms)
	MaxDelay           time.Duration // Maximum delay between retries (default: 10s)
	BackoffFactor      float64       // Exponential backoff multiplier (default: 2.0)
	Jitter             bool          // Add random jitter to delays (default: true)
	RetryOnTimeout     bool          // Retry on context deadline exceeded
	RetryOnNetworkErr  bool          // Retry on network errors
	RetryOn5xx         bool          // Retry on 5xx HTTP status codes
	RetryOn429         bool          // Retry on 429 rate limit
}

// DefaultRetryConfig returns the standard exponential backoff configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Enabled:           true,
		MaxAttempts:       5,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          10 * time.Second,
		BackoffFactor:     2.0,
		Jitter:            true,
		RetryOnTimeout:    true,
		RetryOnNetworkErr: true,
		RetryOn5xx:        true,
		RetryOn429:        true,
	}
}

// RetryFn is the function type that retry middleware executes.
type RetryFn func(ctx context.Context, attempt int) error

// RetryWithBackoff executes fn with exponential backoff and jitter.
// Returns the last error if all attempts fail, nil on success.
func RetryWithBackoff(ctx context.Context, cfg *RetryConfig, fn RetryFn) error {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}
	if !cfg.Enabled {
		return fn(ctx, 1)
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		lastErr = fn(ctx, attempt)
		if lastErr == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		// Determine if we should retry this error
		if !shouldRetry(cfg, lastErr) {
			break
		}

		// Compute exponential backoff delay with optional jitter
		delay := computeDelay(cfg, attempt)

		// Wait or cancel
		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("retry exhausted after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// shouldRetry determines if an error is eligible for retry based on config.
func shouldRetry(cfg *RetryConfig, err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check configured retry conditions
	if cfg.RetryOnTimeout {
		if err == context.DeadlineExceeded || err == context.Canceled {
			return true
		}
	}

	if cfg.RetryOnNetworkErr {
		// Common network error patterns
		networkErrs := []string{
			"network", "timeout", "deadline", "connection", "refused",
			"reset", "closed", "unreachable", "temporary", "EOF",
		}
		for _, pattern := range networkErrs {
			if contains(errStr, pattern) {
				return true
			}
		}
	}

	// HTTP status code checks would be inserted by the caller typically
	// This is a generic helper for transport/IO errors
	return false
}

// computeDelay calculates the exponential backoff delay with optional jitter.
func computeDelay(cfg *RetryConfig, attempt int) time.Duration {
	// Exponential: initialDelay * backoffFactor^(attempt-1)
	exp := math.Pow(cfg.BackoffFactor, float64(attempt-1))
	delay := float64(cfg.InitialDelay) * exp

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter (±25%) if enabled to avoid thundering herd
	if cfg.Jitter {
		jitterRange := delay * 0.25
		jitter := (rand.Float64()*2 - 1) * jitterRange // [-25%, +25%]
		delay += jitter
	}

	return time.Duration(delay)
}

// contains is a simple substring check (to avoid extra imports).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || index(s, substr) >= 0)
}

func index(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}