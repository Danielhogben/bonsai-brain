// Package middleware provides composable input and output middleware pipelines
// with retry support for the Bonsai Brain agent system.
//
// Middleware differs from guardrails: middleware can *transform* content
// (rewrite, summarize, translate), while guardrails only *accept or reject* it.
package middleware

import (
	"context"
	"fmt"
	"time"
)

// InputMiddleware transforms user input before it reaches the model.
// Returning an error halts the pipeline.
type InputMiddleware func(ctx context.Context, input string) (string, error)

// OutputMiddleware transforms model output before it is returned to the user.
// The abort callback allows a middleware to stop the pipeline early; calling
// abort(reason, true) signals that the caller should retry generation.
type OutputMiddleware func(ctx context.Context, output string, abort func(reason string, retry bool)) (string, error)

// --- Input pipeline ----------------------------------------------------------

// InputPipeline chains multiple InputMiddleware functions in order.
type InputPipeline struct {
	middlewares []InputMiddleware
}

// NewInputPipeline creates an input pipeline with the given middlewares.
func NewInputPipeline(mws ...InputMiddleware) *InputPipeline {
	return &InputPipeline{middlewares: mws}
}

// Add appends one or more middlewares to the pipeline.
func (p *InputPipeline) Add(mws ...InputMiddleware) {
	p.middlewares = append(p.middlewares, mws...)
}

// Run passes the input through each middleware in sequence, threading the
// transformed string through. Returns the final transformed input or the first
// error encountered.
func (p *InputPipeline) Run(ctx context.Context, input string) (string, error) {
	var err error
	for _, mw := range p.middlewares {
		input, err = mw(ctx, input)
		if err != nil {
			return "", fmt.Errorf("input middleware: %w", err)
		}
	}
	return input, nil
}

// --- Output pipeline ---------------------------------------------------------

// OutputPipeline chains multiple OutputMiddleware functions in order.
type OutputPipeline struct {
	middlewares []OutputMiddleware
}

// NewOutputPipeline creates an output pipeline with the given middlewares.
func NewOutputPipeline(mws ...OutputMiddleware) *OutputPipeline {
	return &OutputPipeline{middlewares: mws}
}

// Add appends one or more middlewares to the pipeline.
func (p *OutputPipeline) Add(mws ...OutputMiddleware) {
	p.middlewares = append(p.middlewares, mws...)
}

// AbortError is returned when an output middleware calls abort.
type AbortError struct {
	Reason string
	Retry  bool
}

func (e *AbortError) Error() string {
	if e.Retry {
		return fmt.Sprintf("output middleware aborted with retry: %s", e.Reason)
	}
	return fmt.Sprintf("output middleware aborted: %s", e.Reason)
}

// Run passes the output through each middleware in sequence. If any middleware
// calls abort, the pipeline stops and an *AbortError is returned alongside the
// output as it was before abort.
func (p *OutputPipeline) Run(ctx context.Context, output string) (string, error) {
	for _, mw := range p.middlewares {
		aborted := false
		var abortErr *AbortError

		abortFn := func(reason string, retry bool) {
			aborted = true
			abortErr = &AbortError{Reason: reason, Retry: retry}
		}

		var err error
		output, err = mw(ctx, output, abortFn)
		if aborted {
			return output, abortErr
		}
		if err != nil {
			return "", fmt.Errorf("output middleware: %w", err)
		}
	}
	return output, nil
}

// --- Retry-aware execution ---------------------------------------------------

// RunWithRetry executes fn and, if it returns an *AbortError with Retry=true,
// retries up to maxRetries times with the given delay between attempts.
// It returns the final output or the last error encountered.
func RunWithRetry(ctx context.Context, maxRetries int, delay time.Duration, fn func(ctx context.Context) (string, error)) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		output, err := fn(ctx)
		if err == nil {
			return output, nil
		}

		// Check if the error is a retryable abort.
		var abortErr *AbortError
		ok := false
		if abortErr, ok = err.(*AbortError); ok && abortErr.Retry && attempt < maxRetries {
			lastErr = err
			// Wait before retrying.
			t := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				t.Stop()
				return "", ctx.Err()
			case <-t.C:
			}
			continue
		}

		// Non-retryable error or exhausted retries.
		return output, err
	}
	return "", fmt.Errorf("retries exhausted: %w", lastErr)
}

// --- Built-in middlewares ----------------------------------------------------

// TrimWhitespace returns an InputMiddleware that trims leading and trailing whitespace.
func TrimWhitespace() InputMiddleware {
	return func(_ context.Context, input string) (string, error) {
		return trimSpace(input), nil
	}
}

// PrefixSystemPrompt returns an InputMiddleware that prepends a system prompt.
func PrefixSystemPrompt(prefix string) InputMiddleware {
	return func(_ context.Context, input string) (string, error) {
		if prefix == "" {
			return input, nil
		}
		return prefix + "\n\n" + input, nil
	}
}

// TruncateOutput returns an OutputMiddleware that truncates output to maxChars.
// If the output exceeds the limit it aborts without retry so the caller can
// handle the truncation.
func TruncateOutput(maxChars int) OutputMiddleware {
	return func(_ context.Context, output string, abort func(reason string, retry bool)) (string, error) {
		if len(output) > maxChars {
			abort(fmt.Sprintf("output length %d exceeds %d", len(output), maxChars), false)
			return output[:maxChars], nil
		}
		return output, nil
	}
}

// --- Helpers (avoid importing strings to keep the surface small) ------------

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && isSpace(s[i]) {
		i++
	}
	for j > i && isSpace(s[j-1]) {
		j--
	}
	return s[i:j]
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
