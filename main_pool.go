package main

import (
"flag"
"fmt"
"os"
"os/signal"
"runtime"
"syscall"
)

func main() {
var (
poolURL  = flag.String("pool", "", "Pool URL (e.g., pool.example.com:3333)")
address  = flag.String("address", "", "Your wallet address")
password = flag.String("password", "x", "Pool password")
threads  = flag.Int("threads", 0, "Number of mining threads")
boost    = flag.Bool("boost", true, "Enable performance optimizations")
)
flag.Parse()

if *poolURL == "" || *address == "" {
fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Pool Miner v1.0                 ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Println()
fmt.Println("ERROR: Pool URL and wallet address are required!")
fmt.Println()
fmt.Println("Usage:")
fmt.Println("  ./rxminer-pool -pool pool.example.com:3333 -address 0xYourAddress")
fmt.Println()
fmt.Println("Example:")
fmt.Println("  ./rxminer-pool -pool pool.supportxmr.com:3333 -address 0xYourWallet -threads 4")
os.Exit(1)
}

fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Pool Miner v1.0                 ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Printf("Pool URL: %s\n", *poolURL)
fmt.Printf("Wallet Address: %s\n", *address)
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

m, err := NewPoolMiner(*poolURL, *address, *password, *threads, *boost)
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
