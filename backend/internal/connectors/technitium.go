package connectors

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// Technitium answers {"status":"ok","response":{...}}, or a status of
// "error", "invalid-token" or "2fa-required" with errorMessage, often with HTTP 200
type techEnvelope[T any] struct {
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	Response     T      `json:"response"`
}

type techLeases struct {
	Leases []struct {
		Scope           string  `json:"scope"`
		Type            string  `json:"type"`            // Dynamic, Reserved
		HardwareAddress string  `json:"hardwareAddress"` // AA-BB-CC-DD-EE-FF
		Address         string  `json:"address"`
		HostName        *string `json:"hostName"`
	} `json:"leases"`
}

type techZones struct {
	Zones []struct {
		Name     string `json:"name"`
		Type     string `json:"type"` // Primary, Secondary, Forwarder...
		Disabled bool   `json:"disabled"`
	} `json:"zones"`
}

type techRecords struct {
	Records []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Disabled bool   `json:"disabled"`
		RData    struct {
			IPAddress string `json:"ipAddress"`
		} `json:"rData"`
	} `json:"records"`
}

func techGet[T any](ctx context.Context, cl *client, path string) (T, error) {
	var env techEnvelope[T]
	if err := cl.getJSON(ctx, path, &env); err != nil {
		return env.Response, err
	}
	if env.Status != "ok" {
		msg := env.ErrorMessage
		if msg == "" {
			msg = env.Status
		}
		return env.Response, fmt.Errorf("%s: %s", strings.SplitN(path, "?", 2)[0], msg)
	}
	return env.Response, nil
}

// fetchTechnitium - DHCP leases (MAC, IP, host name) and the A records of
// local primary zones (DNS names). Needs an API token made under
// Administration > Sessions > Create Token, with view access to DHCP and zones.
func fetchTechnitium(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	var snap Snapshot

	// DNS names by IP from local zones
	names := make(map[string][]string)
	zones, err := techGet[techZones](ctx, cl, "/api/zones/list")
	if err != nil {
		return snap, err
	}
	for _, z := range zones.Zones {
		if z.Disabled || z.Type != "Primary" || strings.HasSuffix(z.Name, ".arpa") {
			continue
		}
		q := url.Values{"domain": {z.Name}, "zone": {z.Name}, "listZone": {"true"}}
		recs, err := techGet[techRecords](ctx, cl, "/api/zones/records/get?"+q.Encode())
		if err != nil {
			return snap, err
		}
		for _, r := range recs.Records {
			if r.Type == "A" && !r.Disabled && r.RData.IPAddress != "" && !strings.HasPrefix(r.Name, "*") {
				names[r.RData.IPAddress] = append(names[r.RData.IPAddress], r.Name)
			}
		}
	}

	// DHCP leases; a server without DHCP in use just has none
	leases, err := techGet[techLeases](ctx, cl, "/api/dhcp/leases/list")
	if err != nil {
		return snap, err
	}
	seen := make(map[string]bool)
	for _, l := range leases.Leases {
		h := Host{IP: l.Address, Mac: NormMac(l.HardwareAddress)}
		if l.HostName != nil {
			hn := strings.TrimSuffix(*l.HostName, ".")
			if strings.Contains(hn, ".") {
				h.DNS = hn
			} else {
				h.Name = hn
			}
		}
		if d := dnsNames(names[l.Address]); d != "" {
			h.DNS = d
		}
		seen[l.Address] = true
		snap.Hosts = append(snap.Hosts, h)
	}

	// Hosts with only a DNS record (static addresses)
	var ips []string
	for ip := range names {
		if !seen[ip] {
			ips = append(ips, ip)
		}
	}
	sort.Strings(ips)
	for _, ip := range ips {
		snap.Hosts = append(snap.Hosts, Host{IP: ip, DNS: dnsNames(names[ip])})
	}
	return snap, nil
}

// dnsNames - distinct names, shortest first, space separated
func dnsNames(list []string) string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range list {
		if n = strings.TrimSuffix(n, "."); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) < len(out[j]) })
	return strings.Join(out, " ")
}
