#!/bin/bash
# Deploy Bonsai Brain to Raspberry Pi Zero / Zero 2 W
# Run this ON THE PI after copying the binary.

set -e

BINARY="${1:-./bonsai-linux-armv6}"
TARGET_DIR="${2:-$HOME/bonsai}"

echo "🌳 Bonsai Brain v3 — Pi Zero Deploy"
echo "Binary: $BINARY"
echo "Target: $TARGET_DIR"
echo ""

mkdir -p "$TARGET_DIR"
cp "$BINARY" "$TARGET_DIR/bonsai"
chmod +x "$TARGET_DIR/bonsai"

# Create minimal systemd service
cat > "$TARGET_DIR/bonsai.service" << 'EOF'
[Unit]
Description=Bonsai Brain Agent
After=network.target

[Service]
Type=simple
ExecStart=/home/pi/bonsai/bonsai chat
WorkingDirectory=/home/pi/bonsai
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
EOF

echo "Install as systemd service:"
echo "  sudo cp $TARGET_DIR/bonsai.service /etc/systemd/system/"
echo "  sudo systemctl daemon-reload"
echo "  sudo systemctl enable --now bonsai"
echo ""
echo "Run manually:"
echo "  $TARGET_DIR/bonsai chat"
