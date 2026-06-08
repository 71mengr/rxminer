package pool

import (
	"bufio"
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestHandleLoginResultStoresInitialJob(t *testing.T) {
	client := &StratumClient{jobCh: make(chan struct{}, 1)}

	client.handleMessage(StratumMessage{
		ID:     1,
		Result: json.RawMessage(`{"id":"session-1","job":{"job_id":"job-1","blob":"abcdef","target":"ffff","height":42,"seed_hash":"1234"},"status":"OK"}`),
	})

	sealHash, seedHash, target, height, err := client.GetWork()
	if err != nil {
		t.Fatalf("GetWork returned error: %v", err)
	}
	if sealHash != "0xabcdef" {
		t.Fatalf("sealHash = %q, want %q", sealHash, "0xabcdef")
	}
	if seedHash != "0x1234" {
		t.Fatalf("seedHash = %q, want %q", seedHash, "0x1234")
	}
	if target != "ffff" {
		t.Fatalf("target = %q, want %q", target, "ffff")
	}
	if height != 42 {
		t.Fatalf("height = %d, want %d", height, uint64(42))
	}
	if client.sessionID != "session-1" {
		t.Fatalf("sessionID = %q, want %q", client.sessionID, "session-1")
	}
}

func TestHandleJobNotificationSupportsObjectParams(t *testing.T) {
	client := &StratumClient{jobCh: make(chan struct{}, 1)}

	client.handleMessage(StratumMessage{
		Method: "job",
		Params: json.RawMessage(`{"job_id":"job-2","blob":"beef","target":"00ff","height":100}`),
	})

	sealHash, _, target, height, err := client.GetWork()
	if err != nil {
		t.Fatalf("GetWork returned error: %v", err)
	}
	if sealHash != "0xbeef" {
		t.Fatalf("sealHash = %q, want %q", sealHash, "0xbeef")
	}
	if target != "00ff" {
		t.Fatalf("target = %q, want %q", target, "00ff")
	}
	if height != 100 {
		t.Fatalf("height = %d, want %d", height, uint64(100))
	}
}

func TestHandleJobNotificationSupportsArrayParams(t *testing.T) {
	client := &StratumClient{jobCh: make(chan struct{}, 1)}

	client.handleMessage(StratumMessage{
		Method: "job",
		Params: json.RawMessage(`["job-3","feed","0f0f",7]`),
	})

	sealHash, _, target, height, err := client.GetWork()
	if err != nil {
		t.Fatalf("GetWork returned error: %v", err)
	}
	if sealHash != "0xfeed" {
		t.Fatalf("sealHash = %q, want %q", sealHash, "0xfeed")
	}
	if target != "0f0f" {
		t.Fatalf("target = %q, want %q", target, "0f0f")
	}
	if height != 7 {
		t.Fatalf("height = %d, want %d", height, uint64(7))
	}
}

func TestWaitForWorkWaitsForDelayedJob(t *testing.T) {
	client := &StratumClient{jobCh: make(chan struct{}, 1)}

	go func() {
		time.Sleep(10 * time.Millisecond)
		client.handleMessage(StratumMessage{
			Method: "job",
			Params: json.RawMessage(`{"job_id":"job-4","blob":"cafe","target":"ffff"}`),
		})
	}()

	if err := client.WaitForWork(time.Second); err != nil {
		t.Fatalf("WaitForWork returned error: %v", err)
	}
}

func TestHandleNestedJobResultFromBundledPool(t *testing.T) {
	client := &StratumClient{jobCh: make(chan struct{}, 1)}

	client.handleMessage(StratumMessage{
		ID:     0,
		Result: json.RawMessage(`{"method":"job","params":["job-5","abcd","seed1234","00ff"]}`),
	})

	sealHash, seedHash, target, height, err := client.GetWork()
	if err != nil {
		t.Fatalf("GetWork returned error: %v", err)
	}
	if sealHash != "0xabcd" {
		t.Fatalf("sealHash = %q, want %q", sealHash, "0xabcd")
	}
	if seedHash != "0xseed1234" {
		t.Fatalf("seedHash = %q, want %q", seedHash, "0xseed1234")
	}
	if target != "00ff" {
		t.Fatalf("target = %q, want %q", target, "00ff")
	}
	if height != 0 {
		t.Fatalf("height = %d, want %d", height, uint64(0))
	}
}

func TestSubmitWorkUsesBundledPoolArrayParams(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	client := &StratumClient{
		conn:      clientConn,
		address:   "wallet-1",
		sessionID: "session-1",
		jobID:     "job-6",
	}

	done := make(chan StratumMessage, 1)
	go func() {
		line, err := bufio.NewReader(server).ReadBytes('\n')
		if err != nil {
			t.Errorf("ReadBytes returned error: %v", err)
			return
		}
		var msg StratumMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			t.Errorf("Unmarshal returned error: %v", err)
			return
		}
		done <- msg
	}()

	if err := client.SubmitWork(42, []byte{0xab, 0xcd}); err != nil {
		t.Fatalf("SubmitWork returned error: %v", err)
	}

	select {
	case msg := <-done:
		if msg.Method != "submit" {
			t.Fatalf("Method = %q, want %q", msg.Method, "submit")
		}
		var params []string
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			t.Fatalf("submit params are not an array: %v", err)
		}
		want := []string{"session-1", "job-6", "000000000000002a", "abcd"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("params = %#v, want %#v", params, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for submit message")
	}
}
