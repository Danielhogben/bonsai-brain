# Reddit Post — r/LocalLLaMA

**Title:** I built an agent framework that runs 31 free LLMs simultaneously — and it fits on a Pi Zero

**Body:**

Hey r/LocalLLaMA!

I've been frustrated by agent frameworks that need Docker, 16GB RAM, and a PhD to set up. So I built **Bonsai Brain** — a Go-based agent framework that's:

- **Tiny**: 5 MB binary, <10 MB RAM without model
- **Fast**: Groq agents respond in 130ms
- **Distributed**: Spawns agents across OpenRouter, Groq, NVIDIA, Gemini, Cohere, Ollama, and local llama.cpp — all at once
- **Free**: Uses only free-tier models (31 working agents right now)
- **No cgo, no external dependencies** for the core

**What it does:**

```bash
bonsai swarm --prompt "Explain quantum computing in 2 sentences"
# Dispatches to 49 agents in parallel
# Returns results ranked by speed + a consensus winner
```

Live demo showing 31 models responding simultaneously: [GitHub link]

**Why Go?** Single binary, cross-compiles to every architecture (including WASM), starts in 50ms. I run it on a Pi Zero 2 W alongside a 237MB 1-bit model.

**The stack:**
- Core engine with streaming tool calls and approval gates
- Pure-Go vector store (no cgo, no external DB)
- Zero-dep embedders for edge devices
- Hierarchical agents with depth limits
- Plugin system inspired by elizaOS

**I need help with:**
- More provider integrations (Together AI, Fireworks, Cerebras)
- CLI tool wrappers (Git, Docker, filesystem)
- TUI with Bubble Tea
- Benchmarks against LangChain, CrewAI, etc.

Repo: https://github.com/donn/bonsai-brain

MIT licensed. Star it if you think tiny agents are the future 🌱

---
*P.S. — Yes, it actually runs on a Pi Zero. 512MB RAM total. Video coming soon.*
