package middleware

import (
	"context"
	"errors"
	"testing"
)

func TestRetryWithBackoff_Success(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, attempt int) error {
		attempts++
		return nil
	}

	cfg := DefaultRetryConfig()
	err := RetryWithBackoff(context.Background(), cfg, fn)
	if err != nil {
		t.Errorf("RetryWithBackoff returned error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithBackoff_StopsOnNonRetryableError(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, attempt int) error {
		attempts++
		return errors.New("permanent failure")
	}

	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 3
	err := RetryWithBackoff(context.Background(), cfg, fn)
	if err == nil {
		t.Error("RetryWithBackoff should have returned error")
	}
	// shouldRetry returns false for "permanent failure" - no retry
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on non-retryable error)", attempts)
	}
}

func TestRetryWithBackoff_EventualSuccess(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, attempt int) error {
		attempts++
		if attempts < 3 {
			return errors.New("network timeout")
		}
		return nil
	}

	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 5
	err := RetryWithBackoff(context.Background(), cfg, fn)
	if err != nil {
		t.Errorf("RetryWithBackoff returned error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithBackoff_Exhausted(t *testing.T) {
	attempts := 0
	fn := func(ctx context.Context, attempt int) error {
		attempts++
		return errors.New("connection refused")
	}

	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 2
	err := RetryWithBackoff(context.Background(), cfg, fn)
	if err == nil {
		t.Error("RetryWithBackoff should have returned error")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
