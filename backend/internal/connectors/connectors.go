// Package connectors reads hosts, names, services, containers and guests from
// other systems on the network: the Docker API, Dockhand, Scanopy, a UniFi
// console, Technitium DNS, Proxmox VE and Caddy. Every connector is read-only.
package connectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// Kinds - connector kinds and what each needs
var Kinds = map[string]KindInfo{
	"docker":     {Label: "Docker API", Token: "optional", Example: "http://192.168.1.20:2375"},
	"dockhand":   {Label: "Dockhand", Token: "optional", Example: "http://192.168.1.20:3000"},
	"scanopy":    {Label: "Scanopy", Token: "required", Example: "http://192.168.1.20:60072"},
	"unifi":      {Label: "UniFi console", Token: "required", Example: "https://192.168.1.1"},
	"technitium": {Label: "Technitium DNS", Token: "required", Example: "http://192.168.1.53:5380"},
	"proxmox":    {Label: "Proxmox VE", Token: "required", Example: "https://192.168.1.2:8006"},
	"caddy":      {Label: "Caddy", Token: "optional", Example: "http://192.168.1.30:2020"},
}

// KindInfo - how the config page describes a connector kind
type KindInfo struct {
	Label   string
	Token   string // "required" or "optional"
	Example string
}

// Snapshot - what one sync found
type Snapshot struct {
	Hosts []Host
	Sites []Site // names a reverse proxy serves, which become bookmarks
}

// Site - a name a reverse proxy serves and where it forwards to
type Site struct {
	Name     string // photos.example.lan
	URL      string // https://photos.example.lan
	Upstream string // as the proxy has it, like 10.0.0.5:2283
	IP       string // upstream IP, "" when it can't be resolved from here
	Port     int
}

// Host - a host as another system sees it. Mac or IP identifies it.
type Host struct {
	IP         string
	Mac        string
	Name       string // friendly name or alias
	DNS        string // full DNS name
	Containers []models.Container
	Services   []Service
	Guests     []models.Guest // VMs and LXCs, when this host is a hypervisor node
}

// Service - a service another system found on a host
type Service struct {
	Port     int
	Proto    string
	Name     string
	Category string
}

// Fetch - read everything a connector offers
func Fetch(ctx context.Context, c models.Connector) (Snapshot, error) {
	cl := newClient(c)
	switch c.Kind {
	case "docker":
		return fetchDocker(ctx, cl, c)
	case "dockhand":
		return fetchDockhand(ctx, cl, c)
	case "scanopy":
		return fetchScanopy(ctx, cl, c)
	case "unifi":
		return fetchUniFi(ctx, cl, c)
	case "technitium":
		return fetchTechnitium(ctx, cl, c)
	case "proxmox":
		return fetchProxmox(ctx, cl, c)
	case "caddy":
		return fetchCaddy(ctx, cl, c)
	}
	return Snapshot{}, fmt.Errorf("unknown connector kind %q", c.Kind)
}

// Check - a connector's settings make sense
func Check(c models.Connector) error {
	if _, ok := Kinds[c.Kind]; !ok {
		return fmt.Errorf("unknown connector kind %q", c.Kind)
	}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("URL must start with http:// or https://, like %s", Kinds[c.Kind].Example)
	}
	if c.HostIP != "" && net.ParseIP(c.HostIP) == nil {
		return fmt.Errorf("host IP %q is not an IP address", c.HostIP)
	}
	return nil
}

type client struct {
	http  *http.Client
	base  string
	token string
	auth  func(req *http.Request) // adds credentials
}

func newClient(c models.Connector) *client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if c.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opted in per connector
	}
	cl := &client{
		http:  &http.Client{Timeout: 20 * time.Second, Transport: tr},
		base:  strings.TrimRight(c.URL, "/"),
		token: c.Token,
	}
	cl.auth = func(req *http.Request) {
		if cl.token != "" {
			req.Header.Set("Authorization", "Bearer "+cl.token)
		}
	}
	return cl
}

// getJSON - GET base+path and decode the JSON answer into out
func (cl *client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cl.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	cl.auth(req)

	resp, err := cl.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%s: not allowed (HTTP %d), check the token", path, resp.StatusCode)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: HTTP %d: %s", path, resp.StatusCode, snippet(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: unexpected answer: %v", path, err)
	}
	return nil
}

func errorf(format string, a ...any) error { return fmt.Errorf(format, a...) }

func snippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

// urlHostIP - the IP in a URL, resolving a host name if needed
func urlHostIP(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	h := u.Hostname()
	if net.ParseIP(h) != nil {
		return h
	}
	ips, err := net.LookupIP(h)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

// NormMac - lower case, colon separated
func NormMac(mac string) string {
	mac = strings.ToLower(strings.TrimSpace(mac))
	mac = strings.ReplaceAll(mac, "-", ":")
	if len(mac) == 12 && !strings.Contains(mac, ":") {
		var parts []string
		for i := 0; i < 12; i += 2 {
			parts = append(parts, mac[i:i+2])
		}
		mac = strings.Join(parts, ":")
	}
	return mac
}
