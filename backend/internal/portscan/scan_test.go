package portscan

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestParseList(t *testing.T) {
	ports, err := ParseList("443, 22,8000-8002,22")
	if err != nil {
		t.Fatal(err)
	}
	want := []int{22, 443, 8000, 8001, 8002}
	if len(ports) != len(want) {
		t.Fatalf("got %v", ports)
	}
	for i := range want {
		if ports[i] != want[i] {
			t.Fatalf("got %v", ports)
		}
	}

	if all, _ := ParseList("all"); len(all) != 65535 {
		t.Errorf("all = %d ports", len(all))
	}
	if top, _ := ParseList(""); len(top) != len(services) {
		t.Errorf("default should be the top list")
	}
	for _, bad := range []string{"0", "70000", "x", "90-80", ",", "1-"} {
		if _, err := ParseList(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestScan(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	openPort := ln.Addr().(*net.TCPAddr).Port

	// A port that was just open and is now closed
	tmp, _ := net.Listen("tcp", "127.0.0.1:0")
	closedPort := tmp.Addr().(*net.TCPAddr).Port
	tmp.Close()

	calls := 0
	open := Scan(context.Background(), "127.0.0.1", []int{closedPort, openPort}, 4, time.Second, func(int) { calls++ })

	if len(open) != 1 || open[0] != openPort {
		t.Errorf("open = %v, want [%d]", open, openPort)
	}
	if calls != 2 {
		t.Errorf("progress called %d times", calls)
	}
	if !IsOpen("127.0.0.1", strconv.Itoa(openPort)) {
		t.Errorf("IsOpen false for open port")
	}
}

func TestService(t *testing.T) {
	if name, web := Service(8006); name != "Proxmox VE" || web != "https" {
		t.Errorf("8006 = %s %s", name, web)
	}
	if name, web := Service(12345); name != "" || web != "" {
		t.Errorf("unknown port should be empty")
	}
}
