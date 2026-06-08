package pool

import (
"encoding/json"
"fmt"
"html/template"
"net/http"
"sort"
//"strconv"
"time"
)

type WebServer struct {
config *PoolConfig
redis  *RedisClient
server *StratumServer
}

type PoolStats struct {
Name              string    `json:"name"`
Hashrate          float64   `json:"hashrate"`
Miners            int       `json:"miners"`
TotalShares       uint64    `json:"totalShares"`
BlocksFound       int       `json:"blocksFound"`
LastBlock         uint64    `json:"lastBlock"`
PoolFee           int       `json:"poolFee"`
Difficulty        int64     `json:"difficulty"`
NetworkDifficulty string    `json:"networkDifficulty"`
Uptime            string    `json:"uptime"`
StartTime         time.Time `json:"startTime"`
}

type MinerInfo struct {
Address    string    `json:"address"`
Hashrate   float64   `json:"hashrate"`
Shares     uint64    `json:"shares"`
Accepted   uint64    `json:"accepted"`
Rejected   uint64    `json:"rejected"`
LastSeen   time.Time `json:"lastSeen"`
Unpaid     string    `json:"unpaid"`
TotalPaid  string    `json:"totalPaid"`
}

type BlockInfo struct {
Height   uint64    `json:"height"`
Hash     string    `json:"hash"`
Miner    string    `json:"miner"`
Time     time.Time `json:"time"`
Reward   string    `json:"reward"`
}

func NewWebServer(cfg *PoolConfig, redis *RedisClient, server *StratumServer) *WebServer {
return &WebServer{
config: cfg,
redis:  redis,
server: server,
}
}

func (w *WebServer) Start(port int) error {
http.HandleFunc("/", w.handleIndex)
http.HandleFunc("/api/stats", w.handleAPIStats)
http.HandleFunc("/api/miners", w.handleAPIMiners)
http.HandleFunc("/api/miner/", w.handleAPIMiner)
http.HandleFunc("/api/blocks", w.handleAPIBlocks)
http.HandleFunc("/static/", w.handleStatic)

addr := fmt.Sprintf(":%d", port)
fmt.Printf("Web server listening on %s\n", ExternalWebURL(port))

return http.ListenAndServe(addr, nil)
}

func (w *WebServer) handleIndex(wr http.ResponseWriter, r *http.Request) {
if r.URL.Path != "/" {
http.NotFound(wr, r)
return
}

tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Name}} - Mining Pool</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #eee;
            min-height: 100vh;
        }
        .header {
            background: rgba(0,0,0,0.3);
            padding: 20px;
            text-align: center;
            border-bottom: 2px solid #00d4ff;
        }
        .header h1 { font-size: 2.5em; margin-bottom: 5px; }
        .header p { color: #00d4ff; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: rgba(255,255,255,0.1);
            border-radius: 10px;
            padding: 20px;
            text-align: center;
            backdrop-filter: blur(10px);
            transition: transform 0.3s;
        }
        .stat-card:hover { transform: translateY(-5px); }
        .stat-value { font-size: 2em; font-weight: bold; color: #00d4ff; }
        .stat-label { font-size: 0.9em; opacity: 0.8; margin-top: 5px; }
        .section {
            background: rgba(255,255,255,0.05);
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .section h2 {
            margin-bottom: 15px;
            color: #00d4ff;
            border-left: 3px solid #00d4ff;
            padding-left: 15px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        th { background: rgba(0,212,255,0.2); font-weight: bold; }
        tr:hover { background: rgba(255,255,255,0.05); }
        .miner-address {
            font-family: monospace;
            font-size: 0.85em;
        }
        .status-online { color: #00ff88; }
        .status-offline { color: #ff4444; }
        .footer {
            text-align: center;
            padding: 20px;
            background: rgba(0,0,0,0.3);
            margin-top: 40px;
        }
        .connection-box {
            background: rgba(0,0,0,0.5);
            border-radius: 10px;
            padding: 15px;
            text-align: center;
            margin-top: 20px;
        }
        .connection-url {
            font-family: monospace;
            font-size: 1.2em;
            background: #000;
            padding: 10px;
            border-radius: 5px;
            display: inline-block;
            margin-top: 10px;
        }
        .copy-btn {
            background: #00d4ff;
            color: #1a1a2e;
            border: none;
            padding: 5px 15px;
            border-radius: 5px;
            cursor: pointer;
            margin-left: 10px;
        }
        .copy-btn:hover { background: #00b8e6; }
        .refresh-btn {
            float: right;
            background: #00d4ff;
            color: #1a1a2e;
            border: none;
            padding: 5px 15px;
            border-radius: 5px;
            cursor: pointer;
        }
        @media (max-width: 768px) {
            .stats-grid { grid-template-columns: 1fr; }
            th, td { padding: 8px; font-size: 12px; }
        }
    </style>
    <script>
        function copyToClipboard() {
            const url = document.getElementById('pool-url').innerText;
            navigator.clipboard.writeText(url);
            alert('Copied: ' + url);
        }
        
        function refreshData() {
            fetch('/api/stats')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('stat-hashrate').innerText = data.hashrate.toFixed(2) + ' H/s';
                    document.getElementById('stat-miners').innerText = data.miners;
                    document.getElementById('stat-shares').innerText = data.totalShares.toLocaleString();
                    document.getElementById('stat-fee').innerText = data.poolFee + '%';
                    document.getElementById('stat-difficulty').innerText = data.difficulty;
                });
            
            fetch('/api/miners')
                .then(r => r.json())
                .then(data => {
                    const tbody = document.getElementById('miners-table');
                    tbody.innerHTML = '';
                    data.forEach(miner => {
                        const row = tbody.insertRow();
                        row.insertCell(0).innerHTML = '<span class="miner-address">' + miner.address.substring(0, 16) + '...</span>';
                        row.insertCell(1).innerHTML = miner.hashrate.toFixed(2) + ' H/s';
                        row.insertCell(2).innerHTML = miner.shares;
                        row.insertCell(3).innerHTML = miner.accepted;
                        row.insertCell(4).innerHTML = miner.rejected;
                        row.insertCell(5).innerHTML = '<span class="status-online">Online</span>';
                        row.insertCell(6).innerHTML = new Date(miner.lastSeen).toLocaleString();
                    });
                });
        }
        
        setInterval(refreshData, 10000);
    </script>
</head>
<body>
    <div class="header">
        <h1>⛏️ {{.Name}}</h1>
        <p>RandomX Mining Pool</p>
    </div>
    
    <div class="container">
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value" id="stat-hashrate">0 H/s</div>
                <div class="stat-label">Pool Hashrate</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-miners">0</div>
                <div class="stat-label">Active Miners</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-shares">0</div>
                <div class="stat-label">Total Shares</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-fee">0%</div>
                <div class="stat-label">Pool Fee</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-difficulty">0</div>
                <div class="stat-label">Share Difficulty</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-blocks">0</div>
                <div class="stat-label">Blocks Found</div>
            </div>
        </div>
        
        <div class="connection-box">
            <h3>�� Connect Your Miner</h3>
            <p>Use the following settings in your XMRig config:</p>
            <div class="connection-url">
                <span id="pool-url">stratum+tcp://{{.ServerAddress}}:{{.ServerPort}}</span>
                <button class="copy-btn" onclick="copyToClipboard()">Copy</button>
            </div>
            <p style="margin-top: 10px;">
                <strong>Username:</strong> Your wallet address<br>
                <strong>Password:</strong> x
            </p>
        </div>
        
        <div class="section">
            <h2>�� Active Miners</h2>
            <div style="overflow-x: auto;">
                <table>
                    <thead>
                        <tr><th>Address</th><th>Hashrate</th><th>Shares</th><th>Accepted</th><th>Rejected</th><th>Status</th><th>Last Seen</th></tr>
                    </thead>
                    <tbody id="miners-table">
                        <tr><td colspan="7" style="text-align: center;">Loading miners...</td></tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
    
    <div class="footer">
        <p>&copy; 2026 {{.Name}} - Powered by RandomX</p>
    </div>
    
    <script>refreshData();</script>
</body>
</html>`

t := template.Must(template.New("index").Parse(tmpl))

stats := w.getPoolStats()

// Parse listen address to get port
port := w.config.Pool.Listen
if port[0] == ':' {
port = port[1:]
}

data := map[string]interface{}{
"Name":          w.config.Pool.Name,
"ServerAddress": "stratum.tkmchain.site",
"ServerPort":    port,
"Stats":         stats,
}

wr.Header().Set("Content-Type", "text/html")
t.Execute(wr, data)
}

func (w *WebServer) handleAPIStats(wr http.ResponseWriter, r *http.Request) {
stats := w.getPoolStats()
json.NewEncoder(wr).Encode(stats)
}

func (w *WebServer) handleAPIMiners(wr http.ResponseWriter, r *http.Request) {
w.server.mu.RLock()
defer w.server.mu.RUnlock()

var miners []MinerInfo
for _, m := range w.server.miners {
miners = append(miners, MinerInfo{
Address:   m.Address,
Hashrate:  m.Hashrate,
Shares:    m.Shares,
Accepted:  m.Accepted,
Rejected:  m.Rejected,
LastSeen:  m.LastSeen,
})
}

// Sort by hashrate descending
sort.Slice(miners, func(i, j int) bool {
return miners[i].Hashrate > miners[j].Hashrate
})

json.NewEncoder(wr).Encode(miners)
}

func (w *WebServer) handleAPIMiner(wr http.ResponseWriter, r *http.Request) {
address := r.URL.Path[len("/api/miner/"):]
if address == "" {
http.Error(wr, "Address required", 400)
return
}

stats, err := w.redis.GetMinerStats(address)
if err != nil {
json.NewEncoder(wr).Encode(map[string]interface{}{
"address": address,
"error":   "Miner not found",
})
return
}

if stats == nil {
json.NewEncoder(wr).Encode(map[string]interface{}{
"address": address,
"error":   "Miner not found",
})
return
}

json.NewEncoder(wr).Encode(stats)
}

func (w *WebServer) handleAPIBlocks(wr http.ResponseWriter, r *http.Request) {
blocks, err := w.redis.GetBlocks()
if err != nil {
json.NewEncoder(wr).Encode([]interface{}{})
return
}
json.NewEncoder(wr).Encode(blocks)
}

func (w *WebServer) handleStatic(wr http.ResponseWriter, r *http.Request) {
http.ServeFile(wr, r, "static"+r.URL.Path)
}

func (w *WebServer) getPoolStats() PoolStats {
w.server.mu.RLock()
defer w.server.mu.RUnlock()

totalHashrate := 0.0
totalShares := uint64(0)

for _, miner := range w.server.miners {
totalHashrate += miner.Hashrate
totalShares += miner.Accepted
}

blocks, _ := w.redis.GetBlocks()

return PoolStats{
Name:       w.config.Pool.Name,
Hashrate:   totalHashrate,
Miners:     len(w.server.miners),
TotalShares: totalShares,
BlocksFound: len(blocks),
PoolFee:    w.config.Pool.Fee,
Difficulty: w.config.Pool.Difficulty,
StartTime:  time.Now(),
}
}
