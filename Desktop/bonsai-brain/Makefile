# Bonsai Brain v3 — Minimal Build Makefile
# Targets every small device from Pi Zero to WASM edge workers.

BINARY := bonsai
VERSION := 0.3.0
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
TINYFLAGS := -ldflags="-s -w -X main.version=$(VERSION)" -trimpath

.PHONY: all build build-tiny test clean \
	linux-amd64 linux-arm64 linux-armv7 linux-armv6 \
	riscv64 windows-amd64 darwin-amd64 darwin-arm64 wasm \
	deploy-pi-zero deploy-nano size-report

all: build test

# Standard build
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/bonsai

# Minimal stripped build
build-tiny:
	CGO_ENABLED=0 go build $(TINYFLAGS) -o $(BINARY) ./cmd/bonsai

# Run tests
test:
	go test -count=1 ./pkg/...

# Clean artifacts
clean:
	rm -f $(BINARY) $(BINARY)-* dist/*.zip

# Size report
size-report: build-tiny
	@echo "Binary size:"
	@ls -lh $(BINARY)
	@echo ""
	@echo "Sections:"
	@size $(BINARY) 2>/dev/null || echo "size tool not available"

# ---------------------------------------------------------------------------
# Cross-compilation for small targets
# ---------------------------------------------------------------------------

linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(TINYFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/bonsai

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(TINYFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/bonsai

# Raspberry Pi 3/4 (32-bit)
linux-armv7:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build $(TINYFLAGS) -o dist/$(BINARY)-linux-armv7 ./cmd/bonsai

# Raspberry Pi Zero / Zero 2 W
linux-armv6:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 go build $(TINYFLAGS) -o dist/$(BINARY)-linux-armv6 ./cmd/bonsai

# RISC-V SBCs (e.g. Lichee Pi, Milk-V)
riscv64:
	CGO_ENABLED=0 GOOS=linux GOARCH=riscv64 go build $(TINYFLAGS) -o dist/$(BINARY)-linux-riscv64 ./cmd/bonsai

windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(TINYFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/bonsai

darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(TINYFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/bonsai

darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(TINYFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/bonsai

# WASM for edge workers / browsers
wasm:
	CGO_ENABLED=0 GOOS=js GOARCH=wasm go build $(TINYFLAGS) -o dist/$(BINARY).wasm ./cmd/bonsai

# Build all small-target binaries
build-all: linux-amd64 linux-arm64 linux-armv7 linux-armv6 riscv64 windows-amd64 wasm
	@echo "All targets built in dist/"
	@ls -lh dist/

# ---------------------------------------------------------------------------
# Deployment helpers
# ---------------------------------------------------------------------------

dist:
	mkdir -p dist

# Package for Pi Zero deployment
deploy-pi-zero: dist linux-armv6
	zip -j dist/bonsai-pi-zero-v$(VERSION).zip \
		dist/$(BINARY)-linux-armv6 \
		scripts/deploy-pi-zero.sh

# Package for RISC-V nano boards
deploy-nano: dist riscv64
	zip -j dist/bonsai-nano-v$(VERSION).zip \
		dist/$(BINARY)-linux-riscv64 \
		scripts/deploy-nano.sh
