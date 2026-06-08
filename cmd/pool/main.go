package main

import (
"flag"
"fmt"
"os"
"os/signal"
"syscall"

"rxminer/pool"
)

func main() {
configPath := flag.String("config", "config_pool.json", "Config file path")
webPort := flag.Int("webport", 8080, "Web interface port")
flag.Parse()

cfg, err := pool.LoadPoolConfig(*configPath)
if err != nil {
fmt.Printf("Failed to load config: %v\n", err)
os.Exit(1)
}

fmt.Println("╔═══════════════════════════════════════════════════╗")
fmt.Println("║           RandomX Mining Pool v1.0                ║")
fmt.Println("╚═══════════════════════════════════════════════════╝")
fmt.Printf("Pool Name: %s\n", cfg.Pool.Name)
fmt.Printf("Listen: %s\n", cfg.Pool.Listen)
fmt.Printf("Difficulty: %d\n", cfg.Pool.Difficulty)
fmt.Printf("Pool Fee: %d%%\n", cfg.Pool.Fee)
fmt.Printf("Daemon: %s\n", cfg.Daemon.URL)
fmt.Printf("Pool Address: %s\n", cfg.Daemon.Address)
fmt.Printf("Web Interface: %s\n", pool.ExternalWebURL(*webPort))
fmt.Println()

// Connect to Redis
redisClient, err := pool.NewRedisClient(cfg.Redis.URL, cfg.Redis.Password, cfg.Redis.DB)
if err != nil {
fmt.Printf("Failed to connect to Redis: %v\n", err)
os.Exit(1)
}
defer redisClient.Close()

// Start stratum server
server, err := pool.NewStratumServer(cfg, redisClient)
if err != nil {
fmt.Printf("Failed to create stratum server: %v\n", err)
os.Exit(1)
}

// Start web server
webServer := pool.NewWebServer(cfg, redisClient, server)
go webServer.Start(*webPort)

// Start payment processor
paymentProcessor, err := pool.NewPaymentProcessor(cfg, redisClient)
if err != nil {
fmt.Printf("Failed to create payment processor: %v\n", err)
os.Exit(1)
}
go paymentProcessor.Start()
defer paymentProcessor.Stop()

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
<-sigChan
fmt.Println("\n⚠️  Shutting down pool...")
os.Exit(0)
}()

if err := server.Start(); err != nil {
fmt.Printf("Server error: %v\n", err)
os.Exit(1)
}
}
