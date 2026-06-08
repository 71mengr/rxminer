#!/bin/bash

echo "Building RandomX miner with optimizations..."

# Enable large pages for better performance (requires root)
if [ "$EUID" -eq 0 ]; then
    echo "Configuring large pages for better performance..."
    echo 1280 > /proc/sys/vm/nr_hugepages
    echo "Large pages configured: $(cat /proc/sys/vm/nr_hugepages)"
fi

# Set RandomX library path
export CGO_CFLAGS="-I${HOME}/go-ethereum/build/_workspace/randomx/src"
export CGO_LDFLAGS="-L${HOME}/go-ethereum/build/_workspace/randomx/build -lrandomx -lstdc++ -lm"

go build -tags "cgo randomx" -ldflags="-s -w" -o rxminer ./main.go

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Build successful!"
    echo ""
    echo "Usage:"
    echo "  ./rxminer --help"
    echo "  ./rxminer -address 0xYourAddress -rpc http://127.0.0.1:8545 -threads 4 -boost"
    echo "  ./rxminer -pool pool.example.com:3333 -address 0xYourAddress -threads 4 -boost"
    echo ""
    echo "To enable large pages (run as root):"
    echo "  sudo ./rxminer -address 0xYourAddress -boost"
else
    echo "❌ Build failed!"
    exit 1
fi
