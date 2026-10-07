// Package portscan checks which TCP ports are open on a host.
package portscan

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IsOpen - check one tcp port
func IsOpen(host, port string) bool {
	return isOpen(context.Background(), net.JoinHostPort(host, port), 3*time.Second)
}

func isOpen(ctx context.Context, target string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}

	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Scan - check ports on ip with a pool of workers and return the open ones, sorted.
// progress, if set, is called with the number of ports checked so far.
func Scan(ctx context.Context, ip string, ports []int, workers int, timeout time.Duration, progress func(done int)) []int {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan int)
	var mu sync.Mutex
	var open []int
	done := 0

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				ok := isOpen(ctx, net.JoinHostPort(ip, strconv.Itoa(p)), timeout)
				mu.Lock()
				if ok {
					open = append(open, p)
				}
				done++
				if progress != nil {
					progress(done)
				}
				mu.Unlock()
			}
		}()
	}

loop:
	for _, p := range ports {
		select {
		case <-ctx.Done():
			break loop
		case jobs <- p:
		}
	}
	close(jobs)
	wg.Wait()

	sort.Ints(open)
	return open
}

// ParseList - turn "top", "all" or "22,80,8000-8100" into a list of ports
func ParseList(str string) ([]int, error) {
	str = strings.TrimSpace(str)

	switch str {
	case "", "top":
		return TopPorts(), nil
	case "all":
		return rangePorts(1, 65535), nil
	}

	seen := make(map[int]bool)
	var ports []int

	for _, part := range strings.Split(str, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		first, last, found := strings.Cut(part, "-")
		if !found {
			last = first
		}
		a, errA := strconv.Atoi(strings.TrimSpace(first))
		b, errB := strconv.Atoi(strings.TrimSpace(last))
		if errA != nil || errB != nil || a < 1 || b > 65535 || a > b {
			return nil, fmt.Errorf("invalid port or range %q", part)
		}
		for p := a; p <= b; p++ {
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("no ports in %q", str)
	}
	sort.Ints(ports)
	return ports, nil
}

func rangePorts(a, b int) []int {
	ports := make([]int, 0, b-a+1)
	for p := a; p <= b; p++ {
		ports = append(ports, p)
	}
	return ports
}

// TopPorts - common ports, weighted toward home lab services
func TopPorts() []int {
	ports := make([]int, 0, len(services))
	for p := range services {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports
}

type service struct {
	name string
	web  string // "http", "https" or ""
}

var services = map[int]service{
	21:    {"FTP", ""},
	22:    {"SSH", ""},
	23:    {"Telnet", ""},
	25:    {"SMTP", ""},
	53:    {"DNS", ""},
	80:    {"HTTP", "http"},
	81:    {"HTTP", "http"},
	88:    {"Kerberos", ""},
	110:   {"POP3", ""},
	111:   {"RPC", ""},
	135:   {"MS RPC", ""},
	139:   {"NetBIOS", ""},
	143:   {"IMAP", ""},
	161:   {"SNMP", ""},
	389:   {"LDAP", ""},
	443:   {"HTTPS", "https"},
	445:   {"SMB", ""},
	465:   {"SMTPS", ""},
	548:   {"AFP", ""},
	554:   {"RTSP", ""},
	587:   {"SMTP submission", ""},
	631:   {"IPP printing", "http"},
	636:   {"LDAPS", ""},
	853:   {"DNS over TLS", ""},
	873:   {"rsync", ""},
	993:   {"IMAPS", ""},
	995:   {"POP3S", ""},
	1194:  {"OpenVPN", ""},
	1433:  {"MS SQL", ""},
	1883:  {"MQTT", ""},
	1900:  {"UPnP", ""},
	2049:  {"NFS", ""},
	2375:  {"Docker API", ""},
	2376:  {"Docker API (TLS)", ""},
	3000:  {"Grafana / web app", "http"},
	3001:  {"Uptime Kuma", "http"},
	3306:  {"MySQL", ""},
	3389:  {"RDP", ""},
	3493:  {"NUT", ""},
	4443:  {"HTTPS alt", "https"},
	5000:  {"Web app / Synology", "http"},
	5001:  {"Synology HTTPS", "https"},
	5055:  {"Open Notebook", "http"},
	5353:  {"mDNS", ""},
	5432:  {"PostgreSQL", ""},
	5900:  {"VNC", ""},
	6379:  {"Redis", ""},
	6443:  {"Kubernetes API", "https"},
	7878:  {"Radarr", "http"},
	8000:  {"HTTP alt", "http"},
	8006:  {"Proxmox VE", "https"},
	8007:  {"Proxmox Backup", "https"},
	8008:  {"HTTP alt", "http"},
	8080:  {"HTTP alt", "http"},
	8081:  {"HTTP alt", "http"},
	8083:  {"HTTP alt", "http"},
	8086:  {"InfluxDB", "http"},
	8096:  {"Jellyfin", "http"},
	8123:  {"Home Assistant", "http"},
	8181:  {"HTTP alt", "http"},
	8443:  {"HTTPS alt", "https"},
	8840:  {"Gimlé", "http"},
	8883:  {"MQTT TLS", ""},
	8888:  {"HTTP alt", "http"},
	8920:  {"Jellyfin HTTPS", "https"},
	8989:  {"Sonarr", "http"},
	9000:  {"Portainer / web app", "http"},
	9090:  {"Prometheus / Cockpit", "http"},
	9091:  {"Transmission", "http"},
	9100:  {"Node exporter", "http"},
	9443:  {"Portainer HTTPS", "https"},
	9696:  {"Prowlarr", "http"},
	10000: {"Webmin", "https"},
	11434: {"Ollama", "http"},
	19999: {"Netdata", "http"},
	27017: {"MongoDB", ""},
	32400: {"Plex", "http"},
	51820: {"WireGuard", ""},
}

// Service - a guess at what usually listens on a port, and whether it serves a web page
func Service(port int) (name, web string) {
	s, ok := services[port]
	if !ok {
		return "", ""
	}
	return s.name, s.web
}

// KnownPort - a port in the "top" list and the service usually on it
type KnownPort struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
}

// TopList - the "top" ports with their usual services
func TopList() []KnownPort {
	list := make([]KnownPort, 0, len(services))
	for _, p := range TopPorts() {
		list = append(list, KnownPort{Port: p, Service: services[p].name})
	}
	return list
}
