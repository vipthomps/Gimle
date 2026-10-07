// Package names finds a hostname for an IP address without an agent on the host:
// reverse DNS (optionally against a chosen server), then mDNS, then NetBIOS.
package names

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Options - which lookups to try
type Options struct {
	DNSServer string // "" uses the system resolver; "host" or "host:port"
	MDNS      bool
	NetBIOS   bool
	Timeout   time.Duration
}

// Lookup - the first name any enabled method finds, and which method found it
func Lookup(ip string, o Options) (name, source string) {
	if o.Timeout <= 0 {
		o.Timeout = time.Second
	}
	if n := Reverse(ip, o.DNSServer, o.Timeout); n != "" {
		return n, "dns"
	}
	if o.MDNS {
		if n := MDNS(ip, o.Timeout); n != "" {
			return n, "mdns"
		}
	}
	if o.NetBIOS {
		if n := NetBIOS(ip, o.Timeout); n != "" {
			return n, "netbios"
		}
	}
	return "", ""
}

// Reverse - PTR names for an IP, space separated, without trailing dots
func Reverse(ip, server string, timeout time.Duration) string {
	r := net.DefaultResolver
	if server != "" {
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = net.JoinHostPort(server, "53")
		}
		r = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				return d.DialContext(ctx, network, server)
			},
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	found, err := r.LookupAddr(ctx, ip)
	if err != nil {
		return ""
	}
	out := make([]string, 0, len(found))
	for _, n := range found {
		if n = strings.TrimSuffix(n, "."); n != "" {
			out = append(out, n)
		}
	}
	return strings.Join(out, " ")
}

// Short - the host part of a DNS name ("nas.example.lan" -> "nas"); other text is unchanged
func Short(name string) string {
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	name = strings.TrimSuffix(f[0], ".")
	if net.ParseIP(name) != nil {
		return name
	}
	if i := strings.IndexByte(name, '.'); i > 0 {
		return name[:i]
	}
	return name
}

// Vendor - clean up arp-scan's vendor text: "" when unknown, and a short note
// for locally administered (randomized or virtual) MAC addresses
func Vendor(hw string) string {
	hw = strings.TrimSpace(hw)
	switch {
	case strings.HasPrefix(hw, "(Unknown: locally administered"):
		return "Private MAC"
	case strings.HasPrefix(hw, "(Unknown"):
		return ""
	}
	return hw
}

// CheckServer - "" or an IP address, optionally with a port
func CheckServer(server string) error {
	if server == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host, port = server, "53"
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("DNS server must be an IP address, optionally with :port, got %q", server)
	}
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("bad DNS server port %q", port)
	}
	return nil
}
