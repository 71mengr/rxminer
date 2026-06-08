package pool

import (
"encoding/json"
"os"
)

type PoolConfig struct {
Pool struct {
Listen     string `json:"listen"`
Difficulty int64  `json:"difficulty"`
MaxConn    int    `json:"maxConn"`
Timeout    string `json:"timeout"`
Name       string `json:"name"`
Fee        int    `json:"fee"` // Pool fee percentage (e.g., 1 = 1%)
} `json:"pool"`

Daemon struct {
URL      string `json:"url"`
Address  string `json:"address"` // Pool wallet address
} `json:"daemon"`

Redis struct {
URL      string `json:"url"`
Password string `json:"password"`
DB       int    `json:"db"`
} `json:"redis"`

Payments struct {
Enabled     bool   `json:"enabled"`
Interval    int    `json:"interval"`     // Payment interval in seconds
MinPayout   int64  `json:"minPayout"`    // Minimum payout in wei
GasPrice    int64  `json:"gasPrice"`     // Gas price for payments
} `json:"payments"`

RandomX struct {
Enabled     bool   `json:"enabled"`
EpochLength uint64 `json:"epochLength"`
} `json:"randomx"`
}

func LoadPoolConfig(path string) (*PoolConfig, error) {
data, err := os.ReadFile(path)
if err != nil {
return nil, err
}

var cfg PoolConfig
if err := json.Unmarshal(data, &cfg); err != nil {
return nil, err
}

return &cfg, nil
}
