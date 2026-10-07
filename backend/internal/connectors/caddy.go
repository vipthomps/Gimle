package connectors

import (
	"context"
	"encoding/json"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// Caddy's running config, as GET /config/ returns it. Only the parts that
// say which names proxy where are read.
type caddyConfig struct {
	Apps struct {
		HTTP struct {
			HTTPPort int                    `json:"http_port"`
			Servers  map[string]caddyServer `json:"servers"`
		} `json:"http"`
	} `json:"apps"`
}

type caddyServer struct {
	Listen         []string          `json:"listen"`
	Routes         []caddyRoute      `json:"routes"`
	TLSPolicies    []json.RawMessage `json:"tls_connection_policies"`
	AutomaticHTTPS *struct {
		Disable bool `json:"disable"`
	} `json:"automatic_https"`
}

type caddyRoute struct {
	Match  []caddyMatch   `json:"match"`
	Handle []caddyHandler `json:"handle"`
}

type caddyMatch struct {
	Host []string `json:"host"`
}

type caddyHandler struct {
	Handler   string       `json:"handler"`
	Routes    []caddyRoute `json:"routes"`    // subroute
	Upstreams []caddyDial  `json:"upstreams"` // reverse_proxy
}

type caddyDial struct {
	Dial string `json:"dial"`
}

// fetchCaddy - every site Caddy reverse proxies, with the address it forwards
// to. Caddy's admin API only listens on localhost and can change the config,
// so point the connector at a read-only proxy of GET /config/ (see the guide).
func fetchCaddy(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	var cfg caddyConfig
	path := "/config/"
	if strings.HasSuffix(cl.base, "/config") {
		path = "/"
	}
	if err := cl.getJSON(ctx, path, &cfg); err != nil {
		return Snapshot{}, err
	}

	// Upstreams on localhost run on the Caddy host itself
	caddyIP := c.HostIP
	if caddyIP == "" {
		caddyIP = urlHostIP(c.URL)
	}
	return Snapshot{Sites: caddySites(cfg, caddyIP, net.LookupHost)}, nil
}

// caddySites - one site per host name that ends in a reverse proxy
func caddySites(cfg caddyConfig, caddyIP string, lookup func(string) ([]string, error)) []Site {
	seen := make(map[string]bool)
	var out []Site

	httpPort := cfg.Apps.HTTP.HTTPPort
	if httpPort == 0 {
		httpPort = 80
	}
	names := make([]string, 0, len(cfg.Apps.HTTP.Servers))
	for n := range cfg.Apps.HTTP.Servers {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, n := range names {
		srv := cfg.Apps.HTTP.Servers[n]
		base := caddyBase(srv, httpPort)
		if base == "" {
			continue
		}
		var walk func(routes []caddyRoute, hosts []string)
		walk = func(routes []caddyRoute, hosts []string) {
			for _, r := range routes {
				h := hosts
				for _, m := range r.Match {
					if len(m.Host) > 0 {
						h = m.Host
					}
				}
				for _, hd := range r.Handle {
					switch hd.Handler {
					case "subroute":
						walk(hd.Routes, h)
					case "reverse_proxy":
						if len(hd.Upstreams) == 0 {
							continue
						}
						ip, port := caddyUpstream(hd.Upstreams[0].Dial, caddyIP, lookup)
						for _, name := range h {
							name = strings.ToLower(strings.TrimSuffix(name, "."))
							if name == "" || strings.ContainsAny(name, "*{") || seen[name] {
								continue
							}
							seen[name] = true
							out = append(out, Site{
								Name: name, URL: strings.Replace(base, "HOST", name, 1),
								Upstream: hd.Upstreams[0].Dial, IP: ip, Port: port,
							})
						}
					}
				}
			}
		}
		walk(srv.Routes, nil)
	}
	return out
}

// caddyBase - how a server is reached, with HOST standing for the site name;
// "" for a server that only redirects HTTP to HTTPS. Caddy serves HTTPS on
// every port but its HTTP port unless automatic HTTPS is off.
func caddyBase(srv caddyServer, httpPort int) string {
	for _, l := range srv.Listen {
		_, p, err := net.SplitHostPort(strings.TrimPrefix(l, "tcp/"))
		if err != nil {
			continue
		}
		port, _ := strconv.Atoi(p)
		scheme, def := "https", 443
		if port == httpPort || (len(srv.TLSPolicies) == 0 && srv.AutomaticHTTPS != nil && srv.AutomaticHTTPS.Disable) {
			if tlsOnly(srv) {
				return ""
			}
			scheme, def = "http", 80
		}
		if port == def {
			return scheme + "://HOST"
		}
		return scheme + "://HOST:" + p
	}
	return ""
}

// tlsOnly - a server on the HTTP port whose routes are all Caddy's own redirects
func tlsOnly(srv caddyServer) bool {
	for _, r := range srv.Routes {
		for _, h := range r.Handle {
			if h.Handler != "static_response" {
				return false
			}
		}
	}
	return true
}

// caddyUpstream - the IP and port of a dial address like 10.0.0.5:2283,
// localhost:8080 or immich:2283; the IP is "" when the name doesn't resolve
// here, such as a container name only Docker's network knows
func caddyUpstream(dial, caddyIP string, lookup func(string) ([]string, error)) (string, int) {
	dial = strings.TrimPrefix(strings.TrimPrefix(dial, "tcp/"), "tcp4/")
	host, p, err := net.SplitHostPort(dial)
	if err != nil {
		host, p = dial, "80"
	}
	port, _ := strconv.Atoi(p)
	if strings.Contains(host, "{") {
		return "", 0
	}
	switch {
	case host == "localhost" || host == "::1" || strings.HasPrefix(host, "127."):
		return caddyIP, port
	case net.ParseIP(host) != nil:
		return host, port
	}
	ips, err := lookup(host)
	if err != nil {
		return "", port
	}
	for _, ip := range ips {
		if v := net.ParseIP(ip); v != nil && v.To4() != nil {
			return ip, port
		}
	}
	return "", port
}
