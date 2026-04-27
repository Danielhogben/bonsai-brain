# 🗺️ Bonsai Brain Roadmap

> **Current version:** v0.3.0 | **Next milestone:** v0.4.0 — The Swarm

---

## Milestones

### ✅ v0.3.0 — Core Engine (COMPLETE)
**Released:** 2026-04-26

- [x] Streaming reasoning loop with tool calls
- [x] Typed tool registry with approval gates
- [x] Composable guardrails (input/output safety)
- [x] Composable middleware (transform pipelines)
- [x] Hierarchical agents with depth limits
- [x] Plugin system (Actions, Providers, Evaluators, Services)
- [x] Conversation memory with auto-summarization
- [x] Pure-Go vector store with cosine similarity
- [x] Zero-dependency embedders (HashEmbedder, TFIDFEmbedder)
- [x] OpenAI-compatible client for proxy/Ollama/llama.cpp
- [x] CLI with version, run, chat commands
- [x] Cross-compilation: linux/amd64, arm64, armv6, armv7, riscv64, windows/amd64, js/wasm
- [x] Live demos: proxy integration, Ollama agent, PrismML Bonsai 1.7B
- [x] Professional README with badges and quickstart

---

### 🚧 v0.4.0 — The Swarm (IN PROGRESS)
**Target:** 2026-05-15

- [x] Multi-provider model registry (OpenRouter, Groq, NVIDIA, Gemini, Cohere, Ollama)
- [x] Distributed agent orchestrator with concurrency control
- [x] Per-provider rate limiting
- [x] Custom API adapters (Cohere native chat API)
- [x] `bonsai swarm` CLI command
- [x] Health checks across all agents
- [x] Task routing by capability (coding → code model, creative → creative model)
- [x] Agent-to-agent (A2A) protocol support
- [x] MCP (Model Context Protocol) integration
- [x] Result judge/scoring with a "judge" model
- [x] Automatic fallback chains (if model A fails, try B, then C)
- [x] Swarm config file (YAML/JSON) for persistent agent fleets
- [x] WebSocket-based real-time swarm dashboard

**Good first issues:**
- [#1] Add more provider configs (Together AI, Fireworks, Cerebras)
- [#2] Build a ` JudgeModel` result strategy that uses an LLM to score outputs
- [#3] Add retry with exponential backoff for failed model calls
- [#4] Create a `swarm.yaml` config loader

---

### 📦 v0.5.0 — Tool Ecosystem
**Target:** 2026-06-01

- [ ] Filesystem tools (read, write, glob, grep)
- [ ] Git tools (status, diff, commit, push)
- [ ] Web tools (fetch, search with Tavily/Exa)
- [ ] Docker tools (run, exec, logs)
- [ ] Database tools (SQL query, schema introspection)
- [ ] Messaging tools (Telegram bot, Discord webhook)
- [ ] Code execution sandbox (restricted Go playground)
- [ ] Tool marketplace / registry website
- [ ] 50+ pre-built tools in `pkg/tools/`

**Good first issues:**
- [#5] Wrap `curl` as a Bonsai Brain tool
- [#6] Build a `grep_files` tool using `pkg/dirtyjson` patterns
- [#7] Create a `send_telegram` tool using the Telegram Bot API
- [#8] Build a `docker_ps` tool

---

### 🎨 v0.6.0 — Terminal UI
**Target:** 2026-06-15

- [ ] TUI built with Charm Bubble Tea
- [ ] Real-time agent state visualization
- [ ] Streaming output with syntax highlighting
- [ ] Interactive approval prompts
- [ ] Multi-agent dashboard (swarm view)
- [ ] File tree browser
- [ ] History search and replay
- [ ] Configuration wizard

**Good first issues:**
- [#9] Prototype a simple list view of running agents with Bubble Tea
- [#10] Add a progress spinner component for tool execution
- [#11] Build a chat history viewer with search

---

### 🧪 v0.7.0 — Benchmarks & Quality
**Target:** 2026-07-01

- [ ] Standardized benchmark suite against other frameworks
- [ ] Edge device performance tests (Pi Zero, Pi 4)
- [ ] Token efficiency comparisons
- [ ] Latency measurements across providers
- [ ] Public leaderboard (website or GitHub README)
- [ ] Regression testing for model outputs
- [ ] Promptfoo integration for prompt evaluation

---

### 🔐 v0.8.0 — Security & Sandboxing
**Target:** 2026-07-15

- [ ] File path allowlist sandboxing
- [ ] Network egress controls
- [ ] Docker container isolation for code execution
- [ ] Secret scanning in outputs
- [ ] Audit logging for all tool calls
- [ ] RBAC for multi-user deployments

---

### 🌍 v0.9.0 — The Collective
**Target:** 2026-08-01

- [ ] Decentralized agent discovery protocol
- [ ] Agent migration between nodes
- [ ] Consensus mechanisms for distributed decisions
- [ ] Load balancing across a fleet
- [ ] End-to-end encryption for agent communication
- [ ] Federation: multiple Bonsai Brain instances form a mesh

---

### 🚀 v1.0.0 — Production Ready
**Target:** 2026-09-01

- [ ] Stable API (no breaking changes within major version)
- [ ] Complete documentation site
- [ ] 100+ community-contributed tools
- [ ] 20+ example agents
- [ ] Package manager integration (apt, homebrew, chocolatey)
- [ ] Docker Compose deployment template
- [ ] Kubernetes operator
- [ ] Commercial support tier
- [ ] Foundation / B Corp structure

---

## How to Contribute to the Roadmap

1. **Pick a milestone** — choose something that excites you
2. **Open an issue** — describe what you want to build
3. **Claim a good first issue** — look for the `good first issue` label
4. **Submit a PR** — keep it small, focused, and tested

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## Long-Term Vision (2027+)

- **Bonsai Brain as an OS layer** — agents mediate between human intent and system capabilities
- **Global mesh network** — thousands of tiny agents on edge devices, collaborating
- **Self-improving agents** — agents that write their own tools and train their own models
- **Zero-config deployment** — `curl | sh` and you have a swarm node running in 30 seconds
- **Universal translator** — any CLI tool becomes an agent tool with one command

> *"The best time to plant a bonsai was 20 years ago. The second best time is now."*
