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
rpcURL   = flag.String("rpc", "http://127.0.0.1:8545", "Daemon RPC URL for solo mining")
poolURL  = flag.String("pool", "", "Pool URL for pool mining (e.g., pool.example.com:3333)")
address  = flag.String("address", "", "Mining address (where rewards will be sent) - REQUIRED")
password = flag.String("password", "x", "Pool password")
threads  = flag.Int("threads", 0, "Number of mining threads (default: CPU cores)")
boost    = flag.Bool("boost", true, "Enable performance optimizations")
)
flag.Parse()

// REQUIRE ADDRESS
if *address == "" {
printUsage("ADDRESS REQUIRED", "You must specify a mining address to receive rewards!")
os.Exit(1)
}

soloMining := *poolURL == ""

if soloMining {
// Validate address format (starts with 0x and has 40 chars after)
if len(*address) != 42 || (*address)[:2] != "0x" {
fmt.Println("❌ ERROR: Invalid address format!")
fmt.Println("   Address must be a valid Ethereum address (0x followed by 40 hex characters)")
fmt.Printf("   Got: %s\n", *address)
os.Exit(1)
}
}

if soloMining {
fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Miner v1.0                      ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Printf("Mode: solo\n")
fmt.Printf("RPC URL: %s\n", *rpcURL)
fmt.Printf("Mining Address: %s\n", *address)
} else {
fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Miner v1.0                      ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Printf("Mode: pool\n")
fmt.Printf("Pool URL: %s\n", *poolURL)
fmt.Printf("Wallet Address: %s\n", *address)
}
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

var m interface{
Start()
Stop()
}
var err error

if soloMining {
m, err = miner.NewMiner(*rpcURL, *address, *threads, *boost)
} else {
m, err = miner.NewPoolMiner(*poolURL, *address, *password, *threads, *boost)
}
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

func printUsage(title, message string) {
fmt.Println("╔════════════════════════════════════════════════════════════╗")
fmt.Printf("║                    %-40s║\n", title)
fmt.Println("╠════════════════════════════════════════════════════════════╣")
fmt.Printf("║  %-56s║\n", message)
fmt.Println("║                                                            ║")
fmt.Println("║  Solo mining:                                              ║")
fmt.Println("║    ./rxminer -address 0xYourWalletAddress                  ║")
fmt.Println("║    ./rxminer -address 0xYourWalletAddress -rpc http://127.0.0.1:8545")
fmt.Println("║                                                            ║")
fmt.Println("║  Pool mining:                                              ║")
fmt.Println("║    ./rxminer -pool pool.example.com:3333 -address 0xYourWalletAddress")
fmt.Println("║    ./rxminer -pool pool.example.com:3333 -address 0xYourWalletAddress -threads 4")
fmt.Println("║                                                            ║")
fmt.Println("║  Get help:                                                 ║")
fmt.Println("║    ./rxminer --help                                        ║")
fmt.Println("╚════════════════════════════════════════════════════════════╝")
}
