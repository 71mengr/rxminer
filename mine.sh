#!/bin/bash

# RandomX Mining Script
# Usage: ./mine.sh [address] [threads] [rpc_url]

# Default values
DEFAULT_ADDRESS="0xc40F4A0b4df81F8f67A88B179a8b2271107a9ac2"
DEFAULT_THREADS=2
DEFAULT_RPC="http://127.0.0.1:8545"

# Use provided values or defaults
ADDRESS=${1:-$DEFAULT_ADDRESS}
THREADS=${2:-$DEFAULT_THREADS}
RPC_URL=${3:-$DEFAULT_RPC}

echo "========================================="
echo "     RandomX Mining Script"
echo "========================================="
echo "Mining Address: $ADDRESS"
echo "Threads: $THREADS"
echo "RPC URL: $RPC_URL"
echo "========================================="
echo ""

# Run the miner
./rxminer -address "$ADDRESS" -threads "$THREADS" -boost -rpc "$RPC_URL"
