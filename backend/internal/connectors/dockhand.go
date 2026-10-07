package connectors

import (
	"context"
	"fmt"
	"strconv"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// dockhandEnv - one entry of Dockhand's GET /api/environments
type dockhandEnv struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	ConnectionType string  `json:"connectionType"` // socket, direct, hawser-standard, hawser-edge
	Host           string  `json:"host"`
	PublicIP       *string `json:"publicIp"`
}

// dockhandContainer - one entry of Dockhand's GET /api/containers?env=
type dockhandContainer struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Image  string            `json:"image"`
	State  string            `json:"state"`
	Status string            `json:"status"`
	Ports  []dockerPort      `json:"ports"`
	Labels map[string]string `json:"labels"`
}

// fetchDockhand - containers of every Docker environment Dockhand manages
// (or only the one named in Site). The token is a Dockhand API token
// ("dh_..."); none is needed when Dockhand runs without authentication.
func fetchDockhand(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	var envs []dockhandEnv
	if err := cl.getJSON(ctx, "/api/environments", &envs); err != nil {
		return Snapshot{}, err
	}

	var snap Snapshot
	found := false
	for _, e := range envs {
		if c.Site != "" && c.Site != e.Name && c.Site != strconv.Itoa(e.ID) {
			continue
		}
		found = true

		var list []dockhandContainer
		if err := cl.getJSON(ctx, "/api/containers?env="+strconv.Itoa(e.ID), &list); err != nil {
			return Snapshot{}, fmt.Errorf("environment %s: %w", e.Name, err)
		}
		h := Host{IP: dockhandEnvIP(e, c)}
		if h.IP == "" {
			continue // an edge agent with no public IP set: nothing to match it to
		}
		for _, d := range list {
			ct := container(d.ID, d.Name, d.Image, d.State, d.Status, d.Labels, d.Ports)
			ct.Host = e.Name
			h.Containers = append(h.Containers, ct)
		}
		snap.Hosts = append(snap.Hosts, h)
	}
	if c.Site != "" && !found {
		return Snapshot{}, fmt.Errorf("no Dockhand environment named %q", c.Site)
	}
	return snap, nil
}

// dockhandEnvIP - where an environment's containers publish their ports
func dockhandEnvIP(e dockhandEnv, c models.Connector) string {
	if e.PublicIP != nil && *e.PublicIP != "" {
		return hostIP(*e.PublicIP)
	}
	switch e.ConnectionType {
	case "socket", "":
		if c.HostIP != "" {
			return c.HostIP
		}
		return urlHostIP(c.URL) // Docker runs where Dockhand runs
	case "hawser-edge":
		return ""
	}
	return hostIP(e.Host)
}

// hostIP - an IP for a host name or IP
func hostIP(h string) string {
	return urlHostIP("http://" + h)
}

func itoa(i int) string { return strconv.Itoa(i) }
