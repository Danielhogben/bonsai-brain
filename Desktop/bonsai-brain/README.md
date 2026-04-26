# Bonsai Brain v3

<p align="center">
  <strong>A compiled Go agent reasoning engine. Designed to run smart on tiny hardware.</strong>
</p>

<p align="center">
  <a href="https://github.com/Danielhogben/bonsai-brain/releases"><img src="https://img.shields.io/github/v/release/Danielhogben/bonsai-brain?style=flat-square" alt="Release"></a>
  <a href="https://pkg.go.dev/github.com/donn/bonsai-brain"><img src="https://img.shields.io/badge/reference-pkg.go.dev-blue?style=flat-square" alt="Go Reference"></a>
  <a href="https://github.com/Danielhogben/bonsai-brain/actions"><img src="https://img.shields.io/github/actions/workflow/status/Danielhogben/bonsai-brain/ci.yml?style=flat-square" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
</p>

---

**Bonsai Brain v3** is a modular, production-ready AI agent framework built in Go. It distills proven patterns from Claude Code, agent-zero, elizaOS, VoltAgent, and DeerFlow into a single static binary that starts in milliseconds and runs comfortably on hardware as small as a Raspberry Pi Zero.

Unlike Python-based agent frameworks, Bonsai Brain compiles to native machine code. No interpreter. No dependency hell. No 2 GB PyTorch installs. Just a single executable you can `scp` to an edge device and run.

---

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Package Reference](#package-reference)
- [Design Philosophy](#design-philosophy)
- [Performance](#performance)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## Features

| Feature | Description |
|---------|-------------|
| **Streaming Query Engine** | Core reasoning loop with tool-call iteration, 3-state permission pipeline, and configurable max-iteration guards |
| **Typed Tool System** | JSON-Schema validated parameters, lifecycle hooks (`OnStart` / `OnEnd`), and granular approval gates |
| **Composable Guardrails** | Input/output safety pipelines that can block, redact, warn, or truncate based on your policies |
| **Middleware Pipeline** | Transform content before/after model calls with built-in retry support for transient failures |
| **4-Component Plugin System** | Actions, Providers, Evaluators, and Services — the same pattern that powers elizaOS, adapted for compiled code |
| **Hierarchical Agents** | Spawn sub-agents with depth limits, inherited features, and full pipeline isolation |
| **Tolerant JSON Parser** | `dirtyjson` repairs trailing commas, unquoted keys, single-quoted strings, and missing braces from LLM output |
| **Context Registry** | Thread-safe agent context store with goroutine-local propagation via `context.Context` |
| **Zero External Dependencies** | Pure standard library + Go runtime. No cgo. No CUDA. No conda. |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              User Input                                  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  INPUT PIPELINE                                                         │
│  ├─ middleware.InputPipeline   (transform, rewrite, summarize)          │
│  └─ guardrail.InputPipeline    (block keywords, max length, etc.)       │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  AGENT (hierarchical, depth-limited)                                     │
│  ├─ Context registry (per-agent state + history)                        │
│  ├─ Plugin manager (actions, providers, evaluators, services)           │
│  └─ QueryEngine                                                         │
│       ├─ SystemPromptBuilder (labelled, filterable sources)             │
│       ├─ PermissionChecker (allow / block / ask-user)                   │
│       └─ ModelClient interface (bring your own LLM backend)             │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  OUTPUT PIPELINE                                                        │
│  ├─ guardrail.OutputPipeline   (truncate, content policy)               │
│  └─ middleware.OutputPipeline  (format, inject metadata)                │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              User Output                                 │
└─────────────────────────────────────────────────────────────────────────┘
```

### Permission Pipeline

The engine implements a **3-state permission gate** for every tool call:

- **`PermissionAllow`** — execute immediately
- **`PermissionBlock`** — hard deny, surface error to model
- **`PermissionAskUser`** — pause and invoke an approval callback; only proceed if the user confirms

This is the same safety model used by Claude Code's `buildTool`, adapted for autonomous and semi-autonomous deployments.

---

## Installation

```bash
go get github.com/donn/bonsai-brain@latest
```

Or clone and build locally:

```bash
git clone https://github.com/Danielhogben/bonsai-brain.git
cd bonsai-brain
go build ./...
go test ./...
```

### Cross-compilation examples

```bash
# Linux AMD64 (desktop / server)
GOOS=linux GOARCH=amd64 go build -o bonsai-linux-amd64 ./...

# Linux ARM64 (Raspberry Pi 4, Apple Silicon servers)
GOOS=linux GOARCH=arm64 go build -o bonsai-linux-arm64 ./...

# Linux ARM / RISC-V (Pi Zero, embedded)
GOOS=linux GOARCH=arm GOARM=7 go build -o bonsai-linux-armv7 ./...
GOOS=linux GOARCH=riscv64 go build -o bonsai-linux-riscv64 ./...

# Windows
GOOS=windows GOARCH=amd64 go build -o bonsai-windows-amd64.exe ./...
```

---

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/donn/bonsai-brain/pkg/agent"
	"github.com/donn/bonsai-brain/pkg/engine"
	"github.com/donn/bonsai-brain/pkg/guardrail"
)

// MyModel implements engine.ModelClient — wire this to OpenAI, Groq,
// Ollama, llama.cpp, or any OpenAI-compatible endpoint.
type MyModel struct{}

func (m *MyModel) Stream(ctx context.Context, messages []engine.Message, tools []engine.ToolSchema) (*engine.Response, error) {
	// Your LLM backend integration here.
	return &engine.Response{Content: "Hello from Bonsai Brain!", FinishReason: "stop"}, nil
}

func main() {
	ctx := context.Background()

	// 1. Create the query engine
	eng := engine.NewQueryEngine(&MyModel{})

	// 2. Register a tool
	eng.RegisterTool(
		engine.ToolSchema{
			Name:        "echo",
			Description: "Echo back the input text",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string"},
				},
				"required": []string{"text"},
			},
		},
		func(_ context.Context, args map[string]any) (string, error) {
			return args["text"].(string), nil
		},
	)

	// 3. Wrap the engine in an Agent with guardrails
	cfg := agent.DefaultConfig("bonsai-demo")
	cfg.SystemPrompt = "You are a helpful assistant running inside Bonsai Brain v3."
	ag := agent.New(cfg, eng)

	// 4. Add an input guardrail (max 2,000 characters)
	ag.InGuardrails.Add(guardrail.MaxInputLength(2000))

	// 5. Run
	reply, err := ag.GenerateText(ctx, "Say hello and then echo 'bonsai' using the echo tool.")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply)
}
```

---

## Package Reference

| Package | Path | Description | Pattern Source |
|---------|------|-------------|----------------|
| `engine` | `pkg/engine` | Query engine core loop, streaming interface, 3-state permission pipeline, system-prompt builder | Claude Code |
| `tool` | `pkg/tool` | Typed tool definitions with JSON-Schema validation, approval gates, `OnStart`/`OnEnd` hooks, thread-safe registry | VoltAgent |
| `guardrail` | `pkg/guardrail` | Composable input/output safety pipelines. Block, warn, redact, or truncate content | VoltAgent |
| `middleware` | `pkg/middleware` | Transform pipelines before/after model calls. Retry-aware execution for transient aborts | VoltAgent + DeerFlow |
| `plugin` | `pkg/plugin` | 4-component plugin system: Actions, Providers, Evaluators, Services | elizaOS |
| `context` | `pkg/context` | Thread-safe agent context registry with goroutine-local propagation | agent-zero |
| `dirtyjson` | `pkg/dirtyjson` | Tolerant JSON parser: fixes trailing commas, unquoted keys, single quotes, missing braces | agent-zero |
| `agent` | `pkg/agent` | Hierarchical agents with depth limits, full middleware/guardrail pipeline, retry support | agent-zero + DeerFlow |

---

## Design Philosophy

Bonsai Brain v3 is not another wrapper around a Python LLM library. It is a **distillation** — taking the best architectural patterns from the current generation of agent frameworks and re-implementing them in a language designed for reliability, concurrency, and tiny deployments.

### What we borrowed

- **Claude Code** — The 3-state permission pipeline (`allow` / `block` / `ask-user`) and the streaming query loop that iterates on tool calls until the model is satisfied.
- **agent-zero** — `AgentContext` as a typed, hierarchical state bag; `DirtyJson` for surviving real-world LLM output; sub-agent spawning with depth limits.
- **elizaOS** — The 4-component plugin architecture (Action / Provider / Evaluator / Service) that lets you extend behavior without forking core code.
- **VoltAgent** — Strongly typed tools with JSON-Schema validation, guardrail pipelines that separate safety from transformation, and middleware hooks for retry logic.
- **DeerFlow** — Context engineering through labelled prompt sources, super-agent harness patterns, and output middleware that can trigger regeneration.

### What we changed

| Python Framework | Bonsai Brain v3 Equivalent |
|------------------|---------------------------|
| Dynamic typing | Static types + compile-time checks |
| `asyncio` event loop | Native goroutines + `context.Context` |
| pip / conda / poetry | `go mod` — single lockfile, reproducible builds |
| ~500 MB–2 GB runtime | ~10–20 MB static binary |
| Startup: 2–10 seconds | Startup: <50 milliseconds |
| Cuda / torch required | Pure Go — runs on ARM, RISC-V, WASM |

---

## Performance

Bonsai Brain is designed for **edge and embedded deployments** where every megabyte and every millisecond counts.

| Metric | Typical Value |
|--------|---------------|
| Binary size | ~8–15 MB (stripped, static) |
| Resident memory | ~5–15 MB base + model client overhead |
| Cold startup | <50 ms |
| Goroutine stack | 2 KB default — spawn thousands of sub-agents without worry |
| Cross-compile time | <5 seconds for `linux/arm` on a modern desktop |

> **Target hardware:** Raspberry Pi Zero 2 W (512 MB RAM), RISC-V SBCs, old x86 thin clients, and WASM edge workers.

---

## Roadmap

- [ ] `examples/` — Reference integrations for OpenAI, Groq, Ollama, and llama.cpp
- [ ] `pkg/memory` — Short-term conversation memory with automatic summarization
- [ ] `pkg/vector` — Lightweight in-process vector store for RAG (no external DB)
- [ ] `pkg/codec` — Structured output decoding with JSON Schema enforcement
- [ ] `pkg/telemetry` — OpenTelemetry traces and metrics export
- [ ] WASM target — Compile agents to WebAssembly for edge functions
- [ ] CLI (`bonsai`) — Single-binary agent runner with YAML config and hot reload

---

## Contributing

Contributions are welcome. Please open an issue first to discuss large changes.

```bash
# Fork, clone, and build
git clone https://github.com/yourname/bonsai-brain.git
cd bonsai-brain
go test ./...

# Run linting
go vet ./...
gofmt -w .
```

### Code style

- Go standard formatting (`gofmt`)
- Package-level doc comments on every public package
- Table-driven tests for logic-heavy code
- No `cgo` — keep it pure Go

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  Built with 🌳 by the OmniClaw Collective
</p>
