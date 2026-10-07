package api

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/docsource"
)

func docsSource() docsource.Source {
	c := conf.AppConfig
	return docsource.Source{API: c.DocsAPI, Repo: c.DocsRepo, Branch: c.DocsBranch, Dir: c.DocsDir, Token: c.DocsToken}
}

func docsCanEdit() bool {
	return conf.AppConfig.DocsEdit && conf.AppConfig.DocsToken != ""
}

type docsIndex struct {
	Repo    string
	Branch  string
	Dir     string
	CanEdit bool
	Nav     []docsource.NavItem
	Error   string `json:",omitempty"`
}

// getDocs godoc
// @Summary      The docs menu
// @Description  Pages in the docs folder of the configured GitHub repository, following its mkdocs.yml nav when there is one. Repo is "" until docs are set up.
// @Tags         docs
// @Produce      json
// @Success      200  {object}  docsIndex
// @Router       /docs [get]
func getDocs(c *gin.Context) {
	out := docsIndex{Repo: conf.AppConfig.DocsRepo, Branch: conf.AppConfig.DocsBranch, Dir: conf.AppConfig.DocsDir, CanEdit: docsCanEdit()}
	if out.Repo != "" {
		nav, err := docsSource().Nav()
		if err != nil {
			out.Error = err.Error()
		}
		out.Nav = nav
	}
	c.IndentedJSON(http.StatusOK, out)
}

// getDocPage godoc
// @Summary      One docs page
// @Description  Rendered HTML plus the Markdown and blob sha an edit starts from
// @Tags         docs
// @Produce      json
// @Param        path  query     string  true  "Page path inside the docs folder, e.g. hosts/nas.md"
// @Success      200   {object}  docsource.Page
// @Router       /docs/page [get]
func getDocPage(c *gin.Context) {
	rel, ok := docsource.Clean(c.Query("path"))
	if !ok || conf.AppConfig.DocsRepo == "" {
		badRequest(c, errors.New("no such page"))
		return
	}
	page, err := docsSource().Page(rel)
	if err != nil {
		c.IndentedJSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, page)
}

type docSave struct {
	Path     string
	Markdown string
	Sha      string // the blob the edit started from; "" creates a new page
	Message  string // commit message; one is made up when empty
}

// saveDocPage godoc
// @Summary      Commit a docs page to GitHub
// @Description  Needs editing turned on and a token that may write to the repository. Answers 409 when the page changed on GitHub since Sha.
// @Tags         docs
// @Accept       json
// @Produce      json
// @Param        page  body      docSave  true  "Page"
// @Success      200   {object}  docsource.Page
// @Router       /docs/page [post]
func saveDocPage(c *gin.Context) {
	if !docsCanEdit() {
		c.IndentedJSON(http.StatusForbidden, gin.H{"error": "editing is off; turn it on in Settings, with a token that may write to the repository"})
		return
	}
	var in docSave
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	rel, ok := docsource.Clean(in.Path)
	if !ok || !strings.HasSuffix(rel, ".md") {
		badRequest(c, errors.New("a page path ends in .md, e.g. runbooks/backups.md"))
		return
	}
	src := docsSource()
	sha, err := src.Save(rel, in.Markdown, in.Sha, in.Message)
	switch {
	case errors.Is(err, docsource.ErrConflict):
		c.IndentedJSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	case err != nil:
		c.IndentedJSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	html, title := docsource.Render(rel, []byte(in.Markdown))
	c.IndentedJSON(http.StatusOK, docsource.Page{Path: rel, Title: title, HTML: html, Markdown: in.Markdown, Sha: sha})
}

// previewDoc godoc
// @Summary      Render Markdown without saving it
// @Tags         docs
// @Accept       json
// @Produce      json
// @Param        page  body      object{Path=string,Markdown=string}  true  "Page"
// @Success      200   {object}  object{HTML=string}
// @Router       /docs/preview [post]
func previewDoc(c *gin.Context) {
	var in struct{ Path, Markdown string }
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	rel, _ := docsource.Clean(in.Path)
	html, _ := docsource.Render(rel, []byte(in.Markdown))
	c.IndentedJSON(http.StatusOK, gin.H{"HTML": html})
}

var docImages = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".avif": "image/avif", ".svg": "image/svg+xml",
}

// getDocFile godoc
// @Summary      An image from the docs folder
// @Tags         docs
// @Param        path  query  string  true  "Path inside the docs folder"
// @Success      200
// @Router       /docs/file [get]
func getDocFile(c *gin.Context) {
	rel, ok := docsource.Clean(c.Query("path"))
	kind := docImages[strings.ToLower(path.Ext(rel))]
	if !ok || kind == "" || conf.AppConfig.DocsRepo == "" {
		c.Status(http.StatusNotFound)
		return
	}
	data, found, err := docsSource().File(rel)
	if err != nil || !found {
		c.Status(http.StatusNotFound)
		return
	}
	// an SVG opened on its own must not run script as this site
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Data(http.StatusOK, kind, data)
}

type docsSettings struct {
	Repo       string // owner/name, or the repository's GitHub address
	Branch     string
	Dir        string
	Token      string // "" keeps the saved token
	ClearToken bool
	Edit       bool
}

var repoRe = regexp.MustCompile(`^[\w.-]+/[\w.-]+$`)

// docsRepo - "owner/name" from what someone pasted: the name itself, or a
// github.com address with or without .git
func docsRepo(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if i := strings.Index(s, "github.com"); i >= 0 {
		s = strings.TrimLeft(s[i+len("github.com"):], ":/")
	}
	s = strings.TrimSuffix(strings.TrimSuffix(s, "/"), ".git")
	if parts := strings.Split(s, "/"); len(parts) > 2 {
		s = parts[0] + "/" + parts[1] // an address to a file or folder in the repository
	}
	if !repoRe.MatchString(s) {
		return "", fmt.Errorf("the repository should be owner/name, e.g. octocat/homelab")
	}
	return s, nil
}

// saveDocsSettings godoc
// @Summary      Choose where docs come from
// @Description  An empty Token keeps the saved one; ClearToken removes it. The token is never sent back.
// @Tags         docs
// @Accept       json
// @Produce      json
// @Param        docs  body      docsSettings  true  "Docs settings"
// @Success      200   {object}  docsIndex
// @Router       /config/docs [post]
func saveDocsSettings(c *gin.Context) {
	var in docsSettings
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	repo, err := docsRepo(in.Repo)
	if err != nil {
		badRequest(c, err)
		return
	}

	cfg := &conf.AppConfig
	cfg.DocsRepo = repo
	cfg.DocsBranch = strings.TrimSpace(in.Branch)
	cfg.DocsDir = strings.Trim(strings.TrimSpace(in.Dir), "/")
	cfg.DocsEdit = in.Edit
	if !cfg.DocsTokenEnv {
		if t := strings.TrimSpace(in.Token); t != "" {
			cfg.DocsToken = t
		} else if in.ClearToken {
			cfg.DocsToken = ""
		}
	}
	cfg.DocsHasToken = cfg.DocsToken != ""
	conf.Write(conf.AppConfig)
	docsource.Forget()

	getDocs(c) // answers with the menu, or the error reading it
}
