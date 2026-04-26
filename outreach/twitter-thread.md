# Twitter/X Thread

**Tweet 1:**
I built an agent framework that runs 31 FREE LLMs simultaneously and fits on a Raspberry Pi Zero.

It's called Bonsai Brain 🌳

Thread ↓

---

**Tweet 2:**
The problem: every agent framework needs Docker, 16GB RAM, and a week to configure.

The solution: a 5MB Go binary that cross-compiles to EVERY architecture.

Including WASM. Including RISC-V. Including ARMv6 (Pi Zero).

---

**Tweet 3:**
```bash
$ bonsai swarm --prompt "Explain swarm intelligence"

49 agents spawned
31 successful responses
Fastest: 130ms (Groq llama-3.1-8b)
Consensus winner selected automatically
```

---

**Tweet 4:**
Providers working RIGHT NOW:
• Groq — 6 models, all blazing fast
• OpenRouter — 14 free models including GPT-OSS 120B
• NVIDIA — DeepSeek v4, Gemma 3
• Cohere — Command R, Command A
• Gemini — 2.5 Flash
• Local — llama.cpp + Ollama

All free tier. No credit card required.

---

**Tweet 5:**
Architecture highlights:
• Pure Go vector store — no cgo, no DB
• Zero-dep embedders — <1ms per doc
• Streaming tool calls with approval gates
• Hierarchical agents with depth limits
• Plugin system inspired by elizaOS

---

**Tweet 6:**
The dream: thousands of tiny specialized agents running on edge devices, collaborating through open protocols.

Your Pi Zero talks to your laptop talks to a VPS. A global brain made of tiny brains.

---

**Tweet 7:**
I need help building this.

→ Go developers
→ Tool builders (wrap your favorite CLI!)
→ TUI devs (Bubble Tea)
→ Benchmarkers

Repo: github.com/donn/bonsai-brain
MIT licensed. Star ⭐ if tiny agents are the future.
