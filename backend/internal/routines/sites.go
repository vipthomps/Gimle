package routines

import (
	"strings"

	"github.com/vipthomps/gimle/backend/internal/connectors"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// applySites - keep one bookmark per site a reverse proxy serves, linked to the
// host and service it forwards to. The connector owns each bookmark's address
// and link; its name, icon, note and tags are the user's once it exists.
// Returns how many sites were linked to a discovered host.
func applySites(c models.Connector, sites []connectors.Site, byIP map[string]models.Host) (linked int, err error) {
	have := make(map[string]models.Bookmark) // URL -> bookmark this connector keeps
	for _, b := range gdb.SelectSourceBookmarks(c.ID) {
		have[b.URL] = b
	}
	manual := make(map[string]bool) // addresses already bookmarked by hand
	for _, b := range gdb.SelectBookmarks() {
		if b.Source == 0 {
			manual[strings.TrimRight(b.URL, "/")] = true
		}
	}

	var containers []models.Container
	keep := make(map[int]bool)
	for _, s := range sites {
		if manual[s.URL] {
			continue
		}
		mac, port := "", 0
		if h, ok := byIP[s.IP]; ok && s.IP != "" {
			mac, port = h.Mac, s.Port
		} else if s.IP == "" {
			// an upstream named after a container, on Docker's own network
			if containers == nil {
				containers = gdb.SelectContainers()
			}
			mac, port = containerUpstream(containers, s.Upstream, s.Port)
		}
		if mac != "" {
			linked++
		}

		b, ok := have[s.URL]
		if !ok {
			b = models.Bookmark{URL: s.URL, Source: c.ID, Name: siteLabel(s.Name)}
		}
		if b.Mac != mac || b.Port != port {
			b.Mac, b.Port = mac, port
		}
		if b.Name == siteLabel(s.Name) && mac != "" {
			if n := serviceName(mac, port); n != "" {
				b.Name = n
			}
		}
		if err := gdb.SaveSourceBookmark(&b); err != nil {
			return linked, err
		}
		keep[b.ID] = true
	}

	var gone []int
	for _, b := range have {
		if !keep[b.ID] {
			gone = append(gone, b.ID)
		}
	}
	gdb.DeleteSourceBookmarks(c.ID, gone)
	return linked, nil
}

// siteLabel - "photos.example.lan" becomes "Photos"
func siteLabel(name string) string {
	label, _, _ := strings.Cut(name, ".")
	if label == "" {
		return name
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

// serviceName - the page title or container name Gimle already has for a service
func serviceName(mac string, port int) string {
	if port == 0 {
		return ""
	}
	for _, p := range gdb.SelectPorts(mac) {
		if p.Port != port {
			continue
		}
		if p.Title != "" {
			return p.Title
		}
		return p.Container
	}
	return ""
}

// containerUpstream - the host and published port of a running container
// named like the upstream (immich-server:2283)
func containerUpstream(list []models.Container, upstream string, private int) (string, int) {
	name, _, _ := strings.Cut(upstream, ":")
	for _, ct := range list {
		if ct.Mac == "" || ct.State != "running" || !strings.EqualFold(ct.Name, name) {
			continue
		}
		for _, p := range ct.Ports {
			if p.Private == private && p.Public != 0 {
				return ct.Mac, p.Public
			}
		}
		return ct.Mac, 0
	}
	return "", 0
}
