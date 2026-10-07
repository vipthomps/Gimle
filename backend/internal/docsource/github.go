// Package docsource reads Markdown documentation from a folder in a GitHub
// repository, renders it, and commits edits back when given a token that
// may write to the repository.
package docsource

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// Source - where the docs live
type Source struct {
	API    string // https://api.github.com unless set
	Repo   string // owner/name
	Branch string // "" uses the repository's default branch
	Dir    string // folder inside the repository, "" for the root
	Token  string
}

// ErrConflict - the page changed on GitHub after it was opened for editing
var ErrConflict = errors.New("this page changed on GitHub after you opened it; reload it and make your edit again")

var client = &http.Client{Timeout: 20 * time.Second}

type entry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Sha  string `json:"sha"`
}

type tree struct {
	key     string
	at      time.Time
	entries map[string]entry // keyed by path relative to Dir
}

var (
	mu    sync.Mutex
	cache tree
	blobs = map[string][]byte{} // file contents by blob sha, which never change
)

const treeTTL = 30 * time.Second

// Forget - drop the cached file list, after a save or a settings change
func Forget() {
	mu.Lock()
	cache = tree{}
	mu.Unlock()
}

func (s Source) api() string {
	if s.API == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(s.API, "/")
}

func (s Source) key() string {
	return s.api() + "|" + s.Repo + "|" + s.Branch + "|" + s.Dir
}

func (s Source) do(method, p string, body any, accept string) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.api()+p, rd)
	if err != nil {
		return nil, 0, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	return data, resp.StatusCode, err
}

// explain - turn a GitHub error status into something a person can act on
func (s Source) explain(code int, data []byte, writing bool) error {
	var msg struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &msg)
	switch {
	case code == http.StatusUnauthorized:
		return fmt.Errorf("GitHub refused the token (HTTP 401); check it has not expired")
	case code == http.StatusNotFound && s.Token == "":
		return fmt.Errorf("%s was not found; a private repository needs a token", s.Repo)
	case code == http.StatusNotFound:
		return fmt.Errorf("%s or its branch was not found, or the token cannot see it", s.Repo)
	case code == http.StatusForbidden && writing:
		return fmt.Errorf("the token may read %s but not write to it; give it Contents read and write", s.Repo)
	case code == http.StatusForbidden:
		return fmt.Errorf("GitHub said no (HTTP 403): %s", msg.Message)
	}
	return fmt.Errorf("GitHub answered HTTP %d: %s", code, msg.Message)
}

// files - every file under Dir, keyed by its path relative to Dir
func (s Source) files() (map[string]entry, error) {
	mu.Lock()
	if cache.key == s.key() && time.Since(cache.at) < treeTTL {
		e := cache.entries
		mu.Unlock()
		return e, nil
	}
	mu.Unlock()

	ref := s.Branch
	if ref == "" {
		ref = "HEAD"
	}
	data, code, err := s.do("GET", "/repos/"+s.Repo+"/git/trees/"+url.PathEscape(ref)+"?recursive=1", nil, "")
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, s.explain(code, data, false)
	}
	var t struct {
		Tree []entry `json:"tree"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	prefix := ""
	if s.Dir != "" {
		prefix = s.Dir + "/"
	}
	// MkDocs keeps mkdocs.yml in the folder above its docs; read its menu too
	mkdocs := ""
	if s.Dir != "" {
		mkdocs = path.Join(path.Dir(s.Dir), "mkdocs.yml")
	}
	entries := map[string]entry{}
	for _, e := range t.Tree {
		if e.Type == "blob" && strings.HasPrefix(e.Path, prefix) {
			entries[strings.TrimPrefix(e.Path, prefix)] = e
		} else if e.Type == "blob" && e.Path == mkdocs {
			entries[mkdocsKey] = e
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no files found in %s/%s", s.Repo, s.Dir)
	}

	mu.Lock()
	cache = tree{key: s.key(), at: time.Now(), entries: entries}
	mu.Unlock()
	return entries, nil
}

// read - a file's contents and blob sha; ok is false when there is no such file
func (s Source) read(rel string) (data []byte, sha string, ok bool, err error) {
	entries, err := s.files()
	if err != nil {
		return nil, "", false, err
	}
	e, ok := entries[rel]
	if !ok {
		return nil, "", false, nil
	}

	mu.Lock()
	b, hit := blobs[e.Sha]
	mu.Unlock()
	if hit {
		return b, e.Sha, true, nil
	}

	b, code, err := s.do("GET", "/repos/"+s.Repo+"/git/blobs/"+e.Sha, nil, "application/vnd.github.raw")
	if err != nil {
		return nil, "", false, err
	}
	if code != http.StatusOK {
		return nil, "", false, s.explain(code, b, false)
	}
	mu.Lock()
	if len(blobs) > 2000 {
		blobs = map[string][]byte{}
	}
	blobs[e.Sha] = b
	mu.Unlock()
	return b, e.Sha, true, nil
}

// write - commit one file. sha is the blob being replaced, "" for a new file.
// Returns the new blob sha.
func (s Source) write(rel string, content []byte, sha, message string) (string, error) {
	body := map[string]string{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(content),
	}
	if sha != "" {
		body["sha"] = sha
	}
	if s.Branch != "" {
		body["branch"] = s.Branch
	}

	p := path.Join(s.Dir, rel)
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	data, code, err := s.do("PUT", "/repos/"+s.Repo+"/contents/"+strings.Join(parts, "/"), body, "")
	if err != nil {
		return "", err
	}
	if code == http.StatusConflict || (code == http.StatusUnprocessableEntity && sha == "") {
		if sha == "" {
			return "", fmt.Errorf("%s already exists; open it and edit it instead", rel)
		}
		return "", ErrConflict
	}
	if code != http.StatusOK && code != http.StatusCreated {
		return "", s.explain(code, data, true)
	}
	Forget()

	var out struct {
		Content struct {
			Sha string `json:"sha"`
		} `json:"content"`
	}
	_ = json.Unmarshal(data, &out)
	mu.Lock()
	blobs[out.Content.Sha] = content
	mu.Unlock()
	return out.Content.Sha, nil
}

// webURL - the file on github.com, for a "view on GitHub" link
func (s Source) webURL(rel string) string {
	host := "https://github.com"
	if a := s.api(); a != "https://api.github.com" {
		host = strings.TrimSuffix(strings.TrimSuffix(a, "/api/v3"), "/api")
	}
	ref := s.Branch
	if ref == "" {
		ref = "HEAD"
	}
	return host + "/" + s.Repo + "/blob/" + ref + "/" + path.Join(s.Dir, rel)
}
