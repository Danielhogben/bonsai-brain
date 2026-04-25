# Bonsai Brain v3

A compiled Go agent reasoning engine. Designed to run smart on tiny hardware.

## Architecture

Bonsai Brain v3 is a modular AI agent framework built in Go, extracting proven patterns from Claude Code, agent-zero, elizaOS, VoltAgent, and DeerFlow.

### Packages

| Package | Description | Pattern Source |
|---------|-------------|---------------|
| `pkg/engine` | Query engine core loop with streaming | Claude Code |
| `pkg/tool` | Typed tools with validation + hooks | VoltAgent |
| `pkg/guardrail` | Input/output safety pipeline | VoltAgent |
| `pkg/middleware` | Transform pipeline with retry | VoltAgent + DeerFlow |
| `pkg/plugin` | 4-component plugin system | elizaOS |
| `pkg/context` | Thread-safe agent context registry | agent-zero |
| `pkg/dirtyjson` | Tolerant JSON parser for LLM output | agent-zero |
| `pkg/agent` | Hierarchical agents with pipeline | agent-zero + DeerFlow |

### Core Design

- **Query Engine**: Streams from model, loops on tool calls, 3-state permission pipeline
- **Plugins**: Actions (do things), Providers (inject context), Evaluators (post-analysis), Services (long-running)
- **Guardrails**: Input/output safety checks that can block, modify, or allow
- **Middleware**: Transform pipeline before/after model calls with retry on abort
- **DirtyJson**: State-machine parser tolerant of malformed LLM JSON output
- **Agents**: Hierarchical spawning with depth limits, full middleware pipeline

## Building

```bash
go build ./...
go test ./...
```

## Why Go?

- Compiled binary, single static executable
- Goroutine concurrency for parallel agents
- Tiny memory footprint
- Fast startup, no runtime overhead
- Perfect for 4GB VRAM targets

## License

MIT
