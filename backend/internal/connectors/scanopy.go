package connectors

import (
	"context"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/category"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// Scanopy answers {"success":true,"data":[...],"error":null,"meta":{...}}
type scanopyEnvelope[T any] struct {
	Success bool    `json:"success"`
	Data    T       `json:"data"`
	Error   *string `json:"error"`
}

type scanopyHost struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Hostname    string `json:"hostname"`
	// Newer servers keep addresses in ip_addresses, older ones in interfaces
	IPAddresses []scanopyAddr    `json:"ip_addresses"`
	Interfaces  []scanopyAddr    `json:"interfaces"`
	Ports       []scanopyPort    `json:"ports"`
	Services    []scanopyService `json:"services"`
}

type scanopyAddr struct {
	ID         string `json:"id"`
	IPAddress  string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
}

type scanopyPort struct {
	ID       string `json:"id"`
	Number   int    `json:"number"`
	Protocol string `json:"protocol"` // "Tcp" or "Udp"
}

type scanopyService struct {
	Name           string `json:"name"`
	Definition     string `json:"service_definition"`
	Virtualization *struct {
		Type    string `json:"type"` // "Docker", "Podman"
		Details struct {
			ContainerName  string `json:"container_name"`
			ContainerID    string `json:"container_id"`
			ComposeProject string `json:"compose_project"`
		} `json:"details"`
	} `json:"virtualization_metadata"`
	Bindings []struct {
		Type        string  `json:"type"` // "Port" or "IPAddress"
		PortID      string  `json:"port_id"`
		IPAddressID *string `json:"ip_address_id"`
		InterfaceID *string `json:"interface_id"`
	} `json:"bindings"`
}

// fetchScanopy - hosts with their addresses, host names and detected
// services, including containers. Needs a user API key ("scp_u_...").
func fetchScanopy(ctx context.Context, cl *client, c models.Connector) (Snapshot, error) {
	var env scanopyEnvelope[[]scanopyHost]
	if err := cl.getJSON(ctx, "/api/v1/hosts?limit=0", &env); err != nil {
		return Snapshot{}, err
	}
	if !env.Success && env.Error != nil {
		return Snapshot{}, errorf("scanopy: %s", *env.Error)
	}

	var snap Snapshot
	for _, sh := range env.Data {
		snap.Hosts = append(snap.Hosts, scanopyHosts(sh)...)
	}
	return snap, nil
}

// scanopyHosts - one Host per address, each with the services bound to it
func scanopyHosts(sh scanopyHost) []Host {
	addrs := sh.IPAddresses
	if len(addrs) == 0 {
		addrs = sh.Interfaces
	}
	ports := make(map[string]scanopyPort, len(sh.Ports))
	for _, p := range sh.Ports {
		ports[p.ID] = p
	}
	name := sh.DisplayName
	if name == "" {
		name = sh.Name
	}
	dns := ""
	if strings.Contains(sh.Hostname, ".") {
		dns = strings.TrimSuffix(sh.Hostname, ".")
	} else if name == "" {
		name = sh.Hostname
	}

	var out []Host
	for _, a := range addrs {
		if a.IPAddress == "" && a.MacAddress == "" {
			continue
		}
		h := Host{IP: a.IPAddress, Mac: NormMac(a.MacAddress), Name: name, DNS: dns}
		containers := make(map[string]*models.Container)

		for _, s := range sh.Services {
			svcName := s.Name
			if svcName == "" {
				svcName = s.Definition
			}
			var ct *models.Container
			if v := s.Virtualization; v != nil && v.Details.ContainerName != "" {
				ct = containers[v.Details.ContainerName]
				if ct == nil {
					ct = &models.Container{
						CID: v.Details.ContainerID, Name: v.Details.ContainerName,
						Project: v.Details.ComposeProject, Image: s.Definition,
						Status: "seen by Scanopy", Ports: []models.ContainerPort{},
					}
					containers[ct.Name] = ct
				}
			}
			for _, b := range s.Bindings {
				if b.Type != "Port" {
					continue
				}
				bound := b.IPAddressID
				if bound == nil {
					bound = b.InterfaceID
				}
				if bound != nil && *bound != a.ID {
					continue // bound to another of this host's addresses
				}
				p, ok := ports[b.PortID]
				if !ok || p.Number <= 0 {
					continue
				}
				proto := strings.ToLower(p.Protocol)
				h.Services = append(h.Services, Service{
					Port: p.Number, Proto: proto, Name: svcName,
					Category: category.Suggest(s.Definition, svcName),
				})
				if ct != nil {
					ct.Ports = append(ct.Ports, models.ContainerPort{Private: p.Number, Public: p.Number, Proto: proto})
				}
			}
		}
		for _, ct := range containers {
			h.Containers = append(h.Containers, *ct)
		}
		out = append(out, h)
	}
	return out
}
