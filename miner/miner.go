package miner

import (
"encoding/binary"
"fmt"
"math/big"
"runtime"
"sync/atomic"
"time"

"rxminer/randomx"
"rxminer/rpc"
)

type Work struct {
SealHash []byte
SeedHash []byte
Target   *big.Int
Height   uint64
}

type Miner struct {
client   *rpc.Client
address  string
threads  int
stop     int32
hashes   uint64
accepted uint64
rejected uint64
boost    bool
}

func NewMiner(rpcURL string, address string, threads int, boost bool) (*Miner, error) {
if threads <= 0 {
threads = runtime.NumCPU()
}

if len(address) != 42 || address[:2] != "0x" {
return nil, fmt.Errorf("invalid address format: %s", address)
}

client := rpc.NewClient(rpcURL)

fmt.Printf("✅ Address validated: %s\n", address)
fmt.Println()

return &Miner{
client:   client,
address:  address,
threads:  threads,
boost:    boost,
}, nil
}

func (m *Miner) GetWork() (*Work, error) {
work, err := m.client.GetWork()
if err != nil {
return nil, err
}

if len(work) < 3 {
return nil, fmt.Errorf("invalid work response: %+v", work)
}

sealHash := rpc.HexToHash(work[0])
seedHash := rpc.HexToHash(work[1])
target := rpc.HexToBig(work[2])

var height uint64
if len(work) > 3 {
fmt.Sscanf(work[3], "0x%x", &height)
}

fmt.Printf("�� GetWork: height=%d, sealHash=%x..., target=%s\n", 
height, sealHash[:8], target.String())

return &Work{
SealHash: sealHash,
SeedHash: seedHash,
Target:   target,
Height:   height,
}, nil
}

func (m *Miner) SubmitWork(nonce uint64, sealHash, mixDigest []byte) (bool, error) {
nonceHex := fmt.Sprintf("0x%016x", nonce)
sealHashHex := fmt.Sprintf("0x%x", sealHash)
mixDigestHex := fmt.Sprintf("0x%x", mixDigest)

fmt.Printf("�� Submit: nonce=%s, sealHash=%x..., mix=%x...\n", 
nonceHex, sealHash[:8], mixDigest[:8])

return m.client.SubmitWork(nonceHex, sealHashHex, mixDigestHex)
}

func (m *Miner) Start() {
fmt.Printf("Starting RandomX miner with %d threads\n", m.threads)
if m.boost {
fmt.Println("⚡ BOOST MODE ENABLED")
}
fmt.Printf("RPC: %s\n", m.client)
fmt.Printf("Address: %s\n", m.address)
fmt.Println()

runtime.GOMAXPROCS(m.threads)

work, err := m.GetWork()
if err != nil {
fmt.Printf("❌ Failed to get initial work: %v\n", err)
return
}

fmt.Println("�� Connected to daemon")
fmt.Printf("   Seal hash: %x...\n", work.SealHash[:8])
fmt.Printf("   Seed hash: %x...\n", work.SeedHash[:8])
fmt.Printf("   Target: %s\n", work.Target.String())
fmt.Printf("   Height: %d\n", work.Height)
fmt.Println()
fmt.Println("⛏️  Mining... Press Ctrl+C to stop")
fmt.Println()

for i := 0; i < m.threads; i++ {
go m.mineThread(i, work)
}

go m.monitorHashrate()

select {}
}

func (m *Miner) mineThread(threadID int, initialWork *Work) {
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
fmt.Printf("✅ Thread %d: Cache ready\n", threadID)

dataset := randomx.NewDataset(flags)
if dataset != nil {
dataset.Init(cache, 0, 0)
fmt.Printf("✅ Thread %d: Dataset ready (fast mode)\n", threadID)
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

fmt.Printf("�� Thread %d: VM ready\n", threadID)

work := initialWork
nonce := uint64(threadID)
hashes := uint64(0)
startTime := time.Now()

input := make([]byte, 40)
output := make([]byte, 32)

for atomic.LoadInt32(&m.stop) == 0 {
if hashes%10000 == 0 {
newWork, err := m.GetWork()
if err == nil && !bytesEqual(newWork.SealHash, work.SealHash) {
work = newWork
cache.Init(work.SeedHash)
if dataset != nil {
dataset.Init(cache, 0, 0)
}
fmt.Printf("�� Thread %d: New work - height %d, sealHash=%x...\n", 
threadID, work.Height, work.SealHash[:8])
}
}

// Input = SealHash (32 bytes) + Nonce (8 bytes)
// CRITICAL: The nonce must be in BIG-ENDIAN order (not little-endian)
// because the daemon's header.Nonce stores it as big-endian
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

fmt.Printf("\n�� Thread %d: SOLUTION FOUND!\n", threadID)
fmt.Printf("   Nonce: %d (0x%x)\n", nonce, nonce)
fmt.Printf("   Seal hash: %x...\n", work.SealHash[:8])
fmt.Printf("   Computed mix: %x...\n", mixDigest[:8])
fmt.Printf("   Result: %s\n", result.String())
fmt.Printf("   Target: %s\n", work.Target.String())
fmt.Printf("   Attempts: %d\n", hashes)
fmt.Printf("   Hashrate: %.2f H/s\n", hashrate)

submitted, err := m.SubmitWork(nonce, work.SealHash, mixDigest)
if err != nil {
fmt.Printf("   ❌ Submission error: %v\n", err)
atomic.AddUint64(&m.rejected, 1)
} else if submitted {
fmt.Printf("   ✅✅✅ ACCEPTED! Block reward to: %s\n", m.address)
atomic.AddUint64(&m.accepted, 1)
} else {
fmt.Printf("   ❌ REJECTED!\n")
atomic.AddUint64(&m.rejected, 1)
}
fmt.Println()
}

nonce++
}
}

func (m *Miner) monitorHashrate() {
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

func (m *Miner) Stop() {
atomic.StoreInt32(&m.stop, 1)
}

func bytesEqual(a, b []byte) bool {
if len(a) != len(b) {
return false
}
for i := range a {
if a[i] != b[i] {
return false
}
}
return true
}
