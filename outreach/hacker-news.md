# Hacker News Post

**Title:** Show HN: Bonsai Brain — a 5MB agent framework that runs 31 free LLMs in parallel

**Body:**

Bonsai Brain is a Go-based agent framework designed for constrained hardware and maximum provider coverage.

**Key numbers:**
- Binary: 5 MB (stripped)
- RAM: ~10 MB without model
- Cold start: 50 ms
- Working free-tier models: 31 across 7 providers
- Fastest response: 130 ms (Groq llama-3.1-8b)

**The swarm command** distributes a task to every configured model in parallel, then aggregates results by first response, fastest response, or consensus:

```bash
$ bonsai swarm --prompt "What is 2+2?"
# 49 agents spawned
# 31 successful responses
# Wall-clock time: 92s (rate-limited for politeness)
```

**Architecture:**
- Core: streaming reasoning loop with typed tool registry
- Memory: conversation summarization + pure-Go vector store
- Plugins: Actions, Providers, Evaluators, Services (elizaOS-inspired)
- Agents: hierarchical with middleware/guardrail pipelines
- Swarm: multi-provider registry with per-provider rate limiting

**Built for:**
- Edge devices (Pi Zero, old laptops)
- Privacy-first deployments (local models default)
- Agent researchers who need to compare 30+ models quickly

**Not built for:**
- GPU-heavy training
- Replacing Claude Code (yet)
- People who love Docker

Code: https://github.com/donn/bonsai-brain | MIT

Looking for contributors who want to build the tiniest, most capable agent framework in the world.
