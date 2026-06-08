package pool

import (
"context"
"encoding/json"
"fmt"
"time"

"github.com/redis/go-redis/v9"
)

type RedisClient struct {
client *redis.Client
ctx    context.Context
}

type MinerStats struct {
Address    string    `json:"address"`
Hashrate   float64   `json:"hashrate"`
Shares     uint64    `json:"shares"`
Accepted   uint64    `json:"accepted"`
Rejected   uint64    `json:"rejected"`
LastSeen   time.Time `json:"lastSeen"`
TotalPaid  *big.Int  `json:"totalPaid"`
Unpaid     *big.Int  `json:"unpaid"`
}

type Share struct {
Address    string    `json:"address"`
Nonce      uint64    `json:"nonce"`
Difficulty int64     `json:"difficulty"`
Hash       string    `json:"hash"`
Timestamp  time.Time `json:"timestamp"`
}

func NewRedisClient(url, password string, db int) (*RedisClient, error) {
opt, err := redis.ParseURL(url)
if err != nil {
// Try localhost if URL parsing fails
opt = &redis.Options{
Addr:     "localhost:6379",
Password: password,
DB:       db,
}
}

client := redis.NewClient(opt)
ctx := context.Background()

// Test connection
if err := client.Ping(ctx).Err(); err != nil {
return nil, fmt.Errorf("failed to connect to Redis: %v", err)
}

fmt.Println("✅ Connected to Redis")

return &RedisClient{
client: client,
ctx:    ctx,
}, nil
}

func (r *RedisClient) AddShare(share *Share) error {
// Store share in sorted set for block calculation
key := fmt.Sprintf("shares:%s", time.Now().Format("2006-01-02"))

data, err := json.Marshal(share)
if err != nil {
return err
}

return r.client.ZAdd(r.ctx, key, redis.Z{
Score:  float64(share.Timestamp.Unix()),
Member: data,
}).Err()
}

func (r *RedisClient) GetShares(address string, since time.Time) ([]Share, error) {
// Get shares from last 24 hours
key := fmt.Sprintf("shares:%s", time.Now().Format("2006-01-02"))

zrange := r.client.ZRangeByScore(r.ctx, key, &redis.ZRangeBy{
Min: fmt.Sprintf("%d", since.Unix()),
Max: fmt.Sprintf("%d", time.Now().Unix()),
})

values, err := zrange.Result()
if err != nil {
return nil, err
}

var shares []Share
for _, v := range values {
var share Share
if err := json.Unmarshal([]byte(v), &share); err == nil && share.Address == address {
shares = append(shares, share)
}
}

return shares, nil
}

func (r *RedisClient) UpdateMinerStats(stats *MinerStats) error {
key := fmt.Sprintf("miner:%s", stats.Address)

data, err := json.Marshal(stats)
if err != nil {
return err
}

return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

func (r *RedisClient) GetMinerStats(address string) (*MinerStats, error) {
key := fmt.Sprintf("miner:%s", address)

data, err := r.client.Get(r.ctx, key).Result()
if err == redis.Nil {
return nil, nil
}
if err != nil {
return nil, err
}

var stats MinerStats
if err := json.Unmarshal([]byte(data), &stats); err != nil {
return nil, err
}

return &stats, nil
}

func (r *RedisClient) AddPendingPayout(address string, amount *big.Int) error {
key := "pending_payouts"

return r.client.HSet(r.ctx, key, address, amount.String()).Err()
}

func (r *RedisClient) GetPendingPayouts() (map[string]*big.Int, error) {
key := "pending_payouts"

results, err := r.client.HGetAll(r.ctx, key).Result()
if err != nil {
return nil, err
}

payouts := make(map[string]*big.Int)
for addr, amountStr := range results {
amount := new(big.Int)
amount.SetString(amountStr, 10)
payouts[addr] = amount
}

return payouts, nil
}

func (r *RedisClient) ClearPendingPayout(address string) error {
key := "pending_payouts"
return r.client.HDel(r.ctx, key, address).Err()
}

func (r *RedisClient) IncrementShareCount(address string) error {
key := fmt.Sprintf("miner:%s:shares", address)
return r.client.Incr(r.ctx, key).Err()
}

func (r *RedisClient) GetShareCount(address string) (int64, error) {
key := fmt.Sprintf("miner:%s:shares", address)
return r.client.Get(r.ctx, key).Int64()
}

func (r *RedisClient) UpdateHashrate(address string, hashrate float64) error {
key := fmt.Sprintf("miner:%s:hashrate", address)
return r.client.Set(r.ctx, key, hashrate, 2*time.Minute).Err()
}

func (r *RedisClient) GetHashrate(address string) (float64, error) {
key := fmt.Sprintf("miner:%s:hashrate", address)
return r.client.Get(r.ctx, key).Float64()
}

func (r *RedisClient) AddBlock(height uint64, hash string, miner string) error {
key := "blocks:found"

block := map[string]interface{}{
"height": height,
"hash":   hash,
"miner":  miner,
"time":   time.Now().Unix(),
}

data, _ := json.Marshal(block)
return r.client.LPush(r.ctx, key, data).Err()
}

func (r *RedisClient) GetBlocks() ([]map[string]interface{}, error) {
key := "blocks:found"

results, err := r.client.LRange(r.ctx, key, 0, 99).Result()
if err != nil {
return nil, err
}

var blocks []map[string]interface{}
for _, v := range results {
var block map[string]interface{}
if err := json.Unmarshal([]byte(v), &block); err == nil {
blocks = append(blocks, block)
}
}

return blocks, nil
}

func (r *RedisClient) Close() error {
return r.client.Close()
}
