package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// serve - a fake API answering fixed JSON per path; it records the auth header
func serve(t *testing.T, answers map[string]string, auth *string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth != nil {
			*auth = r.Header.Get("Authorization") + r.Header.Get("X-API-KEY")
		}
		body, ok := answers[r.URL.RequestURI()]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

func TestDocker(t *testing.T) {
	var auth string
	s := serve(t, map[string]string{"/containers/json?all=1": `[
		{"Id":"abc","Names":["/immich_server"],"Image":"ghcr.io/immich-app/immich-server:release","State":"running","Status":"Up 2 hours",
		 "Ports":[{"IP":"0.0.0.0","PrivatePort":2283,"PublicPort":2283,"Type":"tcp"},{"IP":"::","PrivatePort":2283,"PublicPort":2283,"Type":"tcp"},{"PrivatePort":5432,"Type":"tcp"}],
		 "Labels":{"com.docker.compose.project":"immich"}},
		{"Id":"def","Names":["/old"],"Image":"busybox","State":"exited","Status":"Exited (0)","Ports":[],"Labels":{}}
	]`}, &auth)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "docker", URL: s.URL, HostIP: "10.0.0.7", Token: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer t1" {
		t.Errorf("auth header %q", auth)
	}
	if len(snap.Hosts) != 1 || snap.Hosts[0].IP != "10.0.0.7" || len(snap.Hosts[0].Containers) != 2 {
		t.Fatalf("snapshot %+v", snap)
	}
	ct := snap.Hosts[0].Containers[0]
	if ct.Name != "immich_server" || ct.Project != "immich" || ct.State != "running" {
		t.Errorf("container %+v", ct)
	}
	if len(ct.Ports) != 2 || ct.Ports[0].Public != 0 || ct.Ports[1].Public != 2283 {
		t.Errorf("ports %+v", ct.Ports) // IPv4 and IPv6 entries merged, unpublished first
	}
}

func TestDockhand(t *testing.T) {
	s := serve(t, map[string]string{
		"/api/environments": `[
			{"id":1,"name":"local","connectionType":"socket","host":"","publicIp":null},
			{"id":2,"name":"notebook","connectionType":"direct","host":"10.0.0.9","publicIp":null},
			{"id":3,"name":"edge","connectionType":"hawser-edge","host":"","publicIp":null}]`,
		"/api/containers?env=1": `[{"id":"a","name":"dockhand","image":"fnsys/dockhand","state":"running","status":"Up","ports":[{"IP":"0.0.0.0","PrivatePort":3000,"PublicPort":3000,"Type":"tcp"}],"labels":{}}]`,
		"/api/containers?env=2": `[{"id":"b","name":"jellyfin","image":"jellyfin/jellyfin","state":"running","status":"Up","ports":[],"labels":{"com.docker.compose.project":"media"}}]`,
		"/api/containers?env=3": `[]`,
	}, nil)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "dockhand", URL: s.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Hosts) != 2 {
		t.Fatalf("want 2 hosts (edge skipped), got %+v", snap.Hosts)
	}
	if snap.Hosts[0].IP != "127.0.0.1" || snap.Hosts[0].Containers[0].Host != "local" {
		t.Errorf("socket environment %+v", snap.Hosts[0])
	}
	if snap.Hosts[1].IP != "10.0.0.9" || snap.Hosts[1].Containers[0].Project != "media" {
		t.Errorf("direct environment %+v", snap.Hosts[1])
	}

	if _, err := Fetch(context.Background(), models.Connector{Kind: "dockhand", URL: s.URL, Site: "nope"}); err == nil {
		t.Error("unknown environment should fail")
	}
}

func TestScanopy(t *testing.T) {
	s := serve(t, map[string]string{"/api/v1/hosts?limit=0": `{"success":true,"error":null,"data":[{
		"id":"h1","name":"notebook","display_name":"","hostname":"notebook.example.lan",
		"ip_addresses":[{"id":"ip1","ip_address":"10.0.0.9","mac_address":"AA-BB-CC-00-00-09"}],
		"ports":[{"id":"p1","number":2283,"protocol":"Tcp"},{"id":"p2","number":22,"protocol":"Tcp"}],
		"services":[
			{"name":"Immich","service_definition":"Immich","bindings":[{"type":"Port","port_id":"p1","ip_address_id":null}],
			 "virtualization_metadata":{"type":"Docker","details":{"container_name":"immich_server","container_id":"abc","compose_project":"immich"}}},
			{"name":"","service_definition":"SSH","bindings":[{"type":"Port","port_id":"p2","ip_address_id":"ip1"}]}
		]}]}`}, nil)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "scanopy", URL: s.URL, Token: "scp_u_x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Hosts) != 1 {
		t.Fatalf("hosts %+v", snap.Hosts)
	}
	h := snap.Hosts[0]
	if h.Mac != "aa:bb:cc:00:00:09" || h.Name != "notebook" || h.DNS != "notebook.example.lan" {
		t.Errorf("host %+v", h)
	}
	if len(h.Services) != 2 || h.Services[0].Name != "Immich" || h.Services[0].Category != "Media" || h.Services[1].Name != "SSH" {
		t.Errorf("services %+v", h.Services)
	}
	if len(h.Containers) != 1 || h.Containers[0].Name != "immich_server" || h.Containers[0].Ports[0].Public != 2283 {
		t.Errorf("containers %+v", h.Containers)
	}
}

func TestCheck(t *testing.T) {
	if err := Check(models.Connector{Kind: "docker", URL: "http://10.0.0.1:2375"}); err != nil {
		t.Error(err)
	}
	for _, c := range []models.Connector{
		{Kind: "nope", URL: "http://x"},
		{Kind: "docker", URL: "10.0.0.1:2375"},
		{Kind: "docker", URL: "http://10.0.0.1", HostIP: "notebook"},
	} {
		if Check(c) == nil {
			t.Errorf("%+v should fail", c)
		}
	}
}

func TestNormMac(t *testing.T) {
	for in, want := range map[string]string{"AA-BB-CC-00-00-09": "aa:bb:cc:00:00:09", "aabbcc000009": "aa:bb:cc:00:00:09", " aa:bb:cc:00:00:09": "aa:bb:cc:00:00:09"} {
		if got := NormMac(in); got != want {
			t.Errorf("NormMac(%q) = %q", in, got)
		}
	}
}

func TestUniFiClassic(t *testing.T) {
	var auth string
	s := serve(t, map[string]string{
		"/proxy/network/api/s/default/rest/user": `{"meta":{"rc":"ok"},"data":[
			{"mac":"AA:BB:CC:00:00:01","name":"Office printer","last_ip":"10.0.0.50"},
			{"mac":"aa:bb:cc:00:00:02","hostname":"nas","use_fixedip":true,"fixed_ip":"10.0.0.10"}]}`,
		"/proxy/network/api/s/default/stat/sta": `{"meta":{"rc":"ok"},"data":[
			{"mac":"aa:bb:cc:00:00:01","ip":"10.0.0.51","hostname":"BRN123"}]}`,
	}, &auth)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "unifi", URL: s.URL, Token: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "k" {
		t.Errorf("X-API-KEY %q", auth)
	}
	if len(snap.Hosts) != 2 {
		t.Fatalf("hosts %+v", snap.Hosts)
	}
	if h := snap.Hosts[0]; h.Mac != "aa:bb:cc:00:00:01" || h.IP != "10.0.0.51" || h.Name != "BRN123" {
		t.Errorf("active client %+v", h) // current IP wins; the alias comes from whichever list had a name first
	}
	if h := snap.Hosts[1]; h.IP != "10.0.0.10" || h.Name != "nas" {
		t.Errorf("known client %+v", h)
	}
}

func TestUniFiIntegrationFallback(t *testing.T) {
	s := serve(t, map[string]string{
		"/proxy/network/integration/v1/sites?limit=200":                     `{"offset":0,"count":1,"totalCount":1,"data":[{"id":"s1","name":"Default","internalReference":"default"}]}`,
		"/proxy/network/integration/v1/sites/s1/clients?offset=0&limit=200": `{"offset":0,"count":1,"totalCount":1,"data":[{"name":"notebook","ipAddress":"10.0.0.9","macAddress":"aa:bb:cc:00:00:09"}]}`,
	}, nil)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "unifi", URL: s.URL, Token: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Hosts) != 1 || snap.Hosts[0].Name != "notebook" || snap.Hosts[0].IP != "10.0.0.9" {
		t.Errorf("hosts %+v", snap.Hosts)
	}
}

func TestTechnitium(t *testing.T) {
	var auth string
	s := serve(t, map[string]string{
		"/api/zones/list": `{"status":"ok","response":{"zones":[
			{"name":"example.lan","type":"Primary"},{"name":"1.0.10.in-addr.arpa","type":"Primary"},{"name":"example.com","type":"Forwarder"}]}}`,
		"/api/zones/records/get?domain=example.lan&listZone=true&zone=example.lan": `{"status":"ok","response":{"records":[
			{"name":"notebook.example.lan","type":"A","rData":{"ipAddress":"10.0.0.9"}},
			{"name":"immich.example.lan","type":"A","rData":{"ipAddress":"10.0.0.9"}},
			{"name":"dns3.example.lan","type":"A","rData":{"ipAddress":"10.0.0.53"}},
			{"name":"example.lan","type":"SOA","rData":{}}]}}`,
		"/api/dhcp/leases/list": `{"status":"ok","response":{"leases":[
			{"scope":"LAN","type":"Dynamic","hardwareAddress":"AA-BB-CC-00-00-09","address":"10.0.0.9","hostName":"notebook"},
			{"scope":"LAN","type":"Dynamic","hardwareAddress":"AA-BB-CC-00-00-20","address":"10.0.0.120","hostName":null}]}}`,
	}, &auth)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "technitium", URL: s.URL, Token: "tk"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer tk" {
		t.Errorf("auth %q", auth)
	}
	if len(snap.Hosts) != 3 {
		t.Fatalf("hosts %+v", snap.Hosts)
	}
	if h := snap.Hosts[0]; h.Mac != "aa:bb:cc:00:00:09" || h.Name != "notebook" || h.DNS != "immich.example.lan notebook.example.lan" {
		t.Errorf("lease with records %+v", h)
	}
	if h := snap.Hosts[2]; h.IP != "10.0.0.53" || h.DNS != "dns3.example.lan" || h.Mac != "" {
		t.Errorf("static host %+v", h)
	}

	bad := serve(t, map[string]string{"/api/zones/list": `{"status":"invalid-token","errorMessage":"Invalid token or session expired."}`}, nil)
	if _, err := Fetch(context.Background(), models.Connector{Kind: "technitium", URL: bad.URL, Token: "x"}); err == nil {
		t.Error("an invalid token should fail")
	}
}

func TestProxmox(t *testing.T) {
	var auth string
	s := serve(t, map[string]string{
		"/api2/json/cluster/resources?type=vm": `{"data":[
			{"id":"lxc/112","type":"lxc","node":"pve1","vmid":112,"name":"scanopy","status":"running","template":0,"tags":"net;scan","maxcpu":2,"maxmem":2147483648,"uptime":3600},
			{"id":"qemu/400","type":"qemu","node":"pve1","vmid":400,"name":"notebook","status":"running","template":0,"maxcpu":8,"maxmem":17179869184},
			{"id":"qemu/8010","type":"qemu","node":"pve1","vmid":8010,"name":"ubuntu-tmpl","status":"stopped","template":1},
			{"id":"qemu/201","type":"qemu","node":"pve1","vmid":201,"name":"osx","status":"stopped","template":0}]}`,
		"/api2/json/cluster/status":                `{"data":[{"type":"cluster","name":"lab"},{"type":"node","name":"pve1","ip":"10.0.0.2"}]}`,
		"/api2/json/nodes/pve1/lxc/112/config":     `{"data":{"hostname":"scanopy","net0":"name=eth0,bridge=vmbr0,hwaddr=BC:24:11:00:01:12,ip=dhcp,type=veth"}}`,
		"/api2/json/nodes/pve1/lxc/112/interfaces": `{"data":[{"name":"lo","hwaddr":"00:00:00:00:00:00","inet":"127.0.0.1/8"},{"name":"eth0","hwaddr":"bc:24:11:00:01:12","inet":"10.0.0.112/24"}]}`,
		"/api2/json/nodes/pve1/qemu/400/config":    `{"data":{"net1":"virtio=BC:24:11:00:04:01,bridge=vmbr1","net0":"virtio=BC:24:11:00:04:00,bridge=vmbr0,firewall=1"}}`,
		"/api2/json/nodes/pve1/qemu/201/config":    `{"data":{"net0":"model=vmxnet3,macaddr=bc:24:11:00:02:01,bridge=vmbr0"}}`,
	}, &auth)

	snap, err := Fetch(context.Background(), models.Connector{Kind: "proxmox", URL: s.URL, Token: "root@pam!gimle=abc-123"})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "PVEAPIToken=root@pam!gimle=abc-123" {
		t.Errorf("auth header %q", auth)
	}
	if len(snap.Hosts) != 4 {
		t.Fatalf("want the node and 3 guests (template skipped), got %+v", snap.Hosts)
	}
	node := snap.Hosts[0]
	if node.IP != "10.0.0.2" || len(node.Guests) != 3 {
		t.Fatalf("node %+v", node)
	}
	lxc := node.Guests[0]
	if lxc.Type != "lxc" || lxc.Mac != "bc:24:11:00:01:12" || lxc.IP != "10.0.0.112" || lxc.CPUs != 2 || lxc.Tags != "net;scan" {
		t.Errorf("lxc %+v", lxc)
	}
	if vm := node.Guests[1]; vm.Mac != "bc:24:11:00:04:00" || vm.IP != "" {
		t.Errorf("vm takes net0, not net1: %+v", vm)
	}
	if vm := node.Guests[2]; vm.Mac != "bc:24:11:00:02:01" || vm.Status != "stopped" {
		t.Errorf("macaddr= form: %+v", vm)
	}
	if g := snap.Hosts[1]; g.Mac != "bc:24:11:00:01:12" || g.Name != "scanopy" {
		t.Errorf("guest host %+v", g)
	}
}

func TestPveToken(t *testing.T) {
	for in, want := range map[string]string{
		"root@pam!gimle=abc-123":             "root@pam!gimle=abc-123",
		"PVEAPIToken=root@pam!gimle=abc-123": "root@pam!gimle=abc-123",
		"root@pam!gimle\nabc-123":            "root@pam!gimle=abc-123",
		"root@pam!gimle  abc-123":            "root@pam!gimle=abc-123",
		"abc-123":                            "",
		"root@pam!gimle":                     "",
		"root@pam=abc-123":                   "",
	} {
		got, err := pveToken(in)
		if want == "" && err == nil {
			t.Errorf("%q: want an error, got %q", in, got)
		}
		if want != "" && got != want {
			t.Errorf("%q: got %q, %v", in, got, err)
		}
	}
}
