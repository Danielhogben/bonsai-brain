# Bonsai Brain + PrismML Bonsai Model

Run the **PrismML Bonsai 1.7B 1-bit** model locally through Bonsai Brain.

## Model

| Property | Value |
|----------|-------|
| Name | Bonsai-1.7B-Q1_0 |
| Parameters | 1.7 billion |
| Quantization | 1-bit (Q1_0) |
| File size | ~237 MB |
| RAM usage | ~350 MB @ 2K context |
| Source | [prism-ml/Bonsai-1.7B-gguf](https://huggingface.co/prism-ml/Bonsai-1.7B-gguf) |

## Requirements

- llama.cpp server (build 8925+ with Q1_0 support)
- The GGUF file downloaded from HuggingFace

## Quick Start

### 1. Download the model

```bash
wget https://huggingface.co/prism-ml/Bonsai-1.7B-gguf/resolve/main/Bonsai-1.7B-Q1_0.gguf \
  -O ~/llama-models/Bonsai-1.7B-Q1_0.gguf
```

### 2. Start llama-server

```bash
llama-server -m ~/llama-models/Bonsai-1.7B-Q1_0.gguf \
  --port 11434 -np 2 --ctx-size 2048
```

### 3. Run the demo

```bash
cd examples/prism-bonsai
go run main.go
```

## Ollama Support

Ollama 0.20.5 (the version installed on this machine) is too old to load Q1_0 models.
Upgrade to **Ollama v0.5+** for 1-bit support:

```bash
curl -fsSL https://ollama.com/install.sh | sh
ollama create bonsai-1.7b -f Modelfile
ollama run bonsai-1.7b
```

## Other PrismML Models

| Model | Size | Best For |
|-------|------|----------|
| Bonsai-1.7B | 237 MB | Pi Zero, edge devices |
| Bonsai-4B | ~600 MB | Pi 4, mobile |
| Bonsai-8B | ~1.1 GB | Desktop, server |
| Ternary-Bonsai-1.7B | ~300 MB | Better quality than 1-bit |
| Ternary-Bonsai-4B | ~800 MB | Balanced quality/size |
| Ternary-Bonsai-8B | ~1.6 GB | Best quality, still tiny |

All available at [huggingface.co/prism-ml](https://huggingface.co/prism-ml).
