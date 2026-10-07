// Package subnet parses subnets and works out which addresses are used, reserved or free.
package subnet

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// MinPrefix - largest subnet accepted (/16)
const MinPrefix = 16

// GridLimit - largest subnet whose addresses are listed one by one
const GridLimit = 4096

// Methods - accepted scan methods
var Methods = []string{"arp", "none"}

// Parse - check a CIDR and return its network
func Parse(cidr string) (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q", cidr)
	}
	if ipNet.IP.To4() == nil {
		return nil, fmt.Errorf("only IPv4 subnets are supported")
	}
	ones, _ := ipNet.Mask.Size()
	if ones < MinPrefix {
		return nil, fmt.Errorf("subnet is larger than /%d", MinPrefix)
	}
	return ipNet, nil
}

// Validate - check a subnet before it is saved, and normalize its CIDR
func Validate(s *models.Subnet) error {
	ipNet, err := Parse(s.CIDR)
	if err != nil {
		return err
	}
	s.CIDR = ipNet.String()
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		s.Name = s.CIDR
	}
	s.Iface = strings.TrimSpace(s.Iface)
	if s.Method == "" {
		s.Method = "arp"
	}
	if !validMethod(s.Method) {
		return fmt.Errorf("unknown scan method %q", s.Method)
	}
	if _, err := ParseReserved(s.Reserved, ipNet); err != nil {
		return err
	}
	return nil
}

func validMethod(m string) bool {
	for _, v := range Methods {
		if v == m {
			return true
		}
	}
	return false
}

// ParseReserved - parse "10.0.0.1, 10.0.0.100-10.0.0.200" into a set of addresses inside ipNet
func ParseReserved(str string, ipNet *net.IPNet) (map[uint32]bool, error) {
	set := make(map[uint32]bool)

	for _, part := range strings.Split(str, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		first, last, found := strings.Cut(part, "-")
		if !found {
			last = first
		}
		a, err := toUint(first, ipNet)
		if err != nil {
			return nil, err
		}
		b, err := toUint(last, ipNet)
		if err != nil {
			return nil, err
		}
		if a > b {
			return nil, fmt.Errorf("reserved range %q is backwards", part)
		}
		for i := a; i <= b; i++ {
			set[i] = true
			if i == b { // avoid overflow at 255.255.255.255
				break
			}
		}
	}
	return set, nil
}

func toUint(str string, ipNet *net.IPNet) (uint32, error) {
	ip := net.ParseIP(strings.TrimSpace(str)).To4()
	if ip == nil {
		return 0, fmt.Errorf("invalid reserved address %q", str)
	}
	if !ipNet.Contains(ip) {
		return 0, fmt.Errorf("reserved address %s is outside %s", ip, ipNet)
	}
	return binary.BigEndian.Uint32(ip), nil
}

func toIP(n uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip.String()
}

// Map - work out the state of every address in a subnet from the known hosts
func Map(s models.Subnet, hosts []models.Host) (ipam models.IPAM, err error) {
	ipNet, err := Parse(s.CIDR)
	if err != nil {
		return ipam, err
	}
	reserved, err := ParseReserved(s.Reserved, ipNet)
	if err != nil {
		return ipam, err
	}

	byIP := make(map[uint32]models.Host)
	for _, h := range hosts {
		ip := net.ParseIP(h.IP).To4()
		if ip == nil || !ipNet.Contains(ip) {
			continue
		}
		n := binary.BigEndian.Uint32(ip)
		// An online host wins over an old offline one with the same IP
		if prev, ok := byIP[n]; !ok || (prev.Now == 0 && h.Now == 1) {
			byIP[n] = h
		}
	}

	ones, bits := ipNet.Mask.Size()
	size := uint32(1) << uint(bits-ones)
	first := binary.BigEndian.Uint32(ipNet.IP.To4())
	last := first + size - 1
	// /31 and /32 have no network or broadcast address
	hasEdges := size > 2
	listAll := size <= GridLimit

	stat := models.SubnetStat{Subnet: s}

	for n := first; ; n++ {
		addr := models.Address{IP: toIP(n), Reserved: reserved[n]}

		switch {
		case hasEdges && n == first:
			addr.State = "network"
		case hasEdges && n == last:
			addr.State = "broadcast"
		default:
			stat.Total++
			if h, ok := byIP[n]; ok {
				addr.HostID, addr.Name, addr.Mac = h.ID, h.Name, h.Mac
				if h.Now == 1 {
					addr.State = "online"
					stat.Online++
				} else {
					addr.State = "offline"
					stat.Offline++
				}
			} else if addr.Reserved {
				addr.State = "reserved"
				stat.Reserved++
			} else {
				addr.State = "free"
				stat.Free++
				if stat.NextFree == "" {
					stat.NextFree = addr.IP
				}
			}
		}

		if listAll {
			ipam.Addresses = append(ipam.Addresses, addr)
		}
		if n == last {
			break
		}
	}

	if stat.Total > 0 {
		stat.Utilization = (stat.Total - stat.Free) * 100 / stat.Total
	}
	ipam.Stat = stat
	return ipam, nil
}

// Detect - suggest subnets from the IPv4 addresses on local interfaces.
// If ifaces (space separated) is set, only those interfaces are used.
func Detect(ifaces string) (found []models.Subnet) {
	wanted := strings.Fields(ifaces)

	list, err := net.Interfaces()
	if err != nil {
		return found
	}
	seen := make(map[string]bool)

	for _, iface := range list {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		if len(wanted) > 0 && !contains(wanted, iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLinkLocalUnicast() {
				continue
			}
			ones, _ := ipNet.Mask.Size()
			if ones < MinPrefix || ones > 30 {
				continue
			}
			network := &net.IPNet{IP: ipNet.IP.Mask(ipNet.Mask), Mask: ipNet.Mask}
			cidr := network.String()
			if seen[cidr] {
				continue
			}
			seen[cidr] = true
			found = append(found, models.Subnet{
				Name:   iface.Name + " " + cidr,
				CIDR:   cidr,
				Iface:  iface.Name,
				Method: "arp",
			})
		}
	}
	return found
}

// IfaceFor - find the local interface that is on the subnet, for ARP scans
func IfaceFor(cidr string) string {
	ipNet, err := Parse(cidr)
	if err != nil {
		return ""
	}
	list, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range list {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ip, ok := a.(*net.IPNet); ok && ipNet.Contains(ip.IP) {
				return iface.Name
			}
		}
	}
	return ""
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
