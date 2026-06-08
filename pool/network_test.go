package pool

import "testing"

func TestParsePublicIPv4(t *testing.T) {
	got := parsePublicIPv4(" 93.184.216.34\n")
	if got != "93.184.216.34" {
		t.Fatalf("parsePublicIPv4 returned %q, want %q", got, "93.184.216.34")
	}
}

func TestParsePublicIPv4RejectsPrivateAddresses(t *testing.T) {
	privateAddresses := []string{
		"10.0.0.4",
		"172.16.0.1",
		"192.168.1.20",
		"127.0.0.1",
		"0.0.0.0",
	}

	for _, address := range privateAddresses {
		if got := parsePublicIPv4(address); got != "" {
			t.Fatalf("parsePublicIPv4(%q) = %q, want empty string", address, got)
		}
	}
}
