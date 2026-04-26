#!/bin/bash
# Deploy Bonsai Brain to RISC-V nano boards (LicheeRV Nano, Milk-V Duo, etc.)

set -e

BINARY="${1:-./bonsai-linux-riscv64}"
TARGET_DIR="${2:-$HOME/bonsai}"

echo "🌳 Bonsai Brain v3 — RISC-V Nano Deploy"
echo "Binary: $BINARY"
echo "Target: $TARGET_DIR"
echo ""

mkdir -p "$TARGET_DIR"
cp "$BINARY" "$TARGET_DIR/bonsai"
chmod +x "$TARGET_DIR/bonsai"

cat > "$TARGET_DIR/bonsai.service" << 'EOF'
[Unit]
Description=Bonsai Brain Agent
After=network.target

[Service]
Type=simple
ExecStart=/root/bonsai/bonsai chat
WorkingDirectory=/root/bonsai
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

echo "Install as systemd service:"
echo "  cp $TARGET_DIR/bonsai.service /etc/systemd/system/"
echo "  systemctl daemon-reload"
echo "  systemctl enable --now bonsai"
