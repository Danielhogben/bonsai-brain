#!/bin/bash
# OpenCode Multi-Agent Wrapper
# Usage: source this file, then run agent commands

set -a
source ~/.hermes/.env 2>/dev/null || true
set +a

# Update opencode config with all keys and a reliable default
cat > ~/.opencode/config.yaml << CONFIG
model: openrouter/meta-llama/llama-3.3-70b-instruct:free
providers:
  openrouter:
    api_key: ${OPENROUTER_API_KEY}
  kimi-for-coding:
    api_key: ${KIMI_API_KEY}
  openai:
    api_key: ${VOICE_TOOLS_OPENAI_KEY}
CONFIG

# Agent functions - each uses a different model
opencode-groq-fast() { opencode run -m openrouter/meta-llama/llama-3.3-70b-instruct:free "$@"; }
opencode-groq-small() { opencode run -m openrouter/meta-llama/llama-3.2-3b-instruct:free "$@"; }
opencode-gpt-oss() { opencode run -m openrouter/openai/gpt-oss-20b:free "$@"; }
opencode-gpt-oss-large() { opencode run -m openrouter/openai/gpt-oss-120b:free "$@"; }
opencode-gemma() { opencode run -m openrouter/google/gemma-3-4b-it:free "$@"; }
opencode-gemma-large() { opencode run -m openrouter/google/gemma-3-27b-it:free "$@"; }
opencode-deepseek() { opencode run -m openrouter/deepseek/deepseek-v4-flash "$@"; }
opencode-nemotron() { opencode run -m openrouter/nvidia/nemotron-3-super-120b-a12b:free "$@"; }
opencode-nemotron-nano() { opencode run -m openrouter/nvidia/nemotron-nano-9b-v2:free "$@"; }
opencode-hermes() { opencode run -m openrouter/nousresearch/hermes-3-llama-3.1-405b:free "$@"; }
opencode-glm() { opencode run -m openrouter/z-ai/glm-4.5-air:free "$@"; }
opencode-minimax() { opencode run -m openrouter/minimax/minimax-m2.5:free "$@"; }
opencode-kimi() { opencode run -m kimi-for-coding/k2p5 "$@"; }
opencode-openai() { opencode run -m openai/gpt-4o-mini "$@"; }

export -f opencode-groq-fast opencode-groq-small opencode-gpt-oss opencode-gpt-oss-large
export -f opencode-gemma opencode-gemma-large opencode-deepseek opencode-nemotron
export -f opencode-nemotron-nano opencode-hermes opencode-glm opencode-minimax
export -f opencode-kimi opencode-openai

echo "✅ OpenCode config updated with all API keys"
echo ""
echo "🤖 Available agent commands:"
echo "  opencode-groq-fast      — Llama 3.3 70B (fastest, ~130ms)"
echo "  opencode-groq-small     — Llama 3.2 3B (tiny, ultra-fast)"
echo "  opencode-gpt-oss        — GPT-OSS 20B (OpenAI open weights)"
echo "  opencode-gpt-oss-large  — GPT-OSS 120B (most capable)"
echo "  opencode-gemma          — Gemma 3 4B (small but smart)"
echo "  opencode-gemma-large    — Gemma 3 27B (biggest Gemma)"
echo "  opencode-deepseek       — DeepSeek v4 Flash (coding)"
echo "  opencode-nemotron       — Nemotron 3 Super 120B (NVIDIA)"
echo "  opencode-nemotron-nano  — Nemotron Nano 9B (tiny NVIDIA)"
echo "  opencode-hermes         — Hermes 3 405B (Nous Research)"
echo "  opencode-glm            — GLM 4.5 Air (Zhipu AI)"
echo "  opencode-minimax        — MiniMax M2.5 (Chinese LLM)"
echo "  opencode-kimi           — Kimi K2.5 (Moonshot AI)"
echo "  opencode-openai         — GPT-4o Mini (OpenAI)"
echo ""
echo "Usage example:"
echo '  opencode-groq-fast "Explain Go channels in 2 sentences"'
echo '  opencode-deepseek "Write a Python function to sort a list"'
echo '  opencode-kimi "Translate this to Chinese: Hello world"'
