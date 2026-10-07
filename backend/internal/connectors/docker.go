package connectors

import (
	"context"
	"sort"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// dockerContainer - one entry of Docker's GET /containers/json
type dockerContainer struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
	Status string   `json:"Status"`
	Ports  []dockerPort
	Labels map[string]string `json:"Labels"`
}

type dockerPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// fetchDocker - read containers from a Docker API, such as a read-only
// docker-socket-proxy with CONTAINERS=1
func fetchDocker(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	var list []dockerContainer
	if err := cl.getJSON(ctx, "/containers/json?all=1", &list); err != nil {
		return Snapshot{}, err
	}
	ip := c.HostIP
	if ip == "" {
		ip = urlHostIP(c.URL)
	}
	h := Host{IP: ip}
	for _, d := range list {
		name := ""
		if len(d.Names) > 0 {
			name = strings.TrimPrefix(d.Names[0], "/")
		}
		h.Containers = append(h.Containers, container(d.ID, name, d.Image, d.State, d.Status, d.Labels, d.Ports))
	}
	return Snapshot{Hosts: []Host{h}}, nil
}

// container - a stored container from Docker's fields. Ports listed once
// for IPv4 and once for IPv6 are merged.
func container(id, name, image, state, status string, labels map[string]string, ports []dockerPort) models.Container {
	ct := models.Container{
		CID: id, Name: name, Image: image, State: state, Status: status,
		Project: labels["com.docker.compose.project"],
		Ports:   []models.ContainerPort{},
	}
	seen := make(map[string]bool)
	for _, p := range ports {
		proto := p.Type
		if proto == "" {
			proto = "tcp"
		}
		key := proto + ":" + itoa(p.PrivatePort) + ":" + itoa(p.PublicPort)
		if seen[key] {
			continue
		}
		seen[key] = true
		ct.Ports = append(ct.Ports, models.ContainerPort{Private: p.PrivatePort, Public: p.PublicPort, Proto: proto})
	}
	sort.Slice(ct.Ports, func(i, j int) bool {
		a, b := ct.Ports[i], ct.Ports[j]
		if a.Public != b.Public {
			return a.Public < b.Public
		}
		return a.Private < b.Private
	})
	return ct
}
