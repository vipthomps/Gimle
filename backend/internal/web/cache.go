package web

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Page chunks are named like MapPage-B2x9kQ1a.js, so a new version gets new names
var hashedAsset = regexp.MustCompile(`-[A-Za-z0-9_-]{8}\.(js|css)$`)

// cacheHeaders - let browsers keep hashed page chunks for good, and make them
// check everything else (the page, index.js, index.css) on every load, so an
// update never leaves a browser running old and new code together
func cacheHeaders(c *gin.Context) {
	p := c.Request.URL.Path
	switch {
	case strings.HasPrefix(p, "/api/"):
	case strings.HasPrefix(p, "/fs/public/assets/") && hashedAsset.MatchString(p):
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	default:
		c.Header("Cache-Control", "no-cache")
	}
	c.Next()
}
