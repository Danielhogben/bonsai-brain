# 🌳 Bonsai Brain — The Dream Board

> *An agent framework so small it runs on a Pi Zero, so capable it rivals cloud stacks.*

---

## The Vision

**Bonsai Brain** is not just another agent framework. It is the distillation of everything we learned from Claude Code, Codex CLI, Kimi CLI, agent-zero, elizaOS, VoltAgent, and DeerFlow — reimagined for a world where:

- **Privacy is non-negotiable** — your data never leaves your device
- **Hardware is constrained** — 512 MB RAM should be enough for anyone
- **Agents are legion** — dozens of specialized agents working in concert
- **Modularity wins** — compose exactly what you need, nothing more

We believe the future of AI is not one massive model in the cloud, but thousands of tiny, specialized agents running on edge devices, collaborating through open protocols, orchestrated by a framework that fits in a 1.4 MB binary.

---

## The Dream: What We're Building

### 1. 🧠 The Engine (v0.3.0 ✅)
A reasoning loop so clean you can read it in one sitting. Streaming model calls. Typed tools with approval gates. Composable guardrails and middleware. Hierarchical agents with depth limits.

**Status:** Core complete. 8 packages, all tests passing.

### 2. 💾 The Memory (v0.3.0 ✅)
Conversation summarization that auto-compresses when context overflows. Pure-Go vector store with cosine similarity. Zero-dependency embedders (HashEmbedder, TFIDFEmbedder) that run in <1 ms.

**Status:** Working. In-process, no external DB.

### 3. 🔌 The Plugin System (v0.3.0 ✅)
ElizaOS-inspired: Actions, Providers, Evaluators, Services. Thread-safe registry. Fast lookup. Compose providers with error tolerance.

**Status:** Core architecture complete.

### 4. 🌐 The Swarm (v0.4.0 — In Progress)
Multiple Bonsai Brain instances talking to each other. Agent-to-agent (A2A) protocol. Task distribution. Consensus mechanisms. Load balancing across a fleet of Pi Zeros.

**Status:** Design phase. Researching MCP and A2A protocols.

### 5. 🛠️ The Tool Ecosystem (v0.4.0 — In Progress)
Pre-built integrations for 50+ CLI tools and APIs. Filesystem, web search, Git, Docker, databases, messaging platforms. Each tool is a standalone package you `go get`.

**Status:** 1 tool (`get_hostname`) 😅 — we need help here!

### 6. 🎨 The UI (v0.5.0)
A terminal UI (TUI) built with Charm Bubble Tea. Real-time agent state visualization. Streaming output. Interactive approval prompts. Multi-agent dashboard.

**Status:** Not started. Looking for contributors.

### 7. 📦 The Distribution (v0.5.0)
`go install` for the framework. `apt install bonsai-brain` for Debian/Ubuntu. Docker images for `linux/amd64`, `linux/arm64`, `linux/arm/v6`. Homebrew formula for macOS. WASM builds for the browser.

**Status:** Makefile has cross-compile targets. Need packaging.

### 8. 🧪 The Benchmarks (v0.6.0)
Standardized benchmarks against other agent frameworks. Performance on edge devices. Token efficiency comparisons. Latency measurements. A public leaderboard.

**Status:** Not started.

### 9. 📚 The Academy (Ongoing)
Tutorials. Example agents. Video walkthroughs. A "Build Your First Agent in 5 Minutes" guide. Community-contributed plugins showcased in a gallery.

**Status:** 3 examples exist. Need many more.

### 10. 🌍 The Collective (The Dream)
A decentralized network of Bonsai Brain instances. Your Pi Zero at home talks to your VPS in the cloud talks to your friend's laptop. Agents migrate to where compute is cheapest. A global brain made of tiny brains.

**Status:** This is the 10-year vision. Let's build the foundation first.

---

## The Philosophy

| Principle | What It Means |
|-----------|---------------|
| **Small** | Binary < 2 MB. RAM < 10 MB without model. No cgo. No external dependencies for core. |
| **Fast** | Tool calls in < 10 ms. Vector search in < 1 ms. Startup in < 100 ms. |
| **Clear** | Code you can read and understand. No hidden magic. No 500-line functions. |
| **Composable** | Use one package or all ten. Mix and match. No all-or-nothing. |
| **Open** | MIT license. Open protocols. Open models. Open data. |

---

## The Stack (What We Run On)

| Component | Technology | Why |
|-----------|-----------|-----|
| Language | Go 1.22+ | Fast compile, tiny binaries, great concurrency |
| Models | llama.cpp, Ollama, OpenRouter, Groq | Local first, cloud optional |
| Memory | In-process (embedded PostgreSQL optional) | Zero-config, no Docker required |
| Vectors | Pure Go | No cgo, no ML dependencies |
| UI | Charm Bubble Tea (planned) | Beautiful TUIs in Go |
| Protocols | OpenAI-compatible API, MCP, A2A | Interoperability |

---

## The Hardware Targets

| Device | RAM | Use Case |
|--------|-----|----------|
| Raspberry Pi Zero 2 W | 512 MB | Edge agent, IoT controller |
| Raspberry Pi 4/5 | 2-8 GB | Home server, multi-agent host |
| Old laptop | 4-16 GB | Development, local swarm node |
| VPS | 1-4 GB | Cloud agent, public API |
| Desktop/GPU | 16-64 GB | Heavy lifting, training, benchmarks |

---

## The Call to Action

We are building the **tiniest, most capable agent framework in the world**. But we can't do it alone.

### What We Need

- **Go developers** — core framework, performance optimization, new features
- **Tool builders** — integrations for CLI tools, APIs, services
- **UI/UX developers** — TUI, web dashboard, mobile interfaces
- **Documentation writers** — tutorials, API docs, blog posts
- **DevOps engineers** — CI/CD, packaging, deployment automation
- **Community builders** — Discord/Reddit moderation, event organization
- **Researchers** — benchmarks, novel algorithms, edge optimizations
- **Testers** — try it on your weird hardware, report bugs, write tests

### How to Start

1. **Star the repo** ⭐ — shows us you care
2. **Run the examples** — `cd examples/prism-bonsai && go run .`
3. **Pick an issue** — look for `good first issue` labels
4. **Build a tool** — wrap your favorite CLI tool as a Bonsai Brain plugin
5. **Share your agent** — show us what you built

---

## The Dream in Numbers

| Metric | Target | Current |
|--------|--------|---------|
| Binary size (stripped) | < 2 MB | ✅ 1.4 MB |
| Cold start time | < 100 ms | ✅ ~50 ms |
| Vector search (1K docs) | < 1 ms | ✅ ~0.1 ms |
| Tool call overhead | < 10 ms | ✅ ~2 ms |
| Pre-built tools | 50+ | 🚧 1 |
| Example agents | 20+ | 🚧 3 |
| Contributors | 100+ | 🚧 1 |
| Stars | 10,000+ | 🚧 ~0 |

---

## The Roadmap

See [ROADMAP.md](ROADMAP.md) for the detailed technical roadmap with milestones, deadlines, and task breakdowns.

---

## Join the Collective

- **GitHub**: [github.com/donn/bonsai-brain](https://github.com/donn/bonsai-brain)
- **Issues**: [github.com/donn/bonsai-brain/issues](https://github.com/donn/bonsai-brain/issues)
- **Discussions**: [github.com/donn/bonsai-brain/discussions](https://github.com/donn/bonsai-brain/discussions)

> *"The best time to plant a bonsai was 20 years ago. The second best time is now."*

Let's grow this together. 🌱
