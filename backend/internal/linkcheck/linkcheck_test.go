package linkcheck

import (
	"net"
	"testing"
)

func TestTarget(t *testing.T) {
	cases := map[string]struct {
		host string
		port int
		ok   bool
	}{
		"https://pve.lan:8006/":   {"pve.lan", 8006, true},
		"http://10.0.0.5":         {"10.0.0.5", 80, true},
		"https://example.com/x?y": {"example.com", 443, true},
		"ftp://nas":               {"", 0, false},
		"not a url":               {"", 0, false},
	}
	for in, want := range cases {
		h, p, ok := Target(in)
		if h != want.host || p != want.port || ok != want.ok {
			t.Errorf("%q: got %s %d %v", in, h, p, ok)
		}
	}
}

func TestUp(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if !Up("127.0.0.1", port) {
		t.Error("open port reported down")
	}
	ln.Close()
	// Cached: still up within the TTL
	if !Up("127.0.0.1", port) {
		t.Error("cache not used")
	}
}
