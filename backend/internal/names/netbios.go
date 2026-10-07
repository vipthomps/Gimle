package names

import (
	"encoding/binary"
	"net"
	"strings"
	"time"
)

// nbstatRequest - a NetBIOS node status request for the wildcard name "*"
func nbstatRequest() []byte {
	p := []byte{
		0x13, 0x37, // transaction ID
		0x00, 0x00, // flags
		0x00, 0x01, // questions
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x20, // encoded name length
	}
	name := make([]byte, 16) // "*" padded with zero bytes
	name[0] = '*'
	for _, b := range name {
		p = append(p, 'A'+b>>4, 'A'+b&0x0f)
	}
	return append(p, 0x00, 0x00, 0x21, 0x00, 0x01) // end of name, NBSTAT, IN
}

// NetBIOS - the workstation name from a NetBIOS node status query (UDP 137).
// Windows machines and Samba servers answer.
func NetBIOS(ip string, timeout time.Duration) string {
	conn, err := net.DialTimeout("udp4", net.JoinHostPort(ip, "137"), timeout)
	if err != nil {
		return ""
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(nbstatRequest()); err != nil {
		return ""
	}
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return ""
	}
	return parseNBSTAT(buf[:n])
}

// parseNBSTAT - the first unique name with suffix 0x00 (the computer name)
func parseNBSTAT(b []byte) string {
	// header (12) + name (34) + type, class, TTL, data length (10)
	const start = 12 + 34 + 10
	if len(b) < start+1 || binary.BigEndian.Uint16(b[6:8]) == 0 {
		return ""
	}
	count := int(b[start])
	off := start + 1
	for i := 0; i < count && off+18 <= len(b); i++ {
		name := strings.TrimRight(string(b[off:off+15]), " \x00")
		suffix := b[off+15]
		group := b[off+16]&0x80 != 0
		off += 18
		if suffix == 0x00 && !group && name != "" {
			return name
		}
	}
	return ""
}
