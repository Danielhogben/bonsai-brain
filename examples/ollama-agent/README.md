# Bonsai Brain + Ollama Integration

Run Bonsai Brain agents entirely offline using Ollama as the model backend.

## Requirements

- [Ollama](https://ollama.com) installed and running
- A local model (0.5B–3B recommended for small hardware)

## Quick Start

### 1. Start Ollama

```bash
ollama serve
```

### 2. Pull a tiny model

```bash
# For Pi Zero / 512 MB RAM devices
ollama pull qwen2.5:0.5b

# For better tool-calling reliability
ollama pull qwen2.5:1.5b
```

### 3. Create a custom model (optional)

If you have a GGUF file:

```bash
cat > Modelfile <<EOF
FROM /path/to/qwen2.5-0.5b.gguf
PARAMETER temperature 0.3
PARAMETER num_ctx 2048
SYSTEM You are a helpful assistant. Keep responses brief.
EOF

ollama create qwen2.5-tiny -f Modelfile
```

### 4. Run the demo

```bash
cd examples/ollama-agent
go run main.go
```

## Model Size Recommendations

| Hardware | Model | RAM | Tool Calling |
|----------|-------|-----|--------------|
| Pi Zero 2 W (512 MB) | `qwen2.5:0.5b` | ~400 MB | Unreliable |
| Pi 4 (2–4 GB) | `qwen2.5:1.5b` | ~1 GB | Fair |
| Pi 5 / x86 (4+ GB) | `llama3.2:3b` | ~2 GB | Good |
| Desktop (8+ GB) | `qwen2.5:7b` | ~5 GB | Excellent |

## Architecture

```
┌─────────────┐     /api/chat      ┌──────────┐     GGUF    ┌────────┐
│ Bonsai Brain │ ──────────────────→ │  Ollama  │ ─────────→ │  CPU   │
│   Agent      │    (native API)     │  Server  │             │  /GPU  │
└─────────────┘                     └──────────┘             └────────┘
```

The `pkg/ollama` client uses Ollama's native `/api/chat` endpoint and
implements tool calling via prompt engineering — no OpenAI compatibility
layer required.
