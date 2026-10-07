package api

import "testing"

func TestValidIcon(t *testing.T) {
	ok := map[string]string{
		"":                             "",
		" Immich ":                     "immich",
		"none":                         "none",
		"home-assistant.png":           "home-assistant.png",
		"https://example.com/logo.svg": "https://example.com/logo.svg",
		"/fs/public/favicon.png":       "/fs/public/favicon.png",
	}
	for in, want := range ok {
		got := in
		if err := validIcon(&got); err != nil || got != want {
			t.Errorf("validIcon(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"javascript:alert(1)", "bad icon", "//evil.example/x.svg", "ftp://x/y.png", "../x"} {
		got := in
		if err := validIcon(&got); err == nil {
			t.Errorf("validIcon(%q) accepted", in)
		}
	}
}
