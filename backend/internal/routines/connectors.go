package routines

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vipthomps/gimle/backend/internal/connectors"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/portscan"
)

var (
	syncMu  sync.Mutex
	syncing = make(map[int]bool) // connector ID -> sync running
)

// StartConnectors - sync each enabled connector on its own interval
func StartConnectors() {
	go func() {
		last := make(map[int]time.Time)
		for {
			for _, c := range gdb.SelectConnectors() {
				every := time.Duration(max(c.Interval, 1)) * time.Minute
				if c.Enabled && time.Since(last[c.ID]) >= every {
					last[c.ID] = time.Now()
					go func() { _ = SyncConnector(c) }()
				}
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

// SyncConnector - read a connector now and store what it found
func SyncConnector(c models.Connector) error {
	syncMu.Lock()
	if syncing[c.ID] {
		syncMu.Unlock()
		return fmt.Errorf("a sync of %s is already running", c.Name)
	}
	syncing[c.ID] = true
	syncMu.Unlock()
	defer func() {
		syncMu.Lock()
		delete(syncing, c.ID)
		syncMu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	snap, err := connectors.Fetch(ctx, c)
	now := time.Now().Format(dateFormat)
	if err != nil {
		slog.Warn("Connector sync failed", "connector", c.Name, "err", err)
		gdb.SetConnectorResult(c.ID, now, err.Error(), 0)
		return err
	}
	matched, err := applySnapshot(c, snap, now)
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	gdb.SetConnectorResult(c.ID, now, errText, matched)
	slog.Info("Connector synced", "connector", c.Name, "hosts", len(snap.Hosts), "matched", matched)
	return err
}

// applySnapshot - match reported hosts to known ones by MAC, then IP, fill in
// missing names, and store containers and published ports
func applySnapshot(c models.Connector, snap connectors.Snapshot, now string) (matched int, err error) {
	hosts, _ := gdb.Select("now")
	byMac := make(map[string]models.Host, len(hosts))
	byIP := make(map[string]models.Host, len(hosts))
	for _, h := range hosts {
		byMac[connectors.NormMac(h.Mac)] = h
		if h.IP != "" {
			// prefer an online host when two share an old IP
			if old, ok := byIP[h.IP]; !ok || old.Now == 0 {
				byIP[h.IP] = h
			}
		}
	}

	var containers []models.Container
	var guests []models.Guest
	wantPorts := make(map[string]models.Port) // mac/port -> port
	probe := make(map[string]models.Host)

	for _, rh := range snap.Hosts {
		h, ok := byMac[connectors.NormMac(rh.Mac)]
		if !ok || rh.Mac == "" {
			h, ok = byIP[rh.IP]
		}
		if ok {
			matched++
			gdb.FillHost(h.ID, rh.Name, rh.DNS)
		}

		for _, g := range rh.Guests {
			if ok {
				g.NodeMac = h.Mac
			}
			if gh, found := byMac[g.Mac]; found && g.Mac != "" {
				g.HostMac = gh.Mac
			} else if gh, found := byIP[g.IP]; found && g.IP != "" {
				g.HostMac = gh.Mac
			}
			guests = append(guests, g)
		}

		for _, ct := range rh.Containers {
			ct.IP = rh.IP
			if ok {
				ct.Mac = h.Mac
			}
			containers = append(containers, ct)
			if !ok || ct.State != "running" {
				continue
			}
			for _, p := range ct.Ports {
				if p.Public == 0 || p.Proto != "tcp" {
					continue
				}
				name, web := portscan.Service(p.Public)
				wantPorts[h.Mac+"/"+itoa(p.Public)] = models.Port{
					Mac: h.Mac, IP: h.IP, Port: p.Public, Service: name, Web: web,
					Source: c.ID, Container: ct.Name,
				}
				probe[h.Mac] = h
			}
		}
		if !ok {
			continue
		}
		for _, s := range rh.Services {
			if s.Port <= 0 || (s.Proto != "" && s.Proto != "tcp") {
				continue
			}
			key := h.Mac + "/" + itoa(s.Port)
			p := wantPorts[key]
			if p.Mac == "" {
				_, web := portscan.Service(s.Port)
				p = models.Port{Mac: h.Mac, IP: h.IP, Port: s.Port, Web: web, Source: c.ID}
			}
			p.Service, p.Category = s.Name, s.Category
			wantPorts[key] = p
			probe[h.Mac] = h
		}
	}

	if c.Kind == "proxmox" {
		if err := gdb.ReplaceGuests(c.ID, guests); err != nil {
			return matched, err
		}
	}
	if err := gdb.ReplaceContainers(c.ID, containers); err != nil {
		return matched, err
	}
	storeConnectorPorts(c.ID, wantPorts, now)

	// Read titles of new web pages in the background
	go func() {
		for _, h := range probe {
			probeWeb(context.Background(), h)
		}
	}()
	return matched, nil
}

// storeConnectorPorts - add or update the ports a connector reports, and drop
// the ones it reported before but no longer does. Ports a port scan found
// stay; the connector only adds its container name and service details.
func storeConnectorPorts(id int, want map[string]models.Port, now string) {
	done := make(map[string]bool)

	for key, w := range want {
		existing := gdb.SelectPorts(w.Mac)
		var cur *models.Port
		for i := range existing {
			if existing[i].Port == w.Port {
				cur = &existing[i]
				break
			}
		}
		if cur == nil {
			w.First, w.Last = now, now
			gdb.SavePort(w)
		} else {
			cur.Source, cur.Container, cur.Last, cur.IP = id, w.Container, now, w.IP
			if w.Category != "" {
				cur.Category = w.Category
			}
			if w.Service != "" && w.Container == "" {
				cur.Service = w.Service // a named service from Scanopy beats a guess from the port number
			}
			if cur.Web == "" {
				cur.Web = w.Web
			}
			gdb.SavePort(*cur)
		}
		done[key] = true
	}

	for _, p := range gdb.SelectPortsBySource(id) {
		if !done[p.Mac+"/"+itoa(p.Port)] {
			gdb.DeletePort(p.ID)
		}
	}
}

func itoa(i int) string { return fmt.Sprint(i) }
