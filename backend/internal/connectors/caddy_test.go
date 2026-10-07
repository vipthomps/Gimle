package connectors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// testdata/caddy.json is what Caddy 2.10 adapts this Caddyfile to:
//
//	:2020 { read-only proxy of GET /config/ }
//	photos.example.lan { reverse_proxy 10.10.0.5:2283 }
//	*.example.lan {
//		@jelly host jellyfin.example.lan
//		handle @jelly { reverse_proxy localhost:8096 }
//		@ha host ha.example.lan
//		handle @ha { reverse_proxy homeassistant:8123 }
//		handle { respond "nothing here" 404 }
//	}
//	http://wiki.example.lan { reverse_proxy 10.10.0.7:3000 10.10.0.8:3000 }
//	files.example.lan { file_server }
func TestCaddy(t *testing.T) {
	cfg, err := os.ReadFile("testdata/caddy.json")
	if err != nil {
		t.Fatal(err)
	}
	s := serve(t, map[string]string{"/config/": string(cfg)}, nil)
	snap, err := Fetch(context.Background(), models.Connector{Kind: "caddy", URL: s.URL, HostIP: "10.10.0.2"})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]Site{
		"photos.example.lan":   {URL: "https://photos.example.lan", IP: "10.10.0.5", Port: 2283},
		"jellyfin.example.lan": {URL: "https://jellyfin.example.lan", IP: "10.10.0.2", Port: 8096}, // localhost is the Caddy host
		"ha.example.lan":       {URL: "https://ha.example.lan", Port: 8123},                        // a container name
		"wiki.example.lan":     {URL: "http://wiki.example.lan", IP: "10.10.0.7", Port: 3000},      // first upstream
	}
	got := make(map[string]Site)
	for _, s := range snap.Sites {
		got[s.Name] = s
	}
	if len(got) != len(want) {
		t.Errorf("sites: got %+v", snap.Sites)
	}
	for name, w := range want {
		g, ok := got[name]
		if !ok {
			t.Errorf("%s missing", name)
			continue
		}
		if g.URL != w.URL || (w.IP != "" && g.IP != w.IP) || g.Port != w.Port {
			t.Errorf("%s: got %+v, want %+v", name, g, w)
		}
	}
}

func TestCaddyPorts(t *testing.T) {
	cfg := caddyConfig{}
	cfg.Apps.HTTP.HTTPPort = 8080
	cfg.Apps.HTTP.Servers = map[string]caddyServer{
		"a": {Listen: []string{":8443"}, Routes: []caddyRoute{{
			Match:  []caddyMatch{{Host: []string{"a.example.lan"}}},
			Handle: []caddyHandler{{Handler: "reverse_proxy", Upstreams: []caddyDial{{Dial: "nas:5000"}}}},
		}}},
		"b": {Listen: []string{":8080"}, Routes: []caddyRoute{{
			Match:  []caddyMatch{{Host: []string{"b.example.lan"}}},
			Handle: []caddyHandler{{Handler: "reverse_proxy", Upstreams: []caddyDial{{Dial: "{env.UP}"}}}},
		}}},
	}
	lookup := func(h string) ([]string, error) {
		if h == "nas" {
			return []string{"fe80::1", "10.10.0.9"}, nil
		}
		return nil, errors.New("no such host")
	}
	sites := caddySites(cfg, "", lookup)
	got := fmt.Sprint(sites)
	if len(sites) != 2 || sites[0].URL != "https://a.example.lan:8443" || sites[0].IP != "10.10.0.9" || sites[0].Port != 5000 ||
		sites[1].URL != "http://b.example.lan:8080" || sites[1].IP != "" {
		t.Errorf("got %s", got)
	}
}
