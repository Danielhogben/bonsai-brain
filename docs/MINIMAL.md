# Minimal Deployment Guide

Deploy Bonsai Brain v3 on hardware as small as a **Raspberry Pi Zero** (512 MB RAM) or a **RISC-V SBC** with no GPU, no Docker, and no Python.

---

## Binary Size

| Target | Size | Notes |
|--------|------|-------|
| `linux/amd64` | ~1.4 MB | Stripped, static |
| `linux/arm64` | ~1.4 MB | Pi 4, Apple Silicon |
| `linux/armv6` | ~1.4 MB | **Pi Zero / Zero 2 W** |
| `linux/riscv64` | ~1.4 MB | LicheeRV, Milk-V |
| `js/wasm` | ~2.0 MB | Browser / edge worker |

---

## Quick Deploy — Pi Zero

On your build machine:

```bash
cd bonsai-brain
make linux-armv6   # produces dist/bonsai-linux-armv6
scp dist/bonsai-linux-armv6 pi@raspberrypi.local:~/
```

On the Pi:

```bash
chmod +x bonsai-linux-armv6
./bonsai-linux-armv6 chat
```

Memory usage on Pi Zero 2 W: **~3–5 MB RSS** for the agent binary.

---

## Zero-Dependency Vector Search

For embedded RAG without calling an external embedding API:

```go
import (
    "github.com/donn/bonsai-brain/pkg/vector"
    "github.com/donn/bonsai-brain/pkg/embed"
)

// HashEmbedder: 128-dim, deterministic, zero dependencies
store := vector.NewStore(embed.NewHashEmbedder(128))
store.AddText("doc1", "Bonsai Brain runs on tiny hardware", nil)

results, _ := store.SearchText("small device agent", 3)
```

The `HashEmbedder` uses character bi-gram hashing — not SOTA, but **0 KB model download** and **<1 ms per document**.

---

## Pi Zero Systemd Service

```bash
sudo tee /etc/systemd/system/bonsai.service > /dev/null <<EOF
[Unit]
Description=Bonsai Brain Agent
After=network.target

[Service]
Type=simple
ExecStart=/home/pi/bonsai chat
WorkingDirectory=/home/pi
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now bonsai
```

---

## Memory Budget

| Component | RAM |
|-----------|-----|
| Bonsai Brain binary | ~5 MB |
| Per-agent context | ~50 KB |
| Vector store (1k docs) | ~2 MB |
| llama.cpp 0.5b model | ~300 MB |
| **Total** | **~310 MB** |

Fits comfortably on a 512 MB Pi Zero 2 W with room to spare.

---

## Build Flags for Minimal Size

```bash
# Standard stripped build
CGO_ENABLED=0 go build -ldflags="-s -w" -o bonsai ./cmd/bonsai

# Ultra-tiny (trimpath removes module paths)
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o bonsai ./cmd/bonsai
```

Optional: compress with UPX for another ~40% size reduction:

```bash
upx --best bonsai
```
