package pool

import (
"fmt"
"math/big"
"time"

"rxminer/rpc"
)

type PaymentProcessor struct {
config   *PoolConfig
redis    *RedisClient
daemon   *rpc.Client
stopCh   chan struct{}
}

func NewPaymentProcessor(cfg *PoolConfig, redis *RedisClient) (*PaymentProcessor, error) {
daemon := rpc.NewClient(cfg.Daemon.URL)

return &PaymentProcessor{
config: cfg,
redis:  redis,
daemon: daemon,
stopCh: make(chan struct{}),
}, nil
}

func (p *PaymentProcessor) Start() {
if !p.config.Payments.Enabled {
fmt.Println("Payments disabled")
return
}

fmt.Printf("Payment processor started - Interval: %ds, Min payout: %d wei\n", 
p.config.Payments.Interval, p.config.Payments.MinPayout)

ticker := time.NewTicker(time.Duration(p.config.Payments.Interval) * time.Second)
defer ticker.Stop()

for {
select {
case <-ticker.C:
p.processPayments()
case <-p.stopCh:
return
}
}
}

func (p *PaymentProcessor) processPayments() {
fmt.Println("Processing payments...")

payouts, err := p.redis.GetPendingPayouts()
if err != nil {
fmt.Printf("Failed to get pending payouts: %v\n", err)
return
}

minPayout := big.NewInt(p.config.Payments.MinPayout)

for address, amount := range payouts {
if amount.Cmp(minPayout) >= 0 {
if err := p.sendPayment(address, amount); err != nil {
fmt.Printf("Failed to send payment to %s: %v\n", address, err)
} else {
p.redis.ClearPendingPayout(address)
fmt.Printf("✅ Payment sent to %s: %s wei\n", address[:16], amount.String())
}
}
}
}

func (p *PaymentProcessor) sendPayment(to string, amount *big.Int) error {
// Get nonce
var nonce uint64
if err := p.daemon.Call(&nonce, "eth_getTransactionCount", p.config.Daemon.Address, "pending"); err != nil {
return err
}

// Get gas price
var gasPrice *big.Int
if p.config.Payments.GasPrice > 0 {
gasPrice = big.NewInt(p.config.Payments.GasPrice)
} else {
var gasPriceHex string
if err := p.daemon.Call(&gasPriceHex, "eth_gasPrice"); err != nil {
return err
}
gasPrice = hexToBig(gasPriceHex)
}

// Build transaction
tx := map[string]interface{}{
"from":     p.config.Daemon.Address,
"to":       to,
"value":    fmt.Sprintf("0x%x", amount),
"gas":      "0x5208", // 21000 gas
"gasPrice": fmt.Sprintf("0x%x", gasPrice),
"nonce":    fmt.Sprintf("0x%x", nonce),
}

var txHash string
err := p.daemon.Call(&txHash, "eth_sendTransaction", tx)
return err
}

func (p *PaymentProcessor) Stop() {
close(p.stopCh)
}

func hexToBig(hexStr string) *big.Int {
if len(hexStr) >= 2 && hexStr[:2] == "0x" {
hexStr = hexStr[2:]
}
bigInt := new(big.Int)
bigInt.SetString(hexStr, 16)
return bigInt
}
