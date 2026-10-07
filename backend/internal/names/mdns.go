package names

import (
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// reverseName - "192.168.1.5" -> "5.1.168.192.in-addr.arpa."
func reverseName(ip string) string {
	v4 := net.ParseIP(ip).To4()
	if v4 == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.", v4[3], v4[2], v4[1], v4[0])
}

// MDNS - ask the host itself for its name with a unicast mDNS PTR query to port 5353.
// Apple devices, Linux with Avahi, printers and many IoT devices answer.
func MDNS(ip string, timeout time.Duration) string {
	rev := reverseName(ip)
	if rev == "" {
		return ""
	}
	qname, err := dnsmessage.NewName(rev)
	if err != nil {
		return ""
	}
	msg := dnsmessage.Message{
		Header: dnsmessage.Header{ID: 0},
		Questions: []dnsmessage.Question{{
			Name:  qname,
			Type:  dnsmessage.TypePTR,
			Class: dnsmessage.ClassINET | 0x8000, // unicast response wanted
		}},
	}
	packet, err := msg.Pack()
	if err != nil {
		return ""
	}

	conn, err := net.DialTimeout("udp4", net.JoinHostPort(ip, "5353"), timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(packet); err != nil {
		return ""
	}

	buf := make([]byte, 9000)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return ""
		}
		if name := ptrAnswer(buf[:n]); name != "" {
			return name
		}
	}
}

// ptrAnswer - the first PTR target in a DNS response, without the trailing dot
func ptrAnswer(packet []byte) string {
	var p dnsmessage.Parser
	if _, err := p.Start(packet); err != nil {
		return ""
	}
	if err := p.SkipAllQuestions(); err != nil {
		return ""
	}
	for {
		h, err := p.AnswerHeader()
		if err != nil {
			return ""
		}
		if h.Type != dnsmessage.TypePTR {
			if err := p.SkipAnswer(); err != nil {
				return ""
			}
			continue
		}
		r, err := p.PTRResource()
		if err != nil {
			return ""
		}
		return strings.TrimSuffix(r.PTR.String(), ".")
	}
}
