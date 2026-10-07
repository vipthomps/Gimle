package routines

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/notify"
	"github.com/vipthomps/gimle/backend/internal/portscan"
)

const dateFormat = "2006-01-02 15:04:05"

var (
	jobsMu   sync.Mutex
	jobs     = make(map[int]*models.PortJob)
	quitPort = make(chan bool)
)

// PortJob - current or last port scan of a host
func PortJob(hostID int) models.PortJob {
	jobsMu.Lock()
	defer jobsMu.Unlock()

	if j, ok := jobs[hostID]; ok {
		return *j
	}
	return models.PortJob{HostID: hostID}
}

// StartPortScan - scan one host in the background. Returns an error if the
// port list is invalid or a scan of this host is already running.
func StartPortScan(host models.Host, list string) error {
	ports, err := portscan.ParseList(list)
	if err != nil {
		return err
	}
	if !claimJob(host.ID, list, len(ports)) {
		return fmt.Errorf("a port scan of this host is already running")
	}
	go runPortScan(context.Background(), host, list, ports, false)
	return nil
}

func claimJob(hostID int, list string, total int) bool {
	jobsMu.Lock()
	defer jobsMu.Unlock()

	if j, ok := jobs[hostID]; ok && j.Running {
		return false
	}
	jobs[hostID] = &models.PortJob{
		HostID:  hostID,
		List:    list,
		Running: true,
		Total:   total,
		Started: time.Now().Format(dateFormat),
	}
	return true
}

// runPortScan - scan, store the result and, for scheduled scans, notify about changes
func runPortScan(ctx context.Context, host models.Host, list string, ports []int, scheduled bool) {
	slog.Info("Port scan started", "ip", host.IP, "ports", len(ports))

	timeout := time.Duration(conf.AppConfig.PortTimeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 700 * time.Millisecond
	}

	open := portscan.Scan(ctx, host.IP, ports, conf.AppConfig.PortWorkers, timeout, func(done int) {
		jobsMu.Lock()
		jobs[host.ID].Done = done
		jobsMu.Unlock()
	})

	if ctx.Err() == nil {
		_, scannedBefore := gdb.SelectPortScan(host.Mac)
		added, removed := storePorts(host, ports, open)

		probeWeb(ctx, host)
		gdb.SavePortScan(models.PortScan{Mac: host.Mac, Date: time.Now().Format(dateFormat), List: list})

		if scheduled && scannedBefore && (len(added) > 0 || len(removed) > 0) {
			notify.Ports(host, added, removed)
		}
		slog.Info("Port scan finished", "ip", host.IP, "open", open)
	}

	jobsMu.Lock()
	j := jobs[host.ID]
	j.Running = false
	j.Open = len(open)
	j.Finished = time.Now().Format(dateFormat)
	jobsMu.Unlock()
}

// storePorts - update the stored ports of a host. Only ports that were part
// of this scan are touched, so a short scan keeps ports a full scan found.
func storePorts(host models.Host, scanned, open []int) (added, removed []int) {
	now := time.Now().Format(dateFormat)

	inScan := make(map[int]bool, len(scanned))
	for _, p := range scanned {
		inScan[p] = true
	}
	isOpen := make(map[int]bool, len(open))
	for _, p := range open {
		isOpen[p] = true
	}

	stored := make(map[int]bool)
	for _, p := range gdb.SelectPorts(host.Mac) {
		stored[p.Port] = true
		switch {
		case isOpen[p.Port]:
			p.IP, p.Last = host.IP, now
			gdb.SavePort(p)
		case inScan[p.Port]:
			gdb.DeletePort(p.ID)
			removed = append(removed, p.Port)
		}
	}

	for _, p := range open {
		if stored[p] {
			continue
		}
		name, web := portscan.Service(p)
		gdb.SavePort(models.Port{Mac: host.Mac, IP: host.IP, Port: p, Service: name, Web: web, First: now, Last: now})
		added = append(added, p)
	}
	return added, removed
}

// PortScanRestart - start or restart scheduled port scans
func PortScanRestart() {
	close(quitPort)
	quitPort = make(chan bool)

	if conf.AppConfig.PortScan {
		slog.Info("Scheduled port scans every " + fmt.Sprint(conf.AppConfig.PortInterval) + " minutes")
		go schedulePortScans(quitPort)
	}
}

func schedulePortScans(quit chan bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-quit
		cancel()
	}()

	var last time.Time
	for {
		interval := time.Duration(conf.AppConfig.PortInterval) * time.Minute
		if interval < time.Minute {
			interval = time.Minute
		}
		if time.Since(last) >= interval {
			scanAllPorts(ctx)
			last = time.Now()
		}
		select {
		case <-quit:
			return
		case <-time.After(time.Minute):
		}
	}
}

func scanAllPorts(ctx context.Context) {
	list := conf.AppConfig.PortList
	ports, err := portscan.ParseList(list)
	if err != nil {
		slog.Error("Bad port list, skipping scheduled port scan", "list", list, "err", err)
		return
	}

	hosts, _ := gdb.Select("now")
	skip := skippedNets()

	for _, h := range hosts {
		if ctx.Err() != nil {
			return
		}
		if h.Now != 1 || h.IP == "" || inNets(h.IP, skip) {
			continue
		}
		if !claimJob(h.ID, list, len(ports)) {
			continue // a manual scan of this host is running
		}
		runPortScan(ctx, h, list, ports, true)
	}
}

func skippedNets() (nets []*net.IPNet) {
	for _, s := range gdb.SelectSubnets() {
		if !s.SkipPorts {
			continue
		}
		if _, n, err := net.ParseCIDR(s.CIDR); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

func inNets(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	for _, n := range nets {
		if ip != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

// probeWeb - find which open ports of a host serve web pages, and their titles.
// Ports known to be something else (SSH, DNS, databases) are skipped.
func probeWeb(ctx context.Context, host models.Host) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for _, p := range gdb.SelectPorts(host.Mac) {
		if name, web := portscan.Service(p.Port); name != "" && web == "" {
			continue
		}
		wg.Add(1)
		go func(p models.Port) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			web, title := portscan.Probe(host.IP, p.Port, 3*time.Second)
			if web == "" && p.Web != "" {
				return // keep what the port list says if the page did not answer this time
			}
			if web != p.Web || title != p.Title {
				p.Web, p.Title = web, title
				gdb.SavePort(p)
			}
		}(p)
	}
	wg.Wait()
}
