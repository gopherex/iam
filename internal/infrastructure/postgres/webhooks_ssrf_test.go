package postgres

import (
	"net"
	"testing"
)

// TestIsBlockedWebhookIP guards the webhook SSRF blocklist: cloud-metadata,
// RFC1918, link-local, ULA, CGNAT and unspecified addresses are refused, while
// public addresses and the loopback dev escape hatch are allowed.
func TestIsBlockedWebhookIP(t *testing.T) {
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"169.254.169.254", true},       // cloud metadata (link-local)
		{"10.0.0.1", true},              // RFC1918
		{"172.16.5.4", true},            // RFC1918
		{"192.168.1.1", true},           // RFC1918
		{"100.64.0.1", true},            // CGNAT
		{"0.0.0.0", true},               // unspecified
		{"fe80::1", true},               // link-local v6
		{"fc00::1", true},               // ULA v6
		{"::", true},                    // unspecified v6
		{"8.8.8.8", false},              // public
		{"1.1.1.1", false},              // public
		{"127.0.0.1", false},            // loopback (dev escape hatch)
		{"::1", false},                  // loopback v6
		{"2606:4700:4700::1111", false}, // public v6
	}

	for _, c := range cases {
		ip := net.ParseIP(c.ip)
		if ip == nil {
			t.Fatalf("bad test ip %q", c.ip)
		}

		if got := isBlockedWebhookIP(ip); got != c.blocked {
			t.Errorf("isBlockedWebhookIP(%s) = %v, want %v", c.ip, got, c.blocked)
		}
	}
}
