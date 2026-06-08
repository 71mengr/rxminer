package rpc

import (
"bytes"
"encoding/hex"
"encoding/json"
"fmt"
"io"
"math/big"
"net/http"
)

type Client struct {
url string
}

func NewClient(url string) *Client {
return &Client{url: url}
}

func (c *Client) Call(result interface{}, method string, params ...interface{}) error {
request := map[string]interface{}{
"jsonrpc": "2.0",
"method":  method,
"params":  params,
"id":      1,
}

body, err := json.Marshal(request)
if err != nil {
return err
}

resp, err := http.Post(c.url, "application/json", bytes.NewReader(body))
if err != nil {
return err
}
defer resp.Body.Close()

respBody, err := io.ReadAll(resp.Body)
if err != nil {
return err
}

var response struct {
Result json.RawMessage `json:"result"`
Error  *struct {
Code    int    `json:"code"`
Message string `json:"message"`
} `json:"error"`
}

if err := json.Unmarshal(respBody, &response); err != nil {
return err
}

if response.Error != nil {
return fmt.Errorf("RPC error: %s", response.Error.Message)
}

return json.Unmarshal(response.Result, result)
}

func (c *Client) GetWork() ([]string, error) {
var result []string
err := c.Call(&result, "eth_getWork")
return result, err
}

func (c *Client) SubmitWork(nonceHex, sealHashHex, mixDigestHex string) (bool, error) {
var result bool
err := c.Call(&result, "eth_submitWork", nonceHex, sealHashHex, mixDigestHex)
return result, err
}

func HexToBytes(hexStr string) []byte {
if len(hexStr) >= 2 && hexStr[:2] == "0x" {
hexStr = hexStr[2:]
}
bytes, _ := hex.DecodeString(hexStr)
return bytes
}

func HexToHash(hexStr string) []byte {
return HexToBytes(hexStr)
}

func HexToBig(hexStr string) *big.Int {
if len(hexStr) >= 2 && hexStr[:2] == "0x" {
hexStr = hexStr[2:]
}
if hexStr == "" {
return big.NewInt(0)
}
bigInt := new(big.Int)
bigInt.SetString(hexStr, 16)
return bigInt
}
