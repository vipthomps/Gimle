package connectors

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// Proxmox answers {"data": ...}
type pveData[T any] struct {
	Data T `json:"data"`
}

type pveResource struct {
	ID       string  `json:"id"` // "qemu/101", "lxc/112"
	Type     string  `json:"type"`
	Node     string  `json:"node"`
	VMID     int     `json:"vmid"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Template int     `json:"template"`
	Tags     string  `json:"tags"`
	MaxCPU   float64 `json:"maxcpu"`
	MaxMem   int64   `json:"maxmem"`
	Uptime   int64   `json:"uptime"`
}

type pveClusterStatus struct {
	Type string `json:"type"` // "node" or "cluster"
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type pveInterface struct {
	Name   string `json:"name"`
	HWAddr string `json:"hwaddr"`
	Inet   string `json:"inet"` // "192.168.1.20/24"
}

// pveToken - USER@REALM!TOKENID=SECRET. Also takes the token ID and secret
// separated by spaces or a line break, as copied from the Proxmox dialog.
func pveToken(t string) (string, error) {
	t = strings.TrimPrefix(strings.TrimSpace(t), "PVEAPIToken=")
	if f := strings.Fields(t); len(f) == 2 && !strings.Contains(f[0], "=") {
		t = f[0] + "=" + f[1]
	}
	id, secret, ok := strings.Cut(t, "=")
	if !ok || secret == "" || !strings.Contains(id, "@") || !strings.Contains(id, "!") {
		return "", fmt.Errorf("the token should be USER@REALM!TOKENID=SECRET, for example root@pam!gimle=1234-abcd: the token ID, an = sign, then the secret")
	}
	return t, nil
}

var macRe = regexp.MustCompile(`(?i)^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)

// fetchProxmox - nodes and their VMs and LXCs. The token is a Proxmox API
// token written as USER@REALM!TOKENID=SECRET, made under Datacenter >
// Permissions > API Tokens; the PVEAuditor role on / is enough.
func fetchProxmox(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	token, err := pveToken(cl.token)
	if err != nil {
		return Snapshot{}, err
	}
	cl.auth = func(req *http.Request) {
		req.Header.Set("Authorization", "PVEAPIToken="+token)
	}

	var res pveData[[]pveResource]
	if err := cl.getJSON(ctx, "/api2/json/cluster/resources?type=vm", &res); err != nil {
		return Snapshot{}, err
	}

	// Node addresses; a single node without a cluster is reached at the URL
	nodeIP := make(map[string]string)
	var status pveData[[]pveClusterStatus]
	if err := cl.getJSON(ctx, "/api2/json/cluster/status", &status); err == nil {
		for _, s := range status.Data {
			if s.Type == "node" && s.IP != "" {
				nodeIP[s.Name] = s.IP
			}
		}
	}

	byNode := make(map[string]*Host)
	var nodes []string
	var guestHosts []Host
	for _, r := range res.Data {
		if r.Template == 1 || (r.Type != "qemu" && r.Type != "lxc") {
			continue
		}
		node := byNode[r.Node]
		if node == nil {
			ip := nodeIP[r.Node]
			if ip == "" && (c.HostIP != "" || len(nodes) == 0) {
				ip = firstNonEmpty(c.HostIP, urlHostIP(c.URL))
			}
			node = &Host{IP: ip}
			byNode[r.Node] = node
			nodes = append(nodes, r.Node)
		}

		g := models.Guest{
			Node: r.Node, VMID: r.VMID, Type: r.Type, Name: r.Name, Status: r.Status,
			Tags: r.Tags, CPUs: int(r.MaxCPU), MaxMem: r.MaxMem, Uptime: r.Uptime,
		}
		base := fmt.Sprintf("/api2/json/nodes/%s/%s/%d", url.PathEscape(r.Node), r.Type, r.VMID)

		var cfg pveData[map[string]any]
		if err := cl.getJSON(ctx, base+"/config", &cfg); err == nil {
			g.Mac, g.IP = guestNIC(cfg.Data)
			if g.Name == "" {
				g.Name, _ = cfg.Data["hostname"].(string)
			}
		}
		// A running LXC reports its addresses; VMs need the guest agent, so they are matched by MAC
		if r.Type == "lxc" && r.Status == "running" {
			var ifs pveData[[]pveInterface]
			if err := cl.getJSON(ctx, base+"/interfaces", &ifs); err == nil {
				for _, i := range ifs.Data {
					if NormMac(i.HWAddr) == g.Mac && i.Inet != "" {
						g.IP = strings.SplitN(i.Inet, "/", 2)[0]
					}
				}
			}
		}
		node.Guests = append(node.Guests, g)
		if g.Mac != "" || g.IP != "" {
			guestHosts = append(guestHosts, Host{Mac: g.Mac, IP: g.IP, Name: g.Name})
		}
	}

	var snap Snapshot
	for _, n := range nodes {
		h := byNode[n]
		for i := range h.Guests {
			h.Guests[i].Node = n
		}
		snap.Hosts = append(snap.Hosts, *h)
	}
	snap.Hosts = append(snap.Hosts, guestHosts...)
	return snap, nil
}

// guestNIC - MAC and static IP of a guest's first network device (net0, net1...)
func guestNIC(cfg map[string]any) (mac, ip string) {
	var keys []string
	for k := range cfg {
		if strings.HasPrefix(k, "net") && len(k) > 3 && k[3] >= '0' && k[3] <= '9' {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return netIndex(keys[i]) < netIndex(keys[j]) })

	for _, k := range keys {
		v, _ := cfg[k].(string)
		// VM: "virtio=BC:24:11:AA:BB:CC,bridge=vmbr0"; LXC: "name=eth0,bridge=vmbr0,hwaddr=BC:24:11:AA:BB:CC,ip=dhcp"
		for _, part := range strings.Split(v, ",") {
			key, val, _ := strings.Cut(part, "=")
			switch {
			case macRe.MatchString(val) && mac == "":
				mac = NormMac(val)
			case key == "ip" && ip == "":
				if a, _, err := net.ParseCIDR(val); err == nil {
					ip = a.String()
				}
			}
		}
		if mac != "" {
			return mac, ip
		}
	}
	return "", ""
}

func netIndex(k string) int {
	n := 0
	for _, r := range k[3:] {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
