package miner

import (
"encoding/binary"
"encoding/hex"
"fmt"
"math/big"
"runtime"
"sync/atomic"
"time"

"rxminer/pool"
"rxminer/randomx"
)

type PoolMiner struct {
pool     *pool.StratumClient
address  string
threads  int
stop     int32
hashes   uint64
accepted uint64
rejected uint64
boost    bool
}

func NewPoolMiner(poolURL, address, password string, threads int, boost bool) (*PoolMiner, error) {
if threads <= 0 {
threads = runtime.NumCPU()
}

// Connect to pool
client, err := pool.NewStratumClient(poolURL, address, password)
if err != nil {
return nil, fmt.Errorf("failed to connect to pool: %v", err)
}

fmt.Printf("✅ Connected to pool: %s\n", poolURL)
fmt.Printf("✅ Mining address: %s\n", address)
fmt.Println()

return &PoolMiner{
pool:     client,
address:  address,
threads:  threads,
boost:    boost,
}, nil
}

func (m *PoolMiner) GetWork() (sealHash, seedHash []byte, target *big.Int, height uint64, err error) {
sealHashHex, seedHashHex, targetHex, height, err := m.pool.GetWork()
if err != nil {
return nil, nil, nil, 0, err
}

sealHash = hexToBytes(sealHashHex)
seedHash = hexToBytes(seedHashHex)
target = hexToBig(targetHex)

return sealHash, seedHash, target, height, nil
}

func (m *PoolMiner) SubmitWork(nonce uint64, sealHash, mixDigest []byte) error {
return m.pool.SubmitWork(nonce, mixDigest)
}

func (m *PoolMiner) Start() {
fmt.Printf("Starting RandomX miner with %d threads\n", m.threads)
if m.boost {
fmt.Println("⚡ BOOST MODE ENABLED")
}
fmt.Println()

runtime.GOMAXPROCS(m.threads)

sealHash, seedHash, target, height, err := m.GetWork()
if err != nil {
fmt.Printf("❌ Failed to get initial work: %v\n", err)
return
}

fmt.Println("�� Connected to pool")
fmt.Printf("   Seal hash: %x...\n", sealHash[:8])
fmt.Printf("   Seed hash: %x...\n", seedHash[:8])
fmt.Printf("   Target: %s\n", target.String())
fmt.Printf("   Height: %d\n", height)
fmt.Println()
fmt.Println("⛏️  Mining... Press Ctrl+C to stop")
fmt.Println()

work := &PoolWork{
SealHash: sealHash,
SeedHash: seedHash,
Target:   target,
Height:   height,
}

for i := 0; i < m.threads; i++ {
go m.mineThread(i, work)
}

go m.monitorHashrate()

select {}
}

func (m *PoolMiner) mineThread(threadID int, initialWork *PoolWork) {
flags := randomx.RANDOMX_FLAG_JIT | randomx.RANDOMX_FLAG_HARD_AES
if m.boost {
flags |= randomx.RANDOMX_FLAG_LARGE_PAGES
}

cache := randomx.NewCache(flags)
if cache == nil {
fmt.Printf("❌ Thread %d: Failed to create cache\n", threadID)
return
}
defer cache.Close()

cache.Init(initialWork.SeedHash)

dataset := randomx.NewDataset(flags)
if dataset != nil {
dataset.Init(cache, 0, 0)
}
defer func() {
if dataset != nil {
dataset.Close()
}
}()

vm := randomx.NewVM(flags, cache, dataset)
if vm == nil {
fmt.Printf("❌ Thread %d: Failed to create VM\n", threadID)
return
}
defer vm.Close()

work := initialWork
nonce := uint64(threadID)
hashes := uint64(0)
startTime := time.Now()

input := make([]byte, 40)
output := make([]byte, 32)

for atomic.LoadInt32(&m.stop) == 0 {
if hashes%10000 == 0 {
newSealHash, newSeedHash, newTarget, newHeight, err := m.GetWork()
if err == nil && !bytesEqual(newSealHash, work.SealHash) {
work = &PoolWork{
SealHash: newSealHash,
SeedHash: newSeedHash,
Target:   newTarget,
Height:   newHeight,
}
cache.Init(work.SeedHash)
if dataset != nil {
dataset.Init(cache, 0, 0)
}
fmt.Printf("�� Thread %d: New job - height %d\n", threadID, work.Height)
}
}

copy(input[:32], work.SealHash)
binary.BigEndian.PutUint64(input[32:], nonce)

vm.CalculateHash(input, output)

hashes++
atomic.AddUint64(&m.hashes, 1)

result := new(big.Int).SetBytes(output)
if result.Cmp(work.Target) <= 0 {
mixDigest := make([]byte, 32)
copy(mixDigest, output)
elapsed := time.Since(startTime)
hashrate := float64(hashes) / elapsed.Seconds()

fmt.Printf("\n�� Thread %d: SHARE FOUND!\n", threadID)
fmt.Printf("   Nonce: %d (0x%x)\n", nonce, nonce)
fmt.Printf("   Mix digest: %x...\n", mixDigest[:8])
fmt.Printf("   Result: %s\n", result.String())
fmt.Printf("   Target: %s\n", work.Target.String())
fmt.Printf("   Attempts: %d\n", hashes)
fmt.Printf("   Hashrate: %.2f H/s\n", hashrate)

if err := m.SubmitWork(nonce, work.SealHash, mixDigest); err != nil {
fmt.Printf("   ❌ Submission error: %v\n", err)
atomic.AddUint64(&m.rejected, 1)
} else {
fmt.Printf("   ✅ ACCEPTED! Share accepted by pool\n")
atomic.AddUint64(&m.accepted, 1)
}
fmt.Println()
}

nonce++
}
}

func (m *PoolMiner) monitorHashrate() {
ticker := time.NewTicker(10 * time.Second)
defer ticker.Stop()

var lastHashes uint64
lastTime := time.Now()

for range ticker.C {
if atomic.LoadInt32(&m.stop) == 1 {
return
}

currentHashes := atomic.LoadUint64(&m.hashes)
currentTime := time.Now()
elapsed := currentTime.Sub(lastTime).Seconds()
hashrate := float64(currentHashes-lastHashes) / elapsed

fmt.Printf("[�� Stats] Hashrate: %.2f H/s | Accepted: %d | Rejected: %d | Address: %s\n",
hashrate,
atomic.LoadUint64(&m.accepted),
atomic.LoadUint64(&m.rejected),
m.address[:16]+"...")

lastHashes = currentHashes
lastTime = currentTime
}
}

func (m *PoolMiner) Stop() {
atomic.StoreInt32(&m.stop, 1)
if m.pool != nil {
m.pool.Close()
}
}

func hexToBytes(hexStr string) []byte {
if len(hexStr) >= 2 && hexStr[:2] == "0x" {
hexStr = hexStr[2:]
}
bytes, _ := hex.DecodeString(hexStr)
return bytes
}

func hexToBig(hexStr string) *big.Int {
if len(hexStr) >= 2 && hexStr[:2] == "0x" {
hexStr = hexStr[2:]
}
bigInt := new(big.Int)
bigInt.SetString(hexStr, 16)
return bigInt
}


type PoolWork struct {
SealHash []byte
SeedHash []byte
Target   *big.Int
Height   uint64
}
