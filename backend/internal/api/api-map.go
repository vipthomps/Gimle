package api

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/category"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// getMap godoc
// @Summary      Map data
// @Description  Subnets and hosts with their subnet, saved map position and web services
// @Tags         map
// @Produce      json
// @Success      200  {object}  models.Map
// @Router       /map [get]
func getMap(c *gin.Context) {
	subnets := gdb.SelectSubnets()
	if subnets == nil {
		subnets = []models.Subnet{}
	}
	c.IndentedJSON(http.StatusOK, models.Map{Subnets: subnets, Hosts: mapHosts(subnets)})
}

// mapHosts - all hosts with their subnet, saved map position and web services
func mapHosts(subnets []models.Subnet) []models.MapHost {
	hosts, _ := gdb.Select("now")

	nets := make([]*net.IPNet, len(subnets))
	for i, s := range subnets {
		_, nets[i], _ = net.ParseCIDR(s.CIDR)
	}

	pos := make(map[string]models.MapPos)
	for _, p := range gdb.SelectMapPos() {
		pos[p.Mac] = p
	}

	web := make(map[string][]models.Port)
	for _, p := range gdb.SelectWebPorts() {
		web[p.Mac] = append(web[p.Mac], p)
	}

	ports := make(map[string][]models.Port)
	for _, p := range gdb.SelectAllPorts() {
		ports[p.Mac] = append(ports[p.Mac], p)
	}

	// One entry per container name and host. Docker and Dockhand know the
	// state, so they win over Scanopy, which only saw the container's ports.
	all := gdb.SelectContainers()
	sort.SliceStable(all, func(i, j int) bool { return all[i].State != "" && all[j].State == "" })
	containers := make(map[string][]models.Container)
	seenCt := make(map[string]bool)
	for _, ct := range all {
		if ct.Mac == "" || seenCt[ct.Mac+"/"+ct.Name] {
			continue
		}
		seenCt[ct.Mac+"/"+ct.Name] = true
		containers[ct.Mac] = append(containers[ct.Mac], describeContainer(ct))
	}
	for _, list := range containers {
		sort.Slice(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	}

	guests := make(map[string][]models.Guest) // node MAC -> its VMs and LXCs
	guestOf := make(map[string]models.Guest)  // guest host MAC -> what it is
	for _, g := range gdb.SelectGuests() {
		if g.NodeMac != "" {
			guests[g.NodeMac] = append(guests[g.NodeMac], g)
		}
		if g.HostMac != "" {
			guestOf[g.HostMac] = g
		}
	}

	icons := make(map[string]string)
	for _, ic := range gdb.SelectIcons() {
		if ic.Port == 0 {
			icons[ic.Mac] = ic.Icon
		}
	}

	chosen := gdb.SelectHostCategories()
	marks := bookmarksByMac(gdb.SelectBookmarks())

	res := []models.MapHost{}
	for _, h := range hosts {
		mh := models.MapHost{Host: h, Web: web[h.Mac], Icon: icons[h.Mac]}
		if mh.Web == nil {
			mh.Web = []models.Port{}
		}
		mh.Containers = containers[h.Mac]
		if mh.Containers == nil {
			mh.Containers = []models.Container{}
		}
		mh.Guests = guests[h.Mac]
		if mh.Guests == nil {
			mh.Guests = []models.Guest{}
		}
		mh.Bookmarks = marks[h.Mac]
		if mh.Bookmarks == nil {
			mh.Bookmarks = []models.Bookmark{}
		}
		if g, ok := guestOf[h.Mac]; ok {
			mh.GuestOf = &g
		}
		if cat, ok := chosen[h.Mac]; ok {
			mh.Category = cat
		} else {
			mh.Category, mh.Suggested = hostCategory(mh, ports[h.Mac]), true
		}
		if p, ok := pos[h.Mac]; ok {
			mh.HasPos, mh.X, mh.Y = true, p.X, p.Y
		}
		if ip := net.ParseIP(h.IP); ip != nil {
			for i, n := range nets {
				if n != nil && n.Contains(ip) {
					mh.SubnetID = subnets[i].ID
					break
				}
			}
		}
		res = append(res, mh)
	}
	return res
}

// saveMapPos godoc
// @Summary      Save a host's map position
// @Tags         map
// @Accept       json
// @Produce      json
// @Param        pos  body      models.MapPos  true  "MAC and position"
// @Success      200  {string}  string  "OK"
// @Failure      400  {object}  map[string]string
// @Router       /map/pos [post]
func saveMapPos(c *gin.Context) {
	var p models.MapPos

	if err := c.ShouldBindJSON(&p); err != nil || p.Mac == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "MAC, X and Y are required"})
		return
	}
	if err := gdb.SaveMapPos(p); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, "OK")
}

// resetMap godoc
// @Summary      Reset the map layout
// @Description  Forget all saved positions so every host is placed automatically
// @Tags         map
// @Produce      json
// @Success      200  {string}  string  "OK"
// @Router       /map/pos [delete]
func resetMap(c *gin.Context) {
	gdb.ClearMapPos()
	c.IndentedJSON(http.StatusOK, "OK")
}

// getCategories godoc
// @Summary      Suggested categories
// @Description  Categories Gimlé suggests for hosts and services, in display order
// @Tags         map
// @Produce      json
// @Success      200  {array}  string
// @Router       /categories [get]
func getCategories(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, category.Order)
}

// saveHostCategory godoc
// @Summary      Choose a host's category
// @Description  Category is one of GET /categories, or "" to go back to the suggested one
// @Tags         map
// @Accept       json
// @Produce      json
// @Param        category  body      models.HostCategory  true  "Host category"
// @Success      200       {object}  models.HostCategory
// @Router       /categories/host [post]
func saveHostCategory(c *gin.Context) {
	var hc models.HostCategory
	if err := c.ShouldBindJSON(&hc); err != nil {
		badRequest(c, err)
		return
	}
	if len(gdb.SelectByMAC("now", hc.Mac)) == 0 {
		badRequest(c, fmt.Errorf("host not found"))
		return
	}
	if hc.Category != "" && !slices.Contains(category.Order, hc.Category) {
		badRequest(c, fmt.Errorf("unknown category %q", hc.Category))
		return
	}
	if err := gdb.SaveHostCategory(hc); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, hc)
}

// hostCategory - suggested category for a host, from what runs on it and what it is called
func hostCategory(h models.MapHost, ports []models.Port) string {
	if len(h.Guests) > 0 {
		return "Virtualization"
	}
	running := 0
	for _, ct := range h.Containers {
		if ct.State == "running" {
			running++
		}
	}
	if running >= 3 {
		return "Containers"
	}
	var texts []string
	for _, p := range ports {
		if p.Category != "" {
			return p.Category
		}
		texts = append(texts, p.Title)
	}
	texts = append(texts, h.Name, h.DNS)
	for _, p := range ports {
		if p.Container == "" {
			texts = append(texts, p.Service)
		}
	}
	texts = append(texts, h.Hw)
	return category.Suggest(texts...)
}

// serviceCategory - suggested category for one service on a host
func serviceCategory(p models.Port, h *models.MapHost) string {
	if p.Category != "" {
		return p.Category
	}
	image := ""
	for _, ct := range h.Containers {
		if ct.Name == p.Container {
			image = ct.Image
		}
	}
	if c := category.Suggest(p.Title, p.Container, image, p.Service); c != "" {
		return c
	}
	return h.Category
}

// describeContainer - add the suggested category
func describeContainer(ct models.Container) models.Container {
	ct.Category = category.Suggest(ct.Name, ct.Image, ct.Project)
	return ct
}

// getSuggestedTags godoc
// @Summary      Suggested categories for a host and its web services
// @Tags         map
// @Produce      json
// @Param        mac  path      string  true  "Host MAC"
// @Success      200  {object}  map[string]string  "port number (0 for the host) -> category"
// @Router       /tags/suggest/{mac} [get]
func getSuggestedTags(c *gin.Context) {
	res := map[string]string{}
	for _, h := range mapHosts(gdb.SelectSubnets()) {
		if h.Mac != c.Param("mac") {
			continue
		}
		if h.Category != "" {
			res["0"] = h.Category
		}
		for _, p := range h.Web {
			if cat := serviceCategory(p, &h); cat != "" {
				res[strconv.Itoa(p.Port)] = cat
			}
		}
	}
	c.IndentedJSON(http.StatusOK, res)
}
