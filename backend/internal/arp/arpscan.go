package arp

import (
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/names"
	"github.com/vipthomps/gimle/backend/internal/subnet"
)

var arpArgs string

func scanIface(iface string) string {
	var cmd *exec.Cmd

	if arpArgs != "" {
		cmd = exec.Command("arp-scan", "-glNx", arpArgs, "-I", iface)
	} else {
		cmd = exec.Command("arp-scan", "-glNx", "-I", iface)
	}
	out, err := cmd.Output()
	slog.Debug(cmd.String())

	if check.IfError(err) {
		return string("")
	}
	return string(out)
}

func scanCIDR(iface, cidr string) string {
	var cmd *exec.Cmd

	if arpArgs != "" {
		cmd = exec.Command("arp-scan", "-gNx", arpArgs, "-I", iface, cidr)
	} else {
		cmd = exec.Command("arp-scan", "-gNx", "-I", iface, cidr)
	}
	out, err := cmd.Output()
	slog.Debug(cmd.String())

	if check.IfError(err) {
		return string("")
	}
	return string(out)
}

func scanStr(str string) string {

	args := strings.Split(str, " ")
	cmd := exec.Command("arp-scan", args...)

	out, err := cmd.Output()
	slog.Debug(cmd.String())

	if check.IfError(err) {
		return string("")
	}
	return string(out)
}

func parseOutput(text, iface string) []models.Host {
	var foundHosts = []models.Host{}

	p := strings.Split(text, "\n")

	for _, host := range p {
		if host != "" {
			var oneHost models.Host
			p := strings.Split(host, "	")
			oneHost.Iface = iface
			oneHost.IP = p[0]
			oneHost.Mac = p[1]
			oneHost.Hw = names.Vendor(p[2])
			oneHost.Date = time.Now().Format("2006-01-02 15:04:05")
			oneHost.Now = 1
			foundHosts = append(foundHosts, oneHost)
		}
	}

	return foundHosts
}

// Scan all subnets with the arp method, or all interfaces when there are none
func Scan(ifaces, args string, strs []string, subnets []models.Subnet) []models.Host {
	var text string
	var p []string
	var foundHosts = []models.Host{}
	arpArgs = args

	scanned := 0
	for _, s := range subnets {
		if s.Method != "arp" {
			continue
		}
		iface := s.Iface
		if iface == "" {
			iface = subnet.IfaceFor(s.CIDR)
		}
		if iface == "" {
			slog.Warn("No local interface on subnet, skipping ARP scan", "subnet", s.CIDR)
			continue
		}
		slog.Debug("Scanning subnet " + s.CIDR + " on " + iface)
		text = scanCIDR(iface, s.CIDR)
		slog.Debug("Found IPs: \n" + text)

		foundHosts = append(foundHosts, parseOutput(text, iface)...)
		scanned++
	}

	if scanned == 0 && ifaces != "" {

		p = strings.Split(ifaces, " ")

		for _, iface := range p {
			slog.Debug("Scanning interface " + iface)
			text = scanIface(iface)
			slog.Debug("Found IPs: \n" + text)

			foundHosts = append(foundHosts, parseOutput(text, iface)...)
		}
	}

	for _, s := range strs {
		slog.Debug("Scanning string " + s)
		text = scanStr(s)
		slog.Debug("Found IPs: \n" + text)
		p = strings.Split(s, " ")

		foundHosts = append(foundHosts, parseOutput(text, p[len(p)-1])...)
	}

	return foundHosts
}
