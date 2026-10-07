package docsource

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var pageKey = parser.NewContextKey()

// Raw HTML in a page is escaped rather than rendered (goldmark's default), so
// a docs repository can never run script inside Gimle.
var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(util.Prioritized(links{}, 100)),
	),
)

// Render - a page's HTML and its first top-level heading. rel is the page's
// path inside the docs folder, so relative links can be resolved.
func Render(rel string, src []byte) (html, title string) {
	src = admonitions(frontMatter(src))
	ctx := parser.NewContext()
	ctx.Set(pageKey, rel)

	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering && h.Level == 1 {
			title = plain(h, src)
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})

	var buf bytes.Buffer
	_ = md.Renderer().Render(&buf, src, doc)
	return buf.String(), title
}

func plain(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if t, ok := c.(*ast.Text); ok && entering {
			b.Write(t.Segment.Value(src))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// links - point links between pages at Gimle's docs pages, images at the
// image proxy, and open links to other sites in a new tab
type links struct{}

func (links) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	rel, _ := pc.Get(pageKey).(string)
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Link:
			if dest, ok := pageLink(rel, string(v.Destination)); ok {
				v.Destination = []byte(dest)
			} else if external(string(v.Destination)) {
				v.SetAttributeString("target", []byte("_blank"))
				v.SetAttributeString("rel", []byte("noopener"))
			}
		case *ast.Image:
			if dest, ok := resolve(rel, string(v.Destination)); ok {
				v.Destination = []byte("/api/docs/file?path=" + url.QueryEscape(dest))
			}
		}
		return ast.WalkContinue, nil
	})
}

func external(dest string) bool {
	return strings.HasPrefix(dest, "http://") || strings.HasPrefix(dest, "https://")
}

// resolve - a relative link's path inside the docs folder
func resolve(rel, dest string) (string, bool) {
	if dest == "" || strings.HasPrefix(dest, "#") || strings.HasPrefix(dest, "/") || strings.Contains(dest, ":") {
		return "", false
	}
	p := path.Join(path.Dir(rel), dest)
	if p == ".." || strings.HasPrefix(p, "../") {
		return "", false
	}
	return p, true
}

func pageLink(rel, dest string) (string, bool) {
	target, frag, _ := strings.Cut(dest, "#")
	if frag != "" {
		frag = "#" + frag
	}
	if strings.HasSuffix(target, "/") {
		target += "index.md"
	}
	if !strings.HasSuffix(target, ".md") {
		return "", false
	}
	p, ok := resolve(rel, target)
	if !ok {
		return "", false
	}
	return "/docs/" + p + frag, true
}

// frontMatter - drop a leading YAML block, which MkDocs reads and does not show
func frontMatter(src []byte) []byte {
	if !bytes.HasPrefix(src, []byte("---\n")) {
		return src
	}
	if end := bytes.Index(src[4:], []byte("\n---\n")); end >= 0 {
		return src[4+end+5:]
	}
	return src
}

var admonitionRe = regexp.MustCompile(`^(?:!!!|\?\?\?\+?)\s+([\w-]+)(?:\s+"(.*)")?\s*$`)

// admonitions - MkDocs's `!!! note "Title"` blocks become quotes with a bold
// title, so pages written for MkDocs still read well
func admonitions(src []byte) []byte {
	if !bytes.Contains(src, []byte("!!!")) && !bytes.Contains(src, []byte("???")) {
		return src
	}
	lines := strings.Split(string(src), "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		m := admonitionRe.FindStringSubmatch(lines[i])
		if m == nil {
			out = append(out, lines[i])
			continue
		}
		title := m[2]
		if title == "" {
			title = strings.ToUpper(m[1][:1]) + m[1][1:]
		}
		out = append(out, "> **"+title+"**", ">")
		for i+1 < len(lines) {
			next := lines[i+1]
			if strings.TrimSpace(next) != "" && !strings.HasPrefix(next, "    ") && !strings.HasPrefix(next, "\t") {
				break
			}
			i++
			next = strings.TrimPrefix(strings.TrimPrefix(next, "\t"), "    ")
			out = append(out, strings.TrimRight("> "+next, " "))
		}
		out = append(out, "")
	}
	return []byte(strings.Join(out, "\n"))
}
