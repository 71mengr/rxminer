package pool

import (
"bufio"
"encoding/hex"
"encoding/json"
"fmt"
"net"
"net/url"
"strings"
"sync"
"time"
)

type StratumClient struct {
conn     net.Conn
url      string
address  string
password string
mu       sync.Mutex
jobID    string
blob     string
target   string
height   uint64
}

type StratumMessage struct {
ID     uint64          `json:"id"`
Method string          `json:"method"`
Params json.RawMessage `json:"params"`
}

func NewStratumClient(rawURL, address, password string) (*StratumClient, error) {
poolAddress, err := normalizePoolAddress(rawURL)
if err != nil {
return nil, err
}

conn, err := net.Dial("tcp", poolAddress)
if err != nil {
return nil, fmt.Errorf("failed to connect to pool: %v", err)
}

client := &StratumClient{
conn:     conn,
url:      rawURL,
address:  address,
password: password,
}

// Login to pool
if err := client.login(); err != nil {
conn.Close()
return nil, err
}

// Start listening for jobs
go client.listen()

return client, nil
}

func normalizePoolAddress(rawURL string) (string, error) {
if !strings.Contains(rawURL, "://") {
return rawURL, nil
}

parsed, err := url.Parse(rawURL)
if err != nil {
return "", fmt.Errorf("invalid pool URL: %v", err)
}

if parsed.Host == "" {
return "", fmt.Errorf("invalid pool URL: missing host")
}

return parsed.Host, nil
}

func (c *StratumClient) login() error {
params := map[string]interface{}{
"login": c.address,
"pass":  c.password,
"agent": "rxminer/1.0",
}

req := StratumMessage{
ID:     1,
Method: "login",
Params: mustMarshal(params),
}

if err := c.send(req); err != nil {
return err
}

time.Sleep(500 * time.Millisecond)
return nil
}

func (c *StratumClient) listen() {
scanner := bufio.NewScanner(c.conn)
for scanner.Scan() {
line := scanner.Text()
var msg StratumMessage
if err := json.Unmarshal([]byte(line), &msg); err != nil {
continue
}

c.handleMessage(msg)
}
}

func (c *StratumClient) handleMessage(msg StratumMessage) {
switch msg.Method {
case "job":
var params []interface{}
if err := json.Unmarshal(msg.Params, &params); err != nil {
return
}

if len(params) >= 3 {
c.mu.Lock()
c.jobID = params[0].(string)
c.blob = params[1].(string)
c.target = params[2].(string)
if len(params) > 3 {
if height, ok := params[3].(float64); ok {
c.height = uint64(height)
}
}
c.mu.Unlock()
}
}
}

func (c *StratumClient) GetWork() (sealHash, seedHash, target string, height uint64, err error) {
c.mu.Lock()
defer c.mu.Unlock()

if c.blob == "" {
return "", "", "", 0, fmt.Errorf("no job received yet")
}

sealHash = "0x" + c.blob
seedHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
target = c.target
height = c.height

return sealHash, seedHash, target, height, nil
}

func (c *StratumClient) SubmitWork(nonce uint64, mixDigest []byte) error {
nonceHex := fmt.Sprintf("%016x", nonce)
mixDigestHex := hex.EncodeToString(mixDigest)

params := []interface{}{
c.address,
c.jobID,
nonceHex,
mixDigestHex,
}

req := StratumMessage{
ID:     2,
Method: "submit",
Params: mustMarshal(params),
}

return c.send(req)
}

func (c *StratumClient) send(msg StratumMessage) error {
data, err := json.Marshal(msg)
if err != nil {
return err
}

_, err = c.conn.Write(append(data, '\n'))
return err
}

func (c *StratumClient) Close() {
if c.conn != nil {
c.conn.Close()
}
}

func mustMarshal(v interface{}) json.RawMessage {
data, _ := json.Marshal(v)
return data
}
