package main

import (
"flag"
"fmt"
"os"
"os/signal"
"runtime"
"syscall"

"rxminer/miner"
)

func main() {
var (
rpcURL  = flag.String("rpc", "http://127.0.0.1:8545", "Daemon RPC URL")
address = flag.String("address", "", "Mining address (where rewards will be sent) - REQUIRED")
threads = flag.Int("threads", 0, "Number of mining threads (default: CPU cores)")
boost   = flag.Bool("boost", true, "Enable performance optimizations")
)
flag.Parse()

// REQUIRE ADDRESS
if *address == "" {
fmt.Println("╔════════════════════════════════════════════════════════════╗")
fmt.Println("║                    ADDRESS REQUIRED                        ║")
fmt.Println("╠════════════════════════════════════════════════════════════╣")
fmt.Println("║  You must specify a mining address to receive rewards!     ║")
fmt.Println("║                                                            ║")
fmt.Println("║  Usage:                                                    ║")
fmt.Println("║    ./rxminer -address 0xYourWalletAddress                  ║")
fmt.Println("║                                                            ║")
fmt.Println("║  Example:                                                  ║")
fmt.Println("║    ./rxminer -address 0xc40F4A0b4df81F8f67A88B179a8b2271107a9ac2 -threads 2 -boost")
fmt.Println("║                                                            ║")
fmt.Println("║  Get help:                                                 ║")
fmt.Println("║    ./rxminer --help                                        ║")
fmt.Println("╚════════════════════════════════════════════════════════════╝")
os.Exit(1)
}

// Validate address format (starts with 0x and has 40 chars after)
if len(*address) != 42 || (*address)[:2] != "0x" {
fmt.Println("❌ ERROR: Invalid address format!")
fmt.Println("   Address must be a valid Ethereum address (0x followed by 40 hex characters)")
fmt.Printf("   Got: %s\n", *address)
os.Exit(1)
}

fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Miner v1.0                      ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Printf("RPC URL: %s\n", *rpcURL)
fmt.Printf("Mining Address: %s\n", *address)
fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
if *threads > 0 {
fmt.Printf("Threads: %d\n", *threads)
} else {
fmt.Printf("Threads: auto (%d)\n", runtime.NumCPU())
}
if *boost {
fmt.Println("Boost: ENABLED ⚡")
} else {
fmt.Println("Boost: disabled")
}
fmt.Println()

m, err := miner.NewMiner(*rpcURL, *address, *threads, *boost)
if err != nil {
fmt.Printf("Failed to create miner: %v\n", err)
os.Exit(1)
}

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
<-sigChan
fmt.Println("\n⚠️  Shutting down miner...")
m.Stop()
os.Exit(0)
}()

m.Start()
}
