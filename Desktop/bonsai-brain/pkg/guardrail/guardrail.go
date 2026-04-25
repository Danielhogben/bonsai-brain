// Package guardrail provides composable input and output guardrail pipelines
// for the Bonsai Brain agent system. Guardrails inspect or block content
// before it reaches the model (input) or before it is returned to the user (output).
package guardrail

import (
	"context"
	"fmt"
	"strings"
)

// GuardrailResult describes the outcome of a single guardrail check.
type GuardrailResult struct {
	Pass    bool   // true = content is acceptable
	Action  string // e.g. "block", "warn", "redact", ""
	Message string // human-readable explanation
}

// InputGuardrail inspects user input before it is sent to the model.
// Returning Pass=false prevents the input from proceeding.
type InputGuardrail func(ctx context.Context, input string) GuardrailResult

// OutputGuardrail inspects model output before it is returned to the user.
// The second return value (abort) signals that the output pipeline should stop
// and the provided result should be used as the final output.
type OutputGuardrail func(ctx context.Context, output string) (GuardrailResult, bool)

// --- Input pipeline ----------------------------------------------------------

// InputPipeline chains multiple InputGuardrail functions.
// Guardrails run in order; the first one that fails short-circuits the pipeline.
type InputPipeline struct {
	guardrails []InputGuardrail
}

// NewInputPipeline creates an empty input guardrail pipeline.
func NewInputPipeline(guardrails ...InputGuardrail) *InputPipeline {
	return &InputPipeline{guardrails: guardrails}
}

// Add appends one or more guardrails to the pipeline.
func (p *InputPipeline) Add(gs ...InputGuardrail) {
	p.guardrails = append(p.guardrails, gs...)
}

// Run executes every guardrail in sequence. It returns the first failing result,
// or a passing result if all guardrails pass. An empty pipeline always passes.
func (p *InputPipeline) Run(ctx context.Context, input string) GuardrailResult {
	for i, g := range p.guardrails {
		r := g(ctx, input)
		if !r.Pass {
			if r.Action == "" {
				r.Action = "block"
			}
			if r.Message == "" {
				r.Message = fmt.Sprintf("input guardrail #%d failed", i)
			}
			return r
		}
	}
	return GuardrailResult{Pass: true}
}

// --- Output pipeline ---------------------------------------------------------

// OutputPipeline chains multiple OutputGuardrail functions.
// Guardrails run in order; any guardrail can abort the pipeline.
type OutputPipeline struct {
	guardrails []OutputGuardrail
}

// NewOutputPipeline creates an empty output guardrail pipeline.
func NewOutputPipeline(guardrails ...OutputGuardrail) *OutputPipeline {
	return &OutputPipeline{guardrails: guardrails}
}

// Add appends one or more guardrails to the pipeline.
func (p *OutputPipeline) Add(gs ...OutputGuardrail) {
	p.guardrails = append(p.guardrails, gs...)
}

// Run executes every guardrail in sequence. If a guardrail sets abort=true the
// pipeline stops and that result is returned. If all pass, the original output
// is returned unmodified.
func (p *OutputPipeline) Run(ctx context.Context, output string) (string, GuardrailResult) {
	for i, g := range p.guardrails {
		r, abort := g(ctx, output)
		if abort {
			if r.Action == "" {
				r.Action = "block"
			}
			if r.Message == "" {
				r.Message = fmt.Sprintf("output guardrail #%d aborted", i)
			}
			return output, r
		}
		if !r.Pass {
			// Non-passing but non-aborting: treat as abort anyway for safety.
			if r.Action == "" {
				r.Action = "block"
			}
			return output, r
		}
	}
	return output, GuardrailResult{Pass: true}
}

// --- Built-in guardrails -----------------------------------------------------

// MaxInputLength returns an InputGuardrail that rejects inputs longer than maxChars.
func MaxInputLength(maxChars int) InputGuardrail {
	return func(_ context.Context, input string) GuardrailResult {
		if len(input) > maxChars {
			return GuardrailResult{
				Pass:    false,
				Action:  "block",
				Message: fmt.Sprintf("input length %d exceeds maximum %d", len(input), maxChars),
			}
		}
		return GuardrailResult{Pass: true}
	}
}

// BlockedKeywords returns an InputGuardrail that rejects inputs containing any
// of the given keywords (case-insensitive).
func BlockedKeywords(keywords ...string) InputGuardrail {
	lower := make([]string, len(keywords))
	for i, k := range keywords {
		lower[i] = strings.ToLower(k)
	}
	return func(_ context.Context, input string) GuardrailResult {
		li := strings.ToLower(input)
		for _, kw := range lower {
			if strings.Contains(li, kw) {
				return GuardrailResult{
					Pass:    false,
					Action:  "block",
					Message: fmt.Sprintf("input contains blocked keyword %q", kw),
				}
			}
		}
		return GuardrailResult{Pass: true}
	}
}

// MaxOutputLength returns an OutputGuardrail that truncates outputs longer than
// maxChars. It does not abort — it replaces the output in the pipeline.
func MaxOutputLength(maxChars int) OutputGuardrail {
	return func(_ context.Context, output string) (GuardrailResult, bool) {
		if len(output) > maxChars {
			return GuardrailResult{
				Pass:    false,
				Action:  "truncate",
				Message: fmt.Sprintf("output length %d exceeds maximum %d", len(output), maxChars),
			}, true // abort to signal the caller should truncate
		}
		return GuardrailResult{Pass: true}, false
	}
}
