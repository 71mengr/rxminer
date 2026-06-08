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
conn      net.Conn
url       string
address   string
password  string
mu        sync.Mutex
jobCh     chan struct{}
sessionID string
jobID     string
blob      string
seedHash  string
target    string
height    uint64
lastError string
}

type StratumMessage struct {
ID     uint64          `json:"id"`
Method string          `json:"method"`
Params json.RawMessage `json:"params"`
Result json.RawMessage `json:"result"`
Error  json.RawMessage `json:"error"`
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
jobCh:    make(chan struct{}, 1),
}

// Start listening before login so login responses that include the first job are captured.
go client.listen()

// Login to pool and wait briefly for the first job instead of returning before any work exists.
if err := client.login(); err != nil {
conn.Close()
return nil, err
}

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

if err := c.WaitForWork(15 * time.Second); err != nil {
return err
}
return nil
}

func (c *StratumClient) listen() {
scanner := bufio.NewScanner(c.conn)
scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
if len(msg.Error) > 0 && string(msg.Error) != "null" {
c.setError(string(msg.Error))
return
}

if len(msg.Result) > 0 && string(msg.Result) != "null" {
c.handleLoginResult(msg.Result)
}

switch msg.Method {
case "job":
c.handleJobPayload(msg.Params)
}
}

func (c *StratumClient) handleLoginResult(result json.RawMessage) {
var res map[string]json.RawMessage
if err := json.Unmarshal(result, &res); err != nil {
return
}

if id, ok := rawString(res["id"]); ok {
c.mu.Lock()
c.sessionID = id
c.mu.Unlock()
}

if status, ok := rawString(res["status"]); ok && !strings.EqualFold(status, "OK") {
c.setError(status)
}

if job, ok := res["job"]; ok {
c.handleJobPayload(job)
}
}

func (c *StratumClient) handleJobPayload(payload json.RawMessage) {
if len(payload) == 0 || string(payload) == "null" {
return
}

var job map[string]json.RawMessage
if err := json.Unmarshal(payload, &job); err == nil {
c.storeJob(jobIDFromMap(job), stringFromMap(job, "blob"), stringFromMap(job, "target"), stringFromMap(job, "seed_hash", "seedHash"), uint64FromMap(job, "height"))
return
}

var params []json.RawMessage
if err := json.Unmarshal(payload, &params); err != nil || len(params) < 3 {
return
}

jobID, _ := rawString(params[0])
blob, _ := rawString(params[1])
target, _ := rawString(params[2])
var height uint64
if len(params) > 3 {
height, _ = rawUint64(params[3])
}
c.storeJob(jobID, blob, target, "", height)
}

func (c *StratumClient) storeJob(jobID, blob, target, seedHash string, height uint64) {
if jobID == "" || blob == "" || target == "" {
return
}

c.mu.Lock()
c.jobID = jobID
c.blob = blob
c.target = target
if seedHash != "" {
c.seedHash = seedHash
}
if height != 0 {
c.height = height
}
c.lastError = ""
c.mu.Unlock()

select {
case c.jobCh <- struct{}{}:
default:
}
}

func (c *StratumClient) setError(message string) {
c.mu.Lock()
c.lastError = message
c.mu.Unlock()

select {
case c.jobCh <- struct{}{}:
default:
}
}

func (c *StratumClient) WaitForWork(timeout time.Duration) error {
timer := time.NewTimer(timeout)
defer timer.Stop()

for {
_, _, _, _, err := c.GetWork()
if err == nil {
return nil
}

select {
case <-c.jobCh:
case <-timer.C:
_, _, _, _, err := c.GetWork()
return err
}
}
}

func (c *StratumClient) GetWork() (sealHash, seedHash, target string, height uint64, err error) {
c.mu.Lock()
defer c.mu.Unlock()

if c.blob == "" {
if c.lastError != "" {
return "", "", "", 0, fmt.Errorf("pool error: %s", c.lastError)
}
return "", "", "", 0, fmt.Errorf("no job received yet")
}

sealHash = "0x" + strings.TrimPrefix(c.blob, "0x")
if c.seedHash != "" {
seedHash = "0x" + strings.TrimPrefix(c.seedHash, "0x")
} else {
seedHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
}
target = c.target
height = c.height

return sealHash, seedHash, target, height, nil
}

func (c *StratumClient) SubmitWork(nonce uint64, mixDigest []byte) error {
nonceHex := fmt.Sprintf("%016x", nonce)
mixDigestHex := hex.EncodeToString(mixDigest)

c.mu.Lock()
sessionID := c.sessionID
jobID := c.jobID
c.mu.Unlock()
if sessionID == "" {
sessionID = c.address
}

params := map[string]interface{}{
"id":     sessionID,
"job_id": jobID,
"nonce":  nonceHex,
"result": mixDigestHex,
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

func jobIDFromMap(job map[string]json.RawMessage) string {
return stringFromMap(job, "job_id", "jobId", "id")
}

func stringFromMap(values map[string]json.RawMessage, keys ...string) string {
for _, key := range keys {
if value, ok := rawString(values[key]); ok {
return value
}
}
return ""
}

func uint64FromMap(values map[string]json.RawMessage, keys ...string) uint64 {
for _, key := range keys {
if value, ok := rawUint64(values[key]); ok {
return value
}
}
return 0
}

func rawString(raw json.RawMessage) (string, bool) {
if len(raw) == 0 || string(raw) == "null" {
return "", false
}
var s string
if err := json.Unmarshal(raw, &s); err == nil {
return s, true
}
return "", false
}

func rawUint64(raw json.RawMessage) (uint64, bool) {
if len(raw) == 0 || string(raw) == "null" {
return 0, false
}
var number uint64
if err := json.Unmarshal(raw, &number); err == nil {
return number, true
}
var floatNumber float64
if err := json.Unmarshal(raw, &floatNumber); err == nil {
return uint64(floatNumber), true
}
return 0, false
}
