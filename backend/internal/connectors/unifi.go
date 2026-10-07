package connectors

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// UniFi's classic API answers {"meta":{"rc":"ok"},"data":[...]}
type unifiLegacy[T any] struct {
	Meta struct {
		RC  string `json:"rc"`
		Msg string `json:"msg"`
	} `json:"meta"`
	Data []T `json:"data"`
}

type unifiClient struct {
	Mac      string `json:"mac"`
	IP       string `json:"ip"`       // stat/sta: current address
	LastIP   string `json:"last_ip"`  // rest/user
	FixedIP  string `json:"fixed_ip"` // rest/user, with use_fixedip
	UseFixed bool   `json:"use_fixedip"`
	Name     string `json:"name"`     // alias set in UniFi
	Hostname string `json:"hostname"` // what the device calls itself in DHCP
}

// The integration API answers {"offset":0,"limit":200,"count":N,"totalCount":N,"data":[...]}
type unifiPage[T any] struct {
	Offset     int `json:"offset"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
	Data       []T `json:"data"`
}

type unifiSite struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	InternalReference string `json:"internalReference"`
}

type unifiIntClient struct {
	Name       string `json:"name"`
	IPAddress  string `json:"ipAddress"`
	MacAddress string `json:"macAddress"`
}

// fetchUniFi - client names, DHCP host names and addresses from a UniFi OS
// console (Dream Machine, Cloud Gateway). Uses an API key made under
// UniFi Network > Settings > Control Plane > Integrations. The classic API
// also lists offline clients and fixed IPs, so it is tried first; the
// documented integration API, which lists connected clients, is the fallback.
func fetchUniFi(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	cl.auth = func(req *http.Request) { req.Header.Set("X-API-KEY", cl.token) }
	site := c.Site
	if site == "" {
		site = "default"
	}

	snap, legacyErr := fetchUniFiLegacy(ctx, cl, site)
	if legacyErr == nil {
		return snap, nil
	}
	snap, err := fetchUniFiIntegration(ctx, cl, c.Site)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%v (classic API: %v)", err, legacyErr)
	}
	return snap, nil
}

func fetchUniFiLegacy(ctx context.Context, cl *client, site string) (Snapshot, error) {
	base := "/proxy/network/api/s/" + url.PathEscape(site)

	var known, active unifiLegacy[unifiClient]
	if err := cl.getJSON(ctx, base+"/rest/user", &known); err != nil {
		return Snapshot{}, err
	}
	if known.Meta.RC != "ok" {
		return Snapshot{}, fmt.Errorf("rest/user: %s", known.Meta.Msg)
	}
	if err := cl.getJSON(ctx, base+"/stat/sta", &active); err != nil {
		return Snapshot{}, err
	}

	byMac := make(map[string]*Host)
	var order []string
	add := func(u unifiClient) {
		mac := NormMac(u.Mac)
		if mac == "" {
			return
		}
		h := byMac[mac]
		if h == nil {
			h = &Host{Mac: mac}
			byMac[mac] = h
			order = append(order, mac)
		}
		ip := u.IP
		if ip == "" && u.UseFixed {
			ip = u.FixedIP
		}
		if ip == "" {
			ip = u.LastIP
		}
		if ip != "" && h.IP == "" {
			h.IP = ip
		}
		if h.Name == "" {
			h.Name = firstNonEmpty(u.Name, u.Hostname)
		}
	}
	for _, u := range active.Data { // current addresses first
		add(u)
	}
	for _, u := range known.Data {
		add(u)
	}

	var snap Snapshot
	for _, mac := range order {
		snap.Hosts = append(snap.Hosts, *byMac[mac])
	}
	return snap, nil
}

func fetchUniFiIntegration(ctx context.Context, cl *client, site string) (Snapshot, error) {
	const base = "/proxy/network/integration/v1"

	var sites unifiPage[unifiSite]
	if err := cl.getJSON(ctx, base+"/sites?limit=200", &sites); err != nil {
		return Snapshot{}, err
	}

	var snap Snapshot
	for _, s := range sites.Data {
		if site != "" && site != s.Name && site != s.InternalReference && site != s.ID {
			continue
		}
		for offset := 0; ; {
			var page unifiPage[unifiIntClient]
			path := fmt.Sprintf("%s/sites/%s/clients?offset=%d&limit=200", base, url.PathEscape(s.ID), offset)
			if err := cl.getJSON(ctx, path, &page); err != nil {
				return Snapshot{}, err
			}
			for _, u := range page.Data {
				snap.Hosts = append(snap.Hosts, Host{IP: u.IPAddress, Mac: NormMac(u.MacAddress), Name: u.Name})
			}
			offset += len(page.Data)
			if len(page.Data) == 0 || offset >= page.TotalCount {
				break
			}
		}
	}
	return snap, nil
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
