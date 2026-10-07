package names

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func TestShort(t *testing.T) {
	cases := map[string]string{
		"":                           "",
		"nas.example.lan":            "nas",
		"nas.example.lan.":           "nas",
		"dns3.example.com other.lan": "dns3",
		"printer":                    "printer",
		"192.168.1.5":                "192.168.1.5",
	}
	for in, want := range cases {
		if got := Short(in); got != want {
			t.Errorf("Short(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVendor(t *testing.T) {
	cases := map[string]string{
		"(Unknown)":                               "",
		"(Unknown: locally administered)":         "Private MAC",
		"(Unknown: locally administered address)": "Private MAC",
		"Raspberry Pi Trading Ltd":                "Raspberry Pi Trading Ltd",
	}
	for in, want := range cases {
		if got := Vendor(in); got != want {
			t.Errorf("Vendor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckServer(t *testing.T) {
	for _, ok := range []string{"", "10.0.0.1", "10.0.0.1:5353", "[fd00::1]:53"} {
		if err := CheckServer(ok); err != nil {
			t.Errorf("CheckServer(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"dns.lan", "10.0.0.1:0", "10.0.0.1:x"} {
		if err := CheckServer(bad); err == nil {
			t.Errorf("CheckServer(%q) should fail", bad)
		}
	}
}

func TestReverseName(t *testing.T) {
	if got := reverseName("192.168.1.5"); got != "5.1.168.192.in-addr.arpa." {
		t.Errorf("got %q", got)
	}
	if got := reverseName("nope"); got != "" {
		t.Errorf("got %q", got)
	}
}

func ptrResponse(t *testing.T, target string) []byte {
	t.Helper()
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
	if err := b.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	name := dnsmessage.MustNewName("5.1.168.192.in-addr.arpa.")
	err := b.PTRResource(
		dnsmessage.ResourceHeader{Name: name, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.PTRResource{PTR: dnsmessage.MustNewName(target)},
	)
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestPTRAnswer(t *testing.T) {
	if got := ptrAnswer(ptrResponse(t, "macbook.local.")); got != "macbook.local" {
		t.Errorf("got %q", got)
	}
	if got := ptrAnswer([]byte{1, 2, 3}); got != "" {
		t.Errorf("garbage gave %q", got)
	}
}

// nbstatResponse - a node status answer listing the given names
func nbstatResponse(entries []struct {
	name   string
	suffix byte
	group  bool
}) []byte {
	b := make([]byte, 12+34+10)
	binary.BigEndian.PutUint16(b[6:8], 1) // one answer
	b = append(b, byte(len(entries)))
	for _, e := range entries {
		n := []byte(e.name + "               ")[:15]
		flags := byte(0x04)
		if e.group {
			flags |= 0x80
		}
		b = append(b, n...)
		b = append(b, e.suffix, flags, 0x00)
	}
	return b
}

func TestParseNBSTAT(t *testing.T) {
	resp := nbstatResponse([]struct {
		name   string
		suffix byte
		group  bool
	}{
		{"WORKGROUP", 0x00, true},
		{"GAMING-PC", 0x20, false},
		{"GAMING-PC", 0x00, false},
	})
	if got := parseNBSTAT(resp); got != "GAMING-PC" {
		t.Errorf("got %q", got)
	}
	if got := parseNBSTAT(resp[:20]); got != "" {
		t.Errorf("short packet gave %q", got)
	}
	if len(nbstatRequest()) != 12+34+4 {
		t.Errorf("request is %d bytes", len(nbstatRequest()))
	}
}

// TestReverseAgainstServer - a fake DNS server on localhost answers the PTR query
func TestReverseAgainstServer(t *testing.T) {
	pc, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Skip(err)
	}
	defer pc.Close()
	go func() {
		buf := make([]byte, 1500)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		var p dnsmessage.Parser
		h, err := p.Start(buf[:n])
		if err != nil {
			return
		}
		q, _ := p.Question()
		b := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: h.ID, Response: true, Authoritative: true})
		_ = b.StartQuestions()
		_ = b.Question(q)
		_ = b.StartAnswers()
		_ = b.PTRResource(
			dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET, TTL: 60},
			dnsmessage.PTRResource{PTR: dnsmessage.MustNewName("notebook.example.lan.")},
		)
		out, _ := b.Finish()
		_, _ = pc.WriteTo(out, addr)
	}()

	got := Reverse("192.168.1.5", pc.LocalAddr().String(), 2*time.Second)
	if got != "notebook.example.lan" {
		t.Errorf("Reverse = %q", got)
	}
}
