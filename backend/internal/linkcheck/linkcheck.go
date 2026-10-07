// Package linkcheck tells whether a TCP endpoint answers, with a short cache
// so dashboards can ask often without flooding the network.
package linkcheck

import (
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	timeout = 800 * time.Millisecond
	ttl     = 60 * time.Second
)

type result struct {
	up bool
	at time.Time
}

var (
	mu    sync.Mutex
	cache = make(map[string]result)
)

// Up - whether host:port accepts a TCP connection (cached for a minute)
func Up(host string, port int) bool {
	target := net.JoinHostPort(host, strconv.Itoa(port))

	mu.Lock()
	r, ok := cache[target]
	mu.Unlock()
	if ok && time.Since(r.at) < ttl {
		return r.up
	}

	up := false
	if conn, err := net.DialTimeout("tcp", target, timeout); err == nil {
		conn.Close()
		up = true
	}

	mu.Lock()
	cache[target] = result{up: up, at: time.Now()}
	mu.Unlock()
	return up
}

// Target - host and port a URL points at, with the scheme's default port
func Target(raw string) (host string, port int, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", 0, false
	}
	host = u.Hostname()
	if p := u.Port(); p != "" {
		port, err = strconv.Atoi(p)
		return host, port, err == nil
	}
	switch u.Scheme {
	case "http":
		return host, 80, true
	case "https":
		return host, 443, true
	}
	return "", 0, false
}
