package api

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/category"
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/linkcheck"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/names"
	"github.com/vipthomps/gimle/backend/internal/portscan"
)

func badRequest(c *gin.Context, err error) {
	c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func idParam(c *gin.Context) int {
	id, _ := strconv.Atoi(c.Param("id"))
	return id
}

// getViews godoc
// @Summary      List views
// @Description  Custom dashboard tabs in order
// @Tags         views
// @Produce      json
// @Success      200  {array}   models.View
// @Router       /views [get]
func getViews(c *gin.Context) {
	views := gdb.SelectViews()
	if views == nil {
		views = []models.View{}
	}
	c.IndentedJSON(http.StatusOK, views)
}

// saveView godoc
// @Summary      Create or update a view
// @Description  ID 0 creates a view at the end. Layout is "tiles" or "map"
// @Tags         views
// @Accept       json
// @Produce      json
// @Param        view  body      models.View  true  "View"
// @Success      200   {object}  models.View
// @Router       /views [post]
func saveView(c *gin.Context) {
	var v models.View
	if err := c.ShouldBindJSON(&v); err != nil {
		badRequest(c, err)
		return
	}
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" {
		badRequest(c, fmt.Errorf("a view needs a name"))
		return
	}
	v.Tags = strings.Join(splitTags(v.Tags), ",")
	if v.Layout == "" {
		v.Layout = "tiles"
	}
	if v.Layout != "tiles" && v.Layout != "map" {
		badRequest(c, fmt.Errorf("layout must be tiles or map"))
		return
	}
	if v.ID == 0 {
		v.Sort = len(gdb.SelectViews())
	} else if old, ok := gdb.SelectView(v.ID); ok {
		v.Sort = old.Sort
	} else {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "view not found"})
		return
	}
	if err := gdb.SaveView(&v); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, v)
}

// deleteView godoc
// @Summary      Delete a view
// @Description  Deletes the view only. Tags stay on hosts, services and links
// @Tags         views
// @Produce      json
// @Param        id   path      int  true  "View ID"
// @Success      200  {string}  string  "OK"
// @Router       /views/{id} [delete]
func deleteView(c *gin.Context) {
	gdb.DeleteView(idParam(c))
	if conf.AppConfig.StartPage == "/view/"+c.Param("id") {
		conf.AppConfig.StartPage = "" // back to the network map
		conf.Write(conf.AppConfig)
	}
	c.IndentedJSON(http.StatusOK, "OK")
}

// moveView godoc
// @Summary      Move a view left or right
// @Tags         views
// @Produce      json
// @Param        id   path      int  true  "View ID"
// @Param        dir  query     int  true  "-1 or 1"
// @Success      200  {string}  string  "OK"
// @Router       /views/{id}/move [post]
func moveView(c *gin.Context) {
	views := gdb.SelectViews()
	order := make([]int, len(views))
	for i, v := range views {
		order[i] = v.ID
	}
	order = moveID(order, idParam(c), c.Query("dir"))
	for _, v := range views {
		v.Sort = indexOf(order, v.ID)
		_ = gdb.SaveView(&v)
	}
	c.IndentedJSON(http.StatusOK, "OK")
}

// getView godoc
// @Summary      View with groups and items
// @Description  Items come with a title, a link and an up/down status
// @Tags         views
// @Produce      json
// @Param        id   path      int  true  "View ID"
// @Success      200  {object}  models.ViewData
// @Router       /views/{id} [get]
func getView(c *gin.Context) {
	v, ok := gdb.SelectView(idParam(c))
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "view not found"})
		return
	}

	byTag := make(map[string][]models.Item)
	for _, it := range gdb.SelectAllItems() {
		byTag[it.Tag] = append(byTag[it.Tag], it)
	}

	tags := splitTags(v.Tags)
	if len(tags) == 0 {
		for t := range byTag {
			tags = append(tags, t)
		}
		sort.Slice(tags, func(i, j int) bool { return strings.ToLower(tags[i]) < strings.ToLower(tags[j]) })
	}

	c.IndentedJSON(http.StatusOK, buildView(v, tags, byTag, hostIndex()))
}

// getAutoView godoc
// @Summary      Suggested categories view
// @Description  Every web service, and every host without one, grouped by its suggested category. Needs no tags.
// @Tags         views
// @Produce      json
// @Success      200  {object}  models.ViewData
// @Router       /views/auto [get]
func getAutoView(c *gin.Context) {
	hosts := hostIndex()

	byCat := make(map[string][]models.Item)
	for _, h := range hosts {
		if len(h.Web) == 0 {
			cat := orDefault(h.Category, "Other")
			byCat[cat] = append(byCat[cat], models.Item{Kind: "host", Mac: h.Mac, Tag: cat})
			continue
		}
		for _, p := range h.Web {
			cat := orDefault(serviceCategory(p, h), "Other")
			byCat[cat] = append(byCat[cat], models.Item{Kind: "service", Mac: h.Mac, Port: p.Port, Tag: cat})
		}
	}

	var tags []string
	for _, cat := range category.Order {
		if len(byCat[cat]) > 0 {
			tags = append(tags, cat)
		}
	}
	v := models.View{Name: "Categories", Layout: "tiles"}
	data := buildView(v, tags, byCat, hosts)
	for gi := range data.Groups {
		items := data.Groups[gi].Items
		sort.SliceStable(items, func(i, j int) bool {
			return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
		})
	}
	c.IndentedJSON(http.StatusOK, data)
}

// applyAutoView godoc
// @Summary      Turn suggested categories into tags
// @Description  Tags every host and service that has no tag yet with its suggested category, and adds a "Categories" view showing those tags. Existing tags are left alone.
// @Tags         views
// @Produce      json
// @Success      200  {object}  models.View
// @Router       /views/auto/apply [post]
func applyAutoView(c *gin.Context) {
	tagged := make(map[string]bool)
	for _, it := range gdb.SelectAllItems() {
		if it.Kind == "host" {
			tagged[iconKey(it.Mac, 0)] = true
		} else if it.Kind == "service" {
			tagged[iconKey(it.Mac, it.Port)] = true
		}
	}

	hosts := hostIndex()
	used := make(map[string]bool)
	add := func(cat, kind, mac string, port int) {
		key := iconKey(mac, port)
		if tagged[key] {
			return
		}
		tagged[key] = true
		used[cat] = true
		it := models.Item{Tag: cat, Kind: kind, Mac: mac, Port: port, Sort: len(gdb.SelectItemsByTag(cat))}
		if err := gdb.SaveItem(&it); err != nil {
			check.IfError(err)
		}
	}
	for _, h := range hosts {
		if len(h.Web) == 0 {
			add(orDefault(h.Category, "Other"), "host", h.Mac, 0)
			continue
		}
		for _, p := range h.Web {
			add(orDefault(serviceCategory(p, h), "Other"), "service", h.Mac, p.Port)
		}
	}

	var tags []string
	for _, cat := range category.Order {
		if used[cat] || len(gdb.SelectItemsByTag(cat)) > 0 {
			tags = append(tags, cat)
		}
	}
	v := models.View{Name: "Categories", Layout: "tiles", Tags: strings.Join(tags, ","), Sort: len(gdb.SelectViews())}
	if err := gdb.SaveView(&v); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, v)
}

func hostIndex() map[string]*models.MapHost {
	hosts := make(map[string]*models.MapHost)
	for _, h := range mapHosts(gdb.SelectSubnets()) {
		h := h
		hosts[h.Mac] = &h
	}
	return hosts
}

// buildView - groups of items in tag order, with titles, icons and status
func buildView(v models.View, tags []string, byTag map[string][]models.Item, hosts map[string]*models.MapHost) models.ViewData {
	data := models.ViewData{View: v, Groups: []models.ViewGroup{}}
	var wg sync.WaitGroup

	icons := make(map[string]string)
	for _, ic := range gdb.SelectIcons() {
		icons[iconKey(ic.Mac, ic.Port)] = ic.Icon
	}
	marks := make(map[int]models.Bookmark)
	for _, b := range gdb.SelectBookmarks() {
		marks[b.ID] = b
	}

	for _, t := range tags {
		vg := models.ViewGroup{Tag: t, Items: []models.ViewItem{}}
		for _, it := range byTag[t] {
			vi := describe(it, hosts, marks)
			mac, port := it.Mac, it.Port
			if it.Kind == "host" {
				port = 0
			}
			if it.Kind == "bookmark" {
				mac, port = marks[it.Bookmark].Mac, marks[it.Bookmark].Port
			}
			if vi.Icon == "" && mac != "" {
				vi.Icon = icons[iconKey(mac, port)]
				if vi.Icon == "" {
					vi.Icon = icons[iconKey(mac, 0)] // a service falls back to its host's icon
				}
			}
			vg.Items = append(vg.Items, vi)
		}
		data.Groups = append(data.Groups, vg)
	}

	// Check services and links at the same time
	for gi := range data.Groups {
		for ii := range data.Groups[gi].Items {
			vi := &data.Groups[gi].Items[ii]
			if vi.Kind == "host" {
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				vi.Status = checkItem(vi)
			}()
		}
	}
	wg.Wait()

	return data
}

// describe - fill in what the dashboard shows for an item
func describe(it models.Item, hosts map[string]*models.MapHost, marks map[int]models.Bookmark) models.ViewItem {
	vi := models.ViewItem{Item: it, Status: "unknown"}
	h := hosts[it.Mac]
	if it.Kind == "bookmark" {
		h = hosts[marks[it.Bookmark].Mac]
	}

	switch it.Kind {
	case "host":
		if h == nil {
			vi.Title = orDefault(it.Name, "Removed host")
			return vi
		}
		vi.Host = h
		vi.Title = orDefault(it.Name, hostLabel(h))
		vi.Subtitle = h.IP
		vi.Link = it.URL
		vi.Status = "down"
		if h.Now == 1 {
			vi.Status = "up"
		}

	case "service":
		name, web := portscan.Service(it.Port)
		if h == nil {
			vi.Title = orDefault(it.Name, orDefault(name, "Removed host"))
			return vi
		}
		vi.Host = h
		for _, p := range h.Web {
			if p.Port == it.Port {
				web = p.Web
				name = orDefault(p.Title, orDefault(p.Container, name)) // the page's own title beats a guess from the port number
			}
		}
		if web == "" {
			web = "http"
		}
		vi.Title = orDefault(it.Name, orDefault(name, hostLabel(h)+":"+strconv.Itoa(it.Port)))
		vi.Subtitle = hostLabel(h) + " :" + strconv.Itoa(it.Port)
		vi.Link = orDefault(it.URL, orDefault(serviceBookmark(h, it.Port), webURL(h.IP, it.Port, web)))

	case "bookmark":
		b, ok := marks[it.Bookmark]
		if !ok {
			vi.Title = "Removed bookmark"
			return vi
		}
		host, _, _ := linkcheck.Target(b.URL)
		vi.Link = b.URL
		vi.Title = orDefault(b.Name, host)
		vi.Subtitle = orDefault(b.Note, host)
		vi.Icon = b.Icon
		vi.Host = h // the linked host, if any, opens from the map layout
	}
	return vi
}

func checkItem(vi *models.ViewItem) string {
	var host string
	var port int

	if vi.Kind == "service" && serviceBookmark(vi.Host, vi.Port) == "" {
		if vi.Host == nil {
			return "unknown"
		}
		host, port = vi.Host.IP, vi.Port
	} else {
		var ok bool
		if host, port, ok = linkcheck.Target(vi.Link); !ok {
			return "unknown"
		}
	}
	if linkcheck.Up(host, port) {
		return "up"
	}
	return "down"
}

func hostLabel(h *models.MapHost) string {
	return orDefault(h.Name, orDefault(names.Short(h.DNS), h.IP))
}

func orDefault(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func webURL(ip string, port int, web string) string {
	if (web == "https" && port == 443) || (web == "http" && port == 80) {
		return web + "://" + ip
	}
	return web + "://" + ip + ":" + strconv.Itoa(port)
}

// saveItem godoc
// @Summary      Create or update an item
// @Description  Kind "host" needs Mac, "service" needs Mac and Port. Bookmarks are tagged through /bookmarks
// @Tags         views
// @Accept       json
// @Produce      json
// @Param        item  body      models.Item  true  "Item"
// @Success      200   {object}  models.Item
// @Router       /items [post]
func saveItem(c *gin.Context) {
	var it models.Item
	if err := c.ShouldBindJSON(&it); err != nil {
		badRequest(c, err)
		return
	}
	if err := validateItem(&it); err != nil {
		badRequest(c, err)
		return
	}
	for _, other := range gdb.SelectItemsByTag(it.Tag) {
		if other.ID != it.ID && other.Kind == it.Kind && other.Mac == it.Mac && other.Port == it.Port {
			c.IndentedJSON(http.StatusOK, other) // already tagged
			return
		}
	}
	if old, ok := gdb.SelectItem(it.ID); ok && old.Tag == it.Tag {
		it.Sort = old.Sort
	} else {
		it.Sort = len(gdb.SelectItemsByTag(it.Tag))
	}
	if err := gdb.SaveItem(&it); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, it)
}

func validateItem(it *models.Item) error {
	it.Tag = strings.TrimSpace(strings.ReplaceAll(it.Tag, ",", " "))
	if it.Tag == "" {
		return fmt.Errorf("a tag is required")
	}
	it.Name = strings.TrimSpace(it.Name)
	it.URL = strings.TrimSpace(it.URL)
	if err := validIcon(&it.Icon); err != nil {
		return err
	}

	if it.URL != "" {
		u, err := url.Parse(it.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("URL must start with http:// or https://")
		}
	}

	switch it.Kind {
	case "host", "service":
		if len(gdb.SelectByMAC("now", it.Mac)) == 0 {
			return fmt.Errorf("host not found")
		}
		if it.Kind == "service" && (it.Port < 1 || it.Port > 65535) {
			return fmt.Errorf("port must be 1-65535")
		}
	default:
		return fmt.Errorf("kind must be host or service; bookmarks get their tags through /bookmarks")
	}
	it.Icon = "" // hosts and services keep theirs in the icons table
	it.Bookmark = 0
	return nil
}

// deleteItem godoc
// @Summary      Delete an item
// @Tags         views
// @Produce      json
// @Param        id   path      int  true  "Item ID"
// @Success      200  {string}  string  "OK"
// @Router       /items/{id} [delete]
func deleteItem(c *gin.Context) {
	gdb.DeleteItem(idParam(c))
	c.IndentedJSON(http.StatusOK, "OK")
}

// moveItem godoc
// @Summary      Move an item earlier or later within its tag
// @Tags         views
// @Produce      json
// @Param        id   path      int  true  "Item ID"
// @Param        dir  query     int  true  "-1 or 1"
// @Success      200  {string}  string  "OK"
// @Router       /items/{id}/move [post]
func moveItem(c *gin.Context) {
	it, ok := gdb.SelectItem(idParam(c))
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	items := gdb.SelectItemsByTag(it.Tag)
	order := make([]int, len(items))
	for i, x := range items {
		order[i] = x.ID
	}
	order = moveID(order, it.ID, c.Query("dir"))
	for _, x := range items {
		x.Sort = indexOf(order, x.ID)
		_ = gdb.SaveItem(&x)
	}
	c.IndentedJSON(http.StatusOK, "OK")
}

// moveID - move id one step in order ("-1" earlier, "1" later)
func moveID(order []int, id int, dir string) []int {
	i := indexOf(order, id)
	j := i + 1
	if dir == "-1" {
		j = i - 1
	}
	if i < 0 || j < 0 || j >= len(order) {
		return order
	}
	order[i], order[j] = order[j], order[i]
	return order
}

func indexOf(list []int, id int) int {
	for i, v := range list {
		if v == id {
			return i
		}
	}
	return -1
}

// splitTags - "a, b,,c" to [a b c]
func splitTags(str string) (tags []string) {
	for _, t := range strings.Split(str, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// getTags godoc
// @Summary      List tags
// @Description  Every tag with how many hosts, services and links carry it
// @Tags         views
// @Produce      json
// @Success      200  {array}   models.TagCount
// @Router       /tags [get]
func getTags(c *gin.Context) {
	counts := make(map[string]int)
	for _, it := range gdb.SelectAllItems() {
		counts[it.Tag]++
	}
	res := []models.TagCount{}
	for t, n := range counts {
		res = append(res, models.TagCount{Tag: t, Count: n})
	}
	sort.Slice(res, func(i, j int) bool { return strings.ToLower(res[i].Tag) < strings.ToLower(res[j].Tag) })
	c.IndentedJSON(http.StatusOK, res)
}

// getHostTags godoc
// @Summary      Tags on a host
// @Description  Tag items for a host and its services
// @Tags         views
// @Produce      json
// @Param        mac  path      string  true  "Host MAC"
// @Success      200  {array}   models.Item
// @Router       /tags/host/{mac} [get]
func getHostTags(c *gin.Context) {
	items := gdb.SelectItemsByMAC(c.Param("mac"))
	if items == nil {
		items = []models.Item{}
	}
	c.IndentedJSON(http.StatusOK, items)
}

func iconKey(mac string, port int) string {
	return mac + "/" + strconv.Itoa(port)
}

var iconName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// validIcon - "" (none set), "none", a dashboard-icons name, or an image URL
func validIcon(icon *string) error {
	*icon = strings.TrimSpace(*icon)
	s := *icon
	if s == "" {
		return nil
	}
	if len(s) > 500 {
		return fmt.Errorf("icon is too long")
	}
	if strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//") {
		return nil // image served from the same site
	}
	if u, err := url.Parse(s); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return nil
	}
	if iconName.MatchString(strings.ToLower(s)) {
		*icon = strings.ToLower(s)
		return nil
	}
	return fmt.Errorf("icon must be a dashboard-icons name like immich, or an image URL")
}

// saveIcon godoc
// @Summary      Set the icon of a host or service
// @Description  Port 0 is the host itself. Icon is a dashboard-icons name (e.g. "immich"), an image URL, "none", or "" to go back to the default
// @Tags         views
// @Accept       json
// @Produce      json
// @Param        icon  body      models.Icon  true  "Icon"
// @Success      200   {object}  models.Icon
// @Router       /icons [post]
func saveIcon(c *gin.Context) {
	var ic models.Icon
	if err := c.ShouldBindJSON(&ic); err != nil {
		badRequest(c, err)
		return
	}
	if len(gdb.SelectByMAC("now", ic.Mac)) == 0 {
		badRequest(c, fmt.Errorf("host not found"))
		return
	}
	if ic.Port < 0 || ic.Port > 65535 {
		badRequest(c, fmt.Errorf("port must be 0-65535"))
		return
	}
	if err := validIcon(&ic.Icon); err != nil {
		badRequest(c, err)
		return
	}
	if err := gdb.SaveIcon(ic); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, ic)
}
