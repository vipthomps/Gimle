package portscan

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTitle(t *testing.T) {
	cases := map[string]string{
		"<html><head><title>Immich</title></head></html>":           "Immich",
		"<TITLE lang=en>\n  Proxmox  Virtual\tEnvironment </TITLE>": "Proxmox Virtual Environment",
		"<title>Tom &amp; Jerry</title>":                            "Tom & Jerry",
		"<html>no title</html>":                                     "",
	}
	for in, want := range cases {
		if got := Title([]byte(in)); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
	long := Title([]byte("<title>" + strings.Repeat("a", 200) + "</title>"))
	if n := len([]rune(long)); n != maxTitle {
		t.Errorf("long title is %d runes", n)
	}
}

func hostPort(t *testing.T, url string) (string, int) {
	t.Helper()
	h, p, err := net.SplitHostPort(strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(p)
	return h, port
}

func TestProbe(t *testing.T) {
	page := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("<title>Sign in - Dockhand</title>"))
	})

	plain := httptest.NewServer(page)
	defer plain.Close()
	ip, port := hostPort(t, plain.URL)
	if web, title := Probe(ip, port, 2*time.Second); web != "http" || title != "Sign in - Dockhand" {
		t.Errorf("http: got %q %q", web, title)
	}

	secure := httptest.NewTLSServer(page)
	defer secure.Close()
	ip, port = hostPort(t, secure.URL)
	if web, title := Probe(ip, port, 2*time.Second); web != "https" || title != "Sign in - Dockhand" {
		t.Errorf("https: got %q %q", web, title)
	}

	ln, _ := net.Listen("tcp", "127.0.0.1:0") // accepts and says nothing
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	ip, port = hostPort(t, "http://"+ln.Addr().String())
	if web, _ := Probe(ip, port, time.Second); web != "" {
		t.Errorf("not a web server: got %q", web)
	}
}
