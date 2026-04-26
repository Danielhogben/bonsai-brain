# Discord Announcement

**Channels to post:**
- LocalLLaMA Discord #showcase
- Go Discord #show-and-tell
- AI Engineers Discord #projects
- Hacking With Friends #ai-projects

**Message:**

🌳 **Bonsai Brain** — the tiniest agent framework in the world

I just shipped v0.3.0 with a **full cloud swarm stack**. One command spawns 49 agents across 7 providers and runs them all in parallel.

**What's working:**
✅ 31 free-tier models (Groq, OpenRouter, NVIDIA, Cohere, Gemini, local)
✅ Per-provider rate limiting
✅ Result aggregation (first/fastest/consensus)
✅ 5MB binary, runs on Pi Zero
✅ `bonsai swarm` CLI command

**Stack:** Go 1.22, no cgo, pure Go vector store, zero-dep embedders

**Repo:** https://github.com/donn/bonsai-brain

**I need contributors for:**
• More provider integrations
• Tool wrappers (git, docker, filesystem)
• Bubble Tea TUI
• Benchmarks

Drop a ⭐ and come build with us!
