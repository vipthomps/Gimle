package docsource

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	src := "# Big box\n\nSee [the NAS](../hosts/nas.md#disks), [setup](setup/), [top](#big-box) and [GitHub](https://github.com).\n\n" +
		"![rack](img/rack.png)\n\n<script>alert(1)</script>\n\n!!! warning \"Mind the fans\"\n    They are loud.\n\nAfter.\n"
	html, title := Render("runbooks/box.md", []byte(src))

	if title != "Big box" {
		t.Errorf("title %q", title)
	}
	for _, want := range []string{
		`href="/docs/hosts/nas.md#disks"`,
		`href="/docs/runbooks/setup/index.md"`,
		`href="#big-box"`,
		`href="https://github.com" target="_blank" rel="noopener"`,
		`src="/api/docs/file?path=runbooks%2Fimg%2Frack.png"`,
		`<h1 id="big-box">`,
		"<blockquote>\n<p><strong>Mind the fans</strong></p>\n<p>They are loud.</p>\n</blockquote>",
		"<p>After.</p>",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in\n%s", want, html)
		}
	}
	if strings.Contains(html, "<script>") {
		t.Errorf("raw HTML was rendered:\n%s", html)
	}
}

func TestRenderEscapesFolder(t *testing.T) {
	html, _ := Render("index.md", []byte("[up](../../etc/passwd.md) ![x](../x.png)"))
	if strings.Contains(html, "/docs/..") || strings.Contains(html, "/api/docs/file") {
		t.Errorf("a link left the docs folder:\n%s", html)
	}
}

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"hosts/nas.md":      "hosts/nas.md",
		"/hosts//nas.md":    "hosts/nas.md",
		"../../etc/x.md":    "etc/x.md",
		" a/../b/index.md ": "b/index.md",
		"":                  "",
		"..":                "",
	} {
		if got, _ := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMkdocsNav(t *testing.T) {
	raw := []byte("site_name: x\nnav:\n  - Home: index.md\n  - Hosts:\n      - nas: hosts/nas.md\n      - gone: hosts/gone.md\n  - about.md\n")
	nav := mkdocsNav(raw, map[string]bool{"index.md": true, "hosts/nas.md": true, "about.md": true})
	b, _ := json.Marshal(nav)
	want := `[{"Title":"Home","Path":"index.md"},{"Title":"Hosts","Children":[{"Title":"nas","Path":"hosts/nas.md"}]},{"Title":"About","Path":"about.md"}]`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

// fakeGitHub - just enough of the API: a tree, blobs and the contents PUT
func fakeGitHub(t *testing.T, files map[string]string) *httptest.Server {
	sha := func(p string) string { return fmt.Sprintf("%x", sha1.Sum([]byte(p+files[p]))) }
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/me/lab/git/trees/"):
			var tree []entry
			for p := range files {
				tree = append(tree, entry{Path: p, Type: "blob", Sha: sha(p)})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"tree": tree})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/repos/me/lab/git/blobs/"):
			for p := range files {
				if sha(p) == strings.TrimPrefix(r.URL.Path, "/repos/me/lab/git/blobs/") {
					_, _ = w.Write([]byte(files[p]))
					return
				}
			}
			w.WriteHeader(404)
		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/repos/me/lab/contents/"):
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(403)
				return
			}
			p := strings.TrimPrefix(r.URL.Path, "/repos/me/lab/contents/")
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			if _, exists := files[p]; exists && in["sha"] != sha(p) {
				w.WriteHeader(409)
				return
			}
			b, _ := base64.StdEncoding.DecodeString(in["content"])
			files[p] = string(b)
			_ = json.NewEncoder(w).Encode(map[string]any{"content": map[string]string{"sha": sha(p)}})
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestSource(t *testing.T) {
	files := map[string]string{
		"lab-docs/mkdocs.yml":         "nav:\n  - Home: index.md\n",
		"lab-docs/docs/index.md":      "# Welcome\n",
		"lab-docs/docs/hosts/nas.md":  "# NAS\n",
		"lab-docs/docs/hosts/logo.md": "no heading",
		"other/readme.md":             "# Not docs\n",
	}
	srv := fakeGitHub(t, files)
	defer srv.Close()
	Forget()
	s := Source{API: srv.URL, Repo: "me/lab", Branch: "main", Dir: "lab-docs/docs", Token: "tok"}

	nav, err := s.Nav()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(nav)
	want := `[{"Title":"Home","Path":"index.md"},{"Title":"Not in the menu","Children":[{"Title":"Hosts","Children":[{"Title":"Logo","Path":"hosts/logo.md"},{"Title":"Nas","Path":"hosts/nas.md"}]}]}]`
	if string(b) != want {
		t.Errorf("nav\n got  %s\n want %s", b, want)
	}

	page, err := s.Page("hosts/nas.md")
	if err != nil || page.Title != "NAS" {
		t.Fatalf("page %+v, %v", page, err)
	}

	if _, err := s.Save("hosts/nas.md", "# NAS\n\nedited", page.Sha, ""); err != nil {
		t.Fatal(err)
	}
	if files["lab-docs/docs/hosts/nas.md"] != "# NAS\n\nedited\n" {
		t.Errorf("saved %q", files["lab-docs/docs/hosts/nas.md"])
	}
	if _, err := s.Save("hosts/nas.md", "stale", page.Sha, ""); !errors.Is(err, ErrConflict) {
		t.Errorf("saving over a newer version: %v, want ErrConflict", err)
	}
	again, err := s.Page("hosts/nas.md")
	if err != nil || !strings.Contains(again.Markdown, "edited") {
		t.Errorf("after save %+v, %v", again, err)
	}

	ro := s
	ro.Token = "read-only"
	if _, err := ro.Save("new.md", "# New", "", ""); err == nil || !strings.Contains(err.Error(), "may read") {
		t.Errorf("read-only token: %v", err)
	}
}
