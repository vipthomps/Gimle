package portscan

import (
	"crypto/tls"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const maxTitle = 80

var titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// Probe - check whether a port serves a web page, trying https and then http.
// Returns "https", "http" or "", and the page title if it has one.
func Probe(ip string, port int, timeout time.Duration) (web, title string) {
	for _, scheme := range []string{"https", "http"} {
		if t, ok := get(scheme+"://"+net.JoinHostPort(ip, strconv.Itoa(port))+"/", timeout); ok {
			return scheme, t
		}
	}
	return "", ""
}

func get(url string, timeout time.Duration) (title string, ok bool) {
	client := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// Home lab services mostly use self-signed certificates; only the title is read
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			DisableKeepAlives: true,
			Proxy:             nil,
		},
		// Follow redirects to a login page, but only on the same host and port
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Host != via[0].URL.Host {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return Title(body), true
}

// Title - the text of a page's <title>, tidied and shortened
func Title(page []byte) string {
	m := titleRe.FindSubmatch(page)
	if m == nil {
		return ""
	}
	t := strings.Join(strings.Fields(html.UnescapeString(string(m[1]))), " ")
	if r := []rune(t); len(r) > maxTitle {
		t = strings.TrimSpace(string(r[:maxTitle-1])) + "…"
	}
	return t
}
