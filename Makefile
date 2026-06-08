.PHONY: all clean build-solo build-pool run-solo run-pool help

RANDOMX_DIR = $(HOME)/RandomX
RANDOMX_BUILD = $(RANDOMX_DIR)/build
MINER_DIR = $(shell pwd)

export GO111MODULE = on

all: build-solo build-pool

help:
	@echo "Available targets:"
	@echo "  make build-solo  - Build solo miner (direct to daemon)"
	@echo "  make build-pool  - Build pool miner"
	@echo "  make run-solo    - Run solo miner"
	@echo "  make run-pool    - Run pool miner"
	@echo "  make clean       - Clean build artifacts"
	@echo ""
	@echo "Solo mining:"
	@echo "  ./rxminer -address 0xYourAddress -rpc http://127.0.0.1:8545"
	@echo ""
	@echo "Pool mining:"
	@echo "  ./rxminer-pool -pool pool.example.com:3333 -address 0xYourAddress"

build-randomx:
	@if [ ! -f "$(RANDOMX_BUILD)/librandomx.a" ]; then \
		echo "Building RandomX..."; \
		cd $(RANDOMX_DIR) && mkdir -p build && cd build && cmake -DARCH=native .. && make -j$(nproc); \
	fi

build-solo: build-randomx
	@echo "=== Building Solo Miner ==="
	@CGO_ENABLED=1 CGO_CFLAGS="-I$(MINER_DIR)/randomx -I$(RANDOMX_DIR)/src" CGO_LDFLAGS="-L$(RANDOMX_BUILD) -lrandomx -lstdc++ -lm" \
		go build -tags "cgo randomx" -o rxminer ./main.go
	@echo "✅ Solo miner built"

build-pool: build-randomx
	@echo "=== Building Pool Miner ==="
	@CGO_ENABLED=1 CGO_CFLAGS="-I$(MINER_DIR)/randomx -I$(RANDOMX_DIR)/src" CGO_LDFLAGS="-L$(RANDOMX_BUILD) -lrandomx -lstdc++ -lm" \
		go build -tags "cgo randomx" -o rxminer-pool ./cmd/pool/main.go
	@echo "✅ Pool miner built"

run-solo: build-solo
	./rxminer -address 0xc40F4A0b4df81F8f67A88B179a8b2271107a9ac2 -threads 2 -boost

run-pool: build-pool
	@echo "Usage: ./rxminer-pool -pool POOL_URL -address YOUR_ADDRESS"

clean:
	@rm -f rxminer rxminer-pool
	@echo "✅ Clean complete"

build-pool-server: build-randomx
	@echo "=== Building Pool Server ==="
	@CGO_ENABLED=1 CGO_CFLAGS="-I$(MINER_DIR)/randomx -I$(RANDOMX_DIR)/src" CGO_LDFLAGS="-L$(RANDOMX_BUILD) -lrandomx -lstdc++ -lm" \
		go build -tags "cgo randomx" -o rxpool ./cmd/pool/main.go
	@echo "✅ Pool server built"

run-pool-server: build-pool-server
	@./rxpool -config config_pool.json -webport 8080
