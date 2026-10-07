package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// bookmarkInput - a bookmark and the tags to give it
type bookmarkInput struct {
	models.Bookmark
	Tags []string
}

// getBookmarks godoc
// @Summary      List bookmarks
// @Description  Addresses added by hand, with their tags and the host they are linked to
// @Tags         bookmarks
// @Produce      json
// @Success      200  {array}  models.BookmarkInfo
// @Router       /bookmarks [get]
func getBookmarks(c *gin.Context) {
	tags := make(map[int][]string)
	for _, it := range gdb.SelectAllItems() {
		if it.Kind == "bookmark" {
			tags[it.Bookmark] = append(tags[it.Bookmark], it.Tag)
		}
	}
	hosts := hostIndex()

	res := []models.BookmarkInfo{}
	for _, b := range gdb.SelectBookmarks() {
		bi := models.BookmarkInfo{Bookmark: b, Tags: tags[b.ID]}
		if bi.Tags == nil {
			bi.Tags = []string{}
		}
		if h := hosts[b.Mac]; h != nil {
			bi.HostID, bi.HostName = h.ID, hostLabel(h)
		}
		res = append(res, bi)
	}
	c.IndentedJSON(http.StatusOK, res)
}

// saveBookmark godoc
// @Summary      Create or update a bookmark
// @Description  ID 0 creates one. URL must be http(s). Mac links it to a host, Port to one of its services. Tags replace the bookmark's tags.
// @Tags         bookmarks
// @Accept       json
// @Produce      json
// @Param        bookmark  body      api.bookmarkInput  true  "Bookmark with tags"
// @Success      200       {object}  models.Bookmark
// @Router       /bookmarks [post]
func saveBookmark(c *gin.Context) {
	var in bookmarkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	b := in.Bookmark
	if err := validateBookmark(&b); err != nil {
		badRequest(c, err)
		return
	}
	if b.ID != 0 {
		if _, ok := gdb.SelectBookmark(b.ID); !ok {
			c.IndentedJSON(http.StatusNotFound, gin.H{"error": "bookmark not found"})
			return
		}
	}
	if err := gdb.SaveBookmark(&b, cleanTags(in.Tags)); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, b)
}

// deleteBookmark godoc
// @Summary      Delete a bookmark
// @Description  Also removes it from every group it was tagged into
// @Tags         bookmarks
// @Produce      json
// @Param        id   path      int  true  "Bookmark ID"
// @Success      200  {string}  string  "OK"
// @Router       /bookmarks/{id} [delete]
func deleteBookmark(c *gin.Context) {
	gdb.DeleteBookmark(idParam(c))
	c.IndentedJSON(http.StatusOK, "OK")
}

func validateBookmark(b *models.Bookmark) error {
	b.Name = strings.TrimSpace(b.Name)
	b.Note = strings.TrimSpace(b.Note)
	b.URL = strings.TrimSpace(b.URL)
	if b.URL != "" && !strings.Contains(b.URL, "://") {
		b.URL = "https://" + b.URL // "immich.example.lan" is the usual way to type it
	}
	u, err := url.Parse(b.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("the address must be a web address like https://app.example.lan")
	}
	if err := validIcon(&b.Icon); err != nil {
		return err
	}
	if b.Mac == "" {
		b.Port = 0
		return nil
	}
	if len(gdb.SelectByMAC("now", b.Mac)) == 0 {
		return fmt.Errorf("linked host not found")
	}
	if b.Port < 0 || b.Port > 65535 {
		return fmt.Errorf("port must be 0-65535")
	}
	return nil
}

// cleanTags - trimmed, without commas, empties or repeats
func cleanTags(in []string) []string {
	out := []string{}
	seen := make(map[string]bool)
	for _, t := range in {
		t = strings.TrimSpace(strings.ReplaceAll(t, ",", " "))
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// bookmarksByMac - bookmarks linked to each host
func bookmarksByMac(list []models.Bookmark) map[string][]models.Bookmark {
	res := make(map[string][]models.Bookmark)
	for _, b := range list {
		if b.Mac != "" {
			res[b.Mac] = append(res[b.Mac], b)
		}
	}
	return res
}

// serviceBookmark - the address of the bookmark linked to a host's service, "" if none
func serviceBookmark(h *models.MapHost, port int) string {
	if h == nil || port == 0 {
		return ""
	}
	for _, b := range h.Bookmarks {
		if b.Port == port {
			return b.URL
		}
	}
	return ""
}
