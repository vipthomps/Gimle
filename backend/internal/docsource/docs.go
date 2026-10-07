package docsource

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// NavItem - one entry in the docs menu: a page, or a section of entries
type NavItem struct {
	Title    string
	Path     string    `json:",omitempty"`
	Children []NavItem `json:",omitempty"`
}

// Page - one rendered page
type Page struct {
	Path     string
	Title    string
	HTML     string
	Markdown string
	Sha      string // blob being shown; sent back on save to catch edits made elsewhere
	GitHub   string // the file on GitHub
}

// mkdocsKey - where files() keeps an mkdocs.yml from the folder above the docs
const mkdocsKey = "../mkdocs.yml"

// Clean - a page path from the browser, inside the docs folder, or ok false
func Clean(p string) (string, bool) {
	p = strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(p)), "/")
	if p == "" || p == "." || strings.Contains(p, "\x00") {
		return "", false
	}
	return p, true
}

// Nav - the docs menu. Follows the nav in an mkdocs.yml next to the docs
// folder when there is one; pages it leaves out are listed after it.
func (s Source) Nav() ([]NavItem, error) {
	entries, err := s.files()
	if err != nil {
		return nil, err
	}
	pages := map[string]bool{}
	for p := range entries {
		if strings.HasSuffix(p, ".md") && !strings.HasPrefix(p, "../") {
			pages[p] = true
		}
	}

	var nav []NavItem
	if raw, _, ok, err := s.read(mkdocsKey); err == nil && ok {
		nav = mkdocsNav(raw, pages)
	}

	listed := map[string]bool{}
	var mark func([]NavItem)
	mark = func(items []NavItem) {
		for _, it := range items {
			listed[it.Path] = true
			mark(it.Children)
		}
	}
	mark(nav)

	var rest []string
	for p := range pages {
		if !listed[p] {
			rest = append(rest, p)
		}
	}
	if len(nav) == 0 {
		return folderNav(rest), nil
	}
	if len(rest) > 0 {
		nav = append(nav, NavItem{Title: "Not in the menu", Children: folderNav(rest)})
	}
	return nav, nil
}

// mkdocsNav - the nav from mkdocs.yml, keeping only pages that exist
func mkdocsNav(raw []byte, pages map[string]bool) []NavItem {
	var conf struct {
		Nav []any `yaml:"nav"`
	}
	if yaml.Unmarshal(raw, &conf) != nil {
		return nil
	}
	return navItems(conf.Nav, pages)
}

func navItems(list []any, pages map[string]bool) []NavItem {
	var out []NavItem
	for _, v := range list {
		switch it := v.(type) {
		case string:
			if pages[it] {
				out = append(out, NavItem{Title: pretty(it), Path: it})
			}
		case map[string]any:
			for title, val := range it {
				switch x := val.(type) {
				case string:
					if pages[x] {
						out = append(out, NavItem{Title: title, Path: x})
					}
				case []any:
					if kids := navItems(x, pages); len(kids) > 0 {
						out = append(out, NavItem{Title: title, Children: kids})
					}
				}
			}
		}
	}
	return out
}

// folderNav - pages grouped by folder, index pages first
func folderNav(paths []string) []NavItem {
	sort.Slice(paths, func(i, j int) bool {
		di, dj := path.Dir(paths[i]), path.Dir(paths[j])
		if di != dj {
			if di == "." || dj == "." {
				return di == "."
			}
			return di < dj
		}
		bi, bj := path.Base(paths[i]) == "index.md", path.Base(paths[j]) == "index.md"
		if bi != bj {
			return bi
		}
		return paths[i] < paths[j]
	})

	var out []NavItem
	sections := map[string]int{}
	for _, p := range paths {
		item := NavItem{Title: pretty(p), Path: p}
		dir := path.Dir(p)
		if dir == "." {
			out = append(out, item)
			continue
		}
		i, ok := sections[dir]
		if !ok {
			out = append(out, NavItem{Title: pretty(dir)})
			i = len(out) - 1
			sections[dir] = i
		}
		out[i].Children = append(out[i].Children, item)
	}
	return out
}

// pretty - "runbooks/zfs-scrub.md" reads as "Zfs scrub"
func pretty(p string) string {
	base := strings.TrimSuffix(path.Base(p), ".md")
	if base == "index" {
		if d := path.Dir(p); d != "." {
			base = path.Base(d)
		} else {
			return "Home"
		}
	}
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	if base == "" {
		return p
	}
	return strings.ToUpper(base[:1]) + base[1:]
}

// Page - one page, rendered
func (s Source) Page(rel string) (Page, error) {
	if !strings.HasSuffix(rel, ".md") {
		return Page{}, fmt.Errorf("only Markdown pages can be shown")
	}
	raw, sha, ok, err := s.read(rel)
	if err != nil {
		return Page{}, err
	}
	if !ok {
		return Page{}, fmt.Errorf("there is no page %s", rel)
	}
	html, title := Render(rel, raw)
	if title == "" {
		title = pretty(rel)
	}
	return Page{Path: rel, Title: title, HTML: html, Markdown: string(raw), Sha: sha, GitHub: s.webURL(rel)}, nil
}

// Save - commit a page. sha "" creates it.
func (s Source) Save(rel, markdown, sha, message string) (string, error) {
	if !strings.HasSuffix(rel, ".md") {
		return "", fmt.Errorf("a page must end in .md")
	}
	if s.Token == "" {
		return "", fmt.Errorf("saving needs a GitHub token that may write to %s", s.Repo)
	}
	if message = strings.TrimSpace(message); message == "" {
		message = "Update " + path.Join(s.Dir, rel)
		if sha == "" {
			message = "Add " + path.Join(s.Dir, rel)
		}
	}
	if !strings.HasSuffix(markdown, "\n") {
		markdown += "\n"
	}
	return s.write(rel, []byte(markdown), sha, message)
}

// File - an image or other file from the docs folder, for pages that show one
func (s Source) File(rel string) ([]byte, bool, error) {
	data, _, ok, err := s.read(rel)
	return data, ok, err
}
