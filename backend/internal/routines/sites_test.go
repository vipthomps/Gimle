package routines

import (
	"testing"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/connectors"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
)

func TestApplySites(t *testing.T) {
	conf.Start(t.TempDir(), "")
	gdb.Start()
	defer gdb.Close()

	nas := gdb.Update("now", models.Host{Mac: "aa:bb:cc:00:00:05", IP: "10.10.0.5", Now: 1})
	gdb.SavePort(models.Port{Mac: nas.Mac, IP: nas.IP, Port: 2283, Title: "Immich"})
	docker := gdb.Update("now", models.Host{Mac: "aa:bb:cc:00:00:06", IP: "10.10.0.6", Now: 1})
	if err := gdb.ReplaceContainers(9, []models.Container{{Mac: docker.Mac, Name: "homeassistant", State: "running",
		Ports: []models.ContainerPort{{Private: 8123, Public: 18123, Proto: "tcp"}}}}); err != nil {
		t.Fatal(err)
	}
	manual := models.Bookmark{Name: "Wiki", URL: "https://wiki.example.lan"}
	if err := gdb.SaveBookmark(&manual, nil); err != nil {
		t.Fatal(err)
	}

	caddy := models.Connector{ID: 3, Kind: "caddy"}
	hosts, _ := gdb.Select("now")
	byIP := make(map[string]models.Host)
	for _, h := range hosts {
		byIP[h.IP] = h
	}
	sites := []connectors.Site{
		{Name: "photos.example.lan", URL: "https://photos.example.lan", Upstream: "10.10.0.5:2283", IP: "10.10.0.5", Port: 2283},
		{Name: "ha.example.lan", URL: "https://ha.example.lan", Upstream: "homeassistant:8123", Port: 8123},
		{Name: "router.example.lan", URL: "https://router.example.lan", Upstream: "10.10.0.1:443", IP: "10.10.0.1", Port: 443},
		{Name: "wiki.example.lan", URL: "https://wiki.example.lan", Upstream: "10.10.0.7:3000", IP: "10.10.0.7", Port: 3000},
	}
	linked, err := applySites(caddy, sites, byIP)
	if err != nil {
		t.Fatal(err)
	}
	if linked != 2 {
		t.Errorf("linked %d, want 2", linked)
	}

	byURL := func() map[string]models.Bookmark {
		m := make(map[string]models.Bookmark)
		for _, b := range gdb.SelectBookmarks() {
			m[b.URL] = b
		}
		return m
	}
	got := byURL()
	if len(got) != 4 {
		t.Fatalf("bookmarks: %+v", got)
	}
	if b := got["https://photos.example.lan"]; b.Name != "Immich" || b.Mac != nas.Mac || b.Port != 2283 || b.Source != 3 {
		t.Errorf("photos: %+v", b)
	}
	if b := got["https://ha.example.lan"]; b.Name != "Ha" || b.Mac != docker.Mac || b.Port != 18123 {
		t.Errorf("ha (container upstream): %+v", b)
	}
	if b := got["https://router.example.lan"]; b.Name != "Router" || b.Mac != "" {
		t.Errorf("router (unknown host): %+v", b)
	}
	if b := got["https://wiki.example.lan"]; b.ID != manual.ID || b.Source != 0 {
		t.Errorf("a hand-made bookmark for the same address should be left alone: %+v", b)
	}

	// the user renames one, tags it, and deletes another
	ha := got["https://ha.example.lan"]
	ha.Name = "Home Assistant"
	if err := gdb.SaveBookmark(&ha, []string{"Home"}); err != nil {
		t.Fatal(err)
	}
	gdb.DeleteBookmark(got["https://router.example.lan"].ID)

	// photos leaves Caddy
	if _, err := applySites(caddy, sites[1:], byIP); err != nil {
		t.Fatal(err)
	}
	got = byURL()
	if _, ok := got["https://photos.example.lan"]; ok {
		t.Errorf("a site Caddy no longer serves should go")
	}
	if _, ok := got["https://router.example.lan"]; ok {
		t.Errorf("a deleted site came back")
	}
	if b := got["https://ha.example.lan"]; b.Name != "Home Assistant" || b.ID != ha.ID {
		t.Errorf("rename lost: %+v", b)
	}

	gdb.DeleteConnector(3)
	if got = byURL(); len(got) != 1 {
		t.Errorf("deleting the connector should take its bookmarks: %+v", got)
	}
}
