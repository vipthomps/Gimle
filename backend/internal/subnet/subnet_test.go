package subnet

import (
	"testing"

	"github.com/vipthomps/gimle/backend/internal/models"
)

func TestValidate(t *testing.T) {
	s := models.Subnet{CIDR: "10.1.2.77/24", Reserved: "10.1.2.1, 10.1.2.100-10.1.2.150"}
	if err := Validate(&s); err != nil {
		t.Fatal(err)
	}
	if s.CIDR != "10.1.2.0/24" || s.Name != "10.1.2.0/24" || s.Method != "arp" {
		t.Errorf("not normalized: %+v", s)
	}

	bad := []models.Subnet{
		{CIDR: "nope"},
		{CIDR: "10.0.0.0/8"},
		{CIDR: "fd00::/64"},
		{CIDR: "10.1.2.0/24", Method: "carrier-pigeon"},
		{CIDR: "10.1.2.0/24", Reserved: "10.9.9.9"},
		{CIDR: "10.1.2.0/24", Reserved: "10.1.2.20-10.1.2.10"},
		{CIDR: "10.1.2.0/24", Reserved: "garbage"},
	}
	for _, b := range bad {
		if err := Validate(&b); err == nil {
			t.Errorf("expected error for %+v", b)
		}
	}
}

func TestMap(t *testing.T) {
	s := models.Subnet{CIDR: "192.168.5.0/28", Reserved: "192.168.5.1, 192.168.5.2-192.168.5.3"}
	hosts := []models.Host{
		{ID: 1, IP: "192.168.5.2", Now: 1, Name: "router"}, // reserved and in use
		{ID: 2, IP: "192.168.5.5", Now: 0},
		{ID: 3, IP: "192.168.5.5", Now: 1, Name: "nas"}, // online wins
		{ID: 4, IP: "192.168.5.9", Now: 0},
		{ID: 5, IP: "10.0.0.1", Now: 1}, // other subnet
	}

	ipam, err := Map(s, hosts)
	if err != nil {
		t.Fatal(err)
	}
	st := ipam.Stat
	// 16 addresses minus network and broadcast
	if st.Total != 14 || st.Online != 2 || st.Offline != 1 || st.Reserved != 2 || st.Free != 9 {
		t.Errorf("wrong stats: %+v", st)
	}
	if st.NextFree != "192.168.5.4" {
		t.Errorf("next free = %s", st.NextFree)
	}
	if st.Utilization != 35 {
		t.Errorf("utilization = %d", st.Utilization)
	}
	if len(ipam.Addresses) != 16 {
		t.Fatalf("addresses = %d", len(ipam.Addresses))
	}
	want := map[int]string{0: "network", 1: "reserved", 2: "online", 4: "free", 5: "online", 9: "offline", 15: "broadcast"}
	for i, state := range want {
		if ipam.Addresses[i].State != state {
			t.Errorf("address %d: got %s want %s", i, ipam.Addresses[i].State, state)
		}
	}
	if !ipam.Addresses[2].Reserved || ipam.Addresses[5].Name != "nas" {
		t.Errorf("host details missing: %+v %+v", ipam.Addresses[2], ipam.Addresses[5])
	}
}

func TestMapLarge(t *testing.T) {
	ipam, err := Map(models.Subnet{CIDR: "10.20.0.0/16"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ipam.Addresses != nil || ipam.Stat.Total != 65534 || ipam.Stat.NextFree != "10.20.0.1" {
		t.Errorf("large subnet: %d addresses, %+v", len(ipam.Addresses), ipam.Stat)
	}
}

func TestMapPointToPoint(t *testing.T) {
	ipam, err := Map(models.Subnet{CIDR: "10.0.0.0/31"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ipam.Stat.Total != 2 || ipam.Addresses[0].State != "free" {
		t.Errorf("/31: %+v", ipam)
	}
}
