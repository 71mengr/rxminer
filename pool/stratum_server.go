package pool

import (
"bufio"
"encoding/binary"
"encoding/hex"
"encoding/json"
"fmt"
"math/big"
"net"
"sync"
"time"

"rxminer/rpc"
)

type StratumServer struct {
config     *PoolConfig
daemon     *rpc.Client
redis      *RedisClient
miners     map[string]*MinerSession
mu         sync.RWMutex
jobCounter uint64
currentJob *StratumJob
}

type StratumJob struct {
ID       string
SealHash string
SeedHash string
Target   string
Height   uint64
Created  time.Time
}

type MinerSession struct {
ID       string
Address  string
IP       string
Conn     net.Conn
LoginAt  time.Time
LastSeen time.Time
Shares   uint64
Accepted uint64
Rejected uint64
Hashrate float64
}

type StratumRequest struct {
ID     uint64          `json:"id"`
Method string          `json:"method"`
Params json.RawMessage `json:"params"`
}

type StratumResponse struct {
ID     uint64      `json:"id"`
Result interface{} `json:"result,omitempty"`
Error  *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
Code    int    `json:"code"`
Message string `json:"message"`
}

func NewStratumServer(cfg *PoolConfig, redis *RedisClient) (*StratumServer, error) {
daemon := rpc.NewClient(cfg.Daemon.URL)

// Set pool address as etherbase
var result bool
daemon.Call(&result, "miner_setEtherbase", cfg.Daemon.Address)

return &StratumServer{
config: cfg,
daemon: daemon,
redis:  redis,
miners: make(map[string]*MinerSession),
}, nil
}

func (s *StratumServer) Start() error {
listener, err := net.Listen("tcp", s.config.Pool.Listen)
if err != nil {
return err
}
defer listener.Close()

fmt.Printf("Stratum server listening on %s\n", s.config.Pool.Listen)

// Start job updater
go s.updateJobs()

// Start stats reporter
go s.reportStats()

for {
conn, err := listener.Accept()
if err != nil {
continue
}

go s.handleConnection(conn)
}
}

func (s *StratumServer) updateJobs() {
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

for range ticker.C {
work, err := s.daemon.GetWork()
if err != nil {
continue
}

if len(work) < 3 {
continue
}

if s.currentJob == nil || work[0] != s.currentJob.SealHash {
s.jobCounter++

// Calculate target from pool difficulty
maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
target := new(big.Int).Div(maxUint256, big.NewInt(s.config.Pool.Difficulty))
targetHex := fmt.Sprintf("%064x", target)

s.currentJob = &StratumJob{
ID:       fmt.Sprintf("%d", s.jobCounter),
SealHash: work[0],
SeedHash: work[1],
Target:   targetHex,
Height:   parseHeight(work[3]),
Created:  time.Now(),
}

s.broadcastJob()
}
}
}

func (s *StratumServer) broadcastJob() {
s.mu.RLock()
defer s.mu.RUnlock()

for _, miner := range s.miners {
s.sendJob(miner)
}
}

func (s *StratumServer) sendJob(miner *MinerSession) {
params := []interface{}{
s.currentJob.ID,
s.currentJob.SealHash,
s.currentJob.SeedHash,
s.currentJob.Target,
}

resp := StratumResponse{
ID: 0,
Result: map[string]interface{}{
"method": "job",
"params": params,
},
}

data, _ := json.Marshal(resp)
miner.Conn.Write(append(data, '\n'))
}

func (s *StratumServer) handleConnection(conn net.Conn) {
defer conn.Close()

scanner := bufio.NewScanner(conn)

for scanner.Scan() {
line := scanner.Text()
var req StratumRequest

if err := json.Unmarshal([]byte(line), &req); err != nil {
continue
}

s.handleRequest(conn, &req)
}
}

func (s *StratumServer) handleRequest(conn net.Conn, req *StratumRequest) {
switch req.Method {
case "login":
s.handleLogin(conn, req)
case "submit":
s.handleSubmit(conn, req)
case "keepalive":
s.sendResult(conn, req.ID, "OK")
default:
s.sendError(conn, req.ID, -3, "Method not found")
}
}

func (s *StratumServer) handleLogin(conn net.Conn, req *StratumRequest) {
var params struct {
Login string `json:"login"`
Pass  string `json:"pass"`
}

if err := json.Unmarshal(req.Params, &params); err != nil {
s.sendError(conn, req.ID, -1, "Invalid params")
return
}

miner := &MinerSession{
ID:       params.Login,
Address:  params.Login,
IP:       conn.RemoteAddr().String(),
Conn:     conn,
LoginAt:  time.Now(),
LastSeen: time.Now(),
}

s.mu.Lock()
s.miners[miner.ID] = miner
s.mu.Unlock()

// Send current job if available
if s.currentJob != nil {
s.sendJob(miner)
}

s.sendResult(conn, req.ID, map[string]interface{}{
"id":     miner.ID,
"status": "OK",
})

fmt.Printf("✅ Miner logged in: %s from %s\n", miner.ID[:16], miner.IP)
}

func (s *StratumServer) handleSubmit(conn net.Conn, req *StratumRequest) {
var params []string
if err := json.Unmarshal(req.Params, &params); err != nil {
s.sendError(conn, req.ID, -1, "Invalid params")
return
}

if len(params) < 4 {
s.sendError(conn, req.ID, -1, "Invalid params count")
return
}

minerID := params[0]
jobID := params[1]
nonceHex := params[2]
mixDigestHex := params[3]

// Validate job
if s.currentJob == nil || jobID != s.currentJob.ID {
s.sendError(conn, req.ID, -1, "Invalid job ID")
return
}

// Parse nonce
nonceBytes, _ := hex.DecodeString(nonceHex)
var nonce uint64
if len(nonceBytes) >= 8 {
nonce = binary.BigEndian.Uint64(nonceBytes[:8])
}

// Parse mix digest
mixDigestBytes, _ := hex.DecodeString(mixDigestHex)

// Submit to daemon
nonceHexFull := fmt.Sprintf("0x%016x", nonce)
sealHashHex := s.currentJob.SealHash
mixDigestHexFull := fmt.Sprintf("0x%x", mixDigestBytes)

var submitted bool
err := s.daemon.Call(&submitted, "eth_submitWork", nonceHexFull, sealHashHex, mixDigestHexFull)

// Update miner stats
s.mu.Lock()
if miner, ok := s.miners[minerID]; ok {
miner.LastSeen = time.Now()
miner.Shares++

// Store share in Redis
share := &Share{
Address:    minerID,
Nonce:      nonce,
Difficulty: s.config.Pool.Difficulty,
Hash:       mixDigestHex,
Timestamp:  time.Now(),
}
s.redis.AddShare(share)

if err == nil && submitted {
miner.Accepted++

// Calculate reward (pool fee deducted)
reward := s.calculateReward(s.config.Pool.Difficulty)
s.redis.AddPendingPayout(minerID, reward)

fmt.Printf("�� Share accepted from %s - Reward: %d wei\n", minerID[:16], reward)
} else {
miner.Rejected++
}

// Update hashrate
elapsed := time.Since(miner.LoginAt).Seconds()
if elapsed > 0 {
miner.Hashrate = float64(miner.Accepted) / elapsed * float64(s.config.Pool.Difficulty)
s.redis.UpdateHashrate(minerID, miner.Hashrate)
}

s.redis.UpdateMinerStats(&MinerStats{
Address:   minerID,
Hashrate:  miner.Hashrate,
Shares:    miner.Shares,
Accepted:  miner.Accepted,
Rejected:  miner.Rejected,
LastSeen:  miner.LastSeen,
})
}
s.mu.Unlock()

if err != nil {
s.sendError(conn, req.ID, -1, err.Error())
} else if submitted {
s.sendResult(conn, req.ID, true)
} else {
s.sendError(conn, req.ID, -1, "Invalid share")
}
}

func (s *StratumServer) calculateReward(difficulty int64) *big.Int {
// Base reward per share
baseReward := new(big.Int).Mul(big.NewInt(difficulty), big.NewInt(1e12))

// Apply pool fee
fee := new(big.Int).Mul(baseReward, big.NewInt(int64(s.config.Pool.Fee)))
fee.Div(fee, big.NewInt(100))

reward := new(big.Int).Sub(baseReward, fee)
return reward
}

func (s *StratumServer) sendResult(conn net.Conn, id uint64, result interface{}) {
resp := StratumResponse{
ID:     id,
Result: result,
}
data, _ := json.Marshal(resp)
conn.Write(append(data, '\n'))
}

func (s *StratumServer) sendError(conn net.Conn, id uint64, code int, message string) {
resp := StratumResponse{
ID:    id,
Error: &RPCError{Code: code, Message: message},
}
data, _ := json.Marshal(resp)
conn.Write(append(data, '\n'))
}

func (s *StratumServer) reportStats() {
ticker := time.NewTicker(60 * time.Second)
defer ticker.Stop()

for range ticker.C {
s.mu.RLock()
totalMiners := len(s.miners)
totalHashrate := 0.0
totalShares := uint64(0)

for _, miner := range s.miners {
totalHashrate += miner.Hashrate
totalShares += miner.Accepted
}
s.mu.RUnlock()

fmt.Printf("�� Pool Stats - Miners: %d, Total Hashrate: %.2f H/s, Total Shares: %d\n",
totalMiners, totalHashrate, totalShares)
}
}

func parseHeight(heightStr string) uint64 {
var height uint64
fmt.Sscanf(heightStr, "0x%x", &height)
return height
}
