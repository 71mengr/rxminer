package pool

import (
	"encoding/json"
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
