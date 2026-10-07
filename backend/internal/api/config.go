package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/names"
	"github.com/vipthomps/gimle/backend/internal/portscan"
	"github.com/vipthomps/gimle/backend/internal/routines"
)

func saveConfigHandler(c *gin.Context) {

	conf.AppConfig.Host = c.PostForm("host")
	conf.AppConfig.Port = c.PostForm("port")
	conf.AppConfig.Theme = c.PostForm("theme")
	conf.AppConfig.Color = c.PostForm("color")
	conf.AppConfig.NodePath = c.PostForm("node")
	conf.AppConfig.ShoutURL = c.PostForm("shout")

	conf.Write(conf.AppConfig)

	c.Redirect(http.StatusFound, c.Request.Referer())
}

func saveSettingsHandler(c *gin.Context) {

	conf.AppConfig.LogLevel = c.PostForm("log")
	conf.AppConfig.ArpArgs = c.PostForm("arpargs")
	conf.AppConfig.Ifaces = c.PostForm("ifaces")

	useDB := c.PostForm("usedb")
	pgConnect := c.PostForm("pgconnect")

	if useDB != conf.AppConfig.UseDB || pgConnect != conf.AppConfig.PGConnect {
		conf.AppConfig.UseDB = c.PostForm("usedb")
		conf.AppConfig.PGConnect = c.PostForm("pgconnect")
		gdb.Connect()
	}

	timeout := c.PostForm("timeout")
	trimHist := c.PostForm("trim")
	conf.AppConfig.Timeout, _ = strconv.Atoi(timeout)
	conf.AppConfig.TrimHist, _ = strconv.Atoi(trimHist)
	if days, err := strconv.Atoi(c.PostForm("statsdays")); err == nil && days >= 0 {
		conf.AppConfig.StatsDays = days
	}

	arpStrs := c.PostFormArray("arpstrs")
	conf.AppConfig.ArpStrs = []string{}
	for _, s := range arpStrs {
		if s != "" {
			conf.AppConfig.ArpStrs = append(conf.AppConfig.ArpStrs, s)
		}
	}

	conf.Write(conf.AppConfig)

	routines.ScanRestart()
	routines.PortScanRestart()

	c.Redirect(http.StatusFound, c.Request.Referer())
}

func savePortsHandler(c *gin.Context) {

	list := strings.TrimSpace(c.PostForm("list"))
	if _, err := portscan.ParseList(list); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conf.AppConfig.PortList = list
	conf.AppConfig.PortScan = c.PostForm("enable") == "on"
	conf.AppConfig.PortInterval, _ = strconv.Atoi(c.PostForm("interval"))
	conf.AppConfig.PortWorkers, _ = strconv.Atoi(c.PostForm("workers"))
	conf.AppConfig.PortTimeout, _ = strconv.Atoi(c.PostForm("timeout"))

	conf.Write(conf.AppConfig)

	routines.PortScanRestart()

	c.Redirect(http.StatusFound, c.Request.Referer())
}

func saveNamesHandler(c *gin.Context) {

	server := strings.TrimSpace(c.PostForm("dnsserver"))
	if err := names.CheckServer(server); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conf.AppConfig.DNSServer = server
	conf.AppConfig.NameMDNS = c.PostForm("mdns") == "on"
	conf.AppConfig.NameNetBIOS = c.PostForm("netbios") == "on"

	conf.Write(conf.AppConfig)

	routines.RetryNames() // try hosts without a name again with the new settings

	c.Redirect(http.StatusFound, c.Request.Referer())
}

func saveInfluxHandler(c *gin.Context) {

	conf.AppConfig.InfluxAddr = c.PostForm("addr")
	conf.AppConfig.InfluxToken = c.PostForm("token")
	conf.AppConfig.InfluxOrg = c.PostForm("org")
	conf.AppConfig.InfluxBucket = c.PostForm("bucket")

	enable := c.PostForm("enable")
	skip := c.PostForm("skip")
	conf.AppConfig.InfluxEnable = enable == "on"
	conf.AppConfig.InfluxSkipTLS = skip == "on"

	conf.Write(conf.AppConfig)

	c.Redirect(http.StatusFound, c.Request.Referer())
}

func savePrometheusHandler(c *gin.Context) {
	enable := c.PostForm("enable")

	conf.AppConfig.PrometheusEnable = enable == "on"

	conf.Write(conf.AppConfig)

	c.Redirect(http.StatusFound, c.Request.Referer())
}

var startPageRe = regexp.MustCompile(`^/(view/(auto|[0-9]+)|hosts|stats|bookmarks)$`)

// saveStartPage godoc
// @Summary      Choose the page shown at /
// @Description  Page is "" for the network map, or "/view/auto", "/view/ID", "/hosts", "/stats" or "/bookmarks"
// @Tags         system
// @Accept       json
// @Produce      json
// @Param        start  body      object{Page=string}  true  "Start page"
// @Success      200    {string}  string  "OK"
// @Router       /config/start [post]
func saveStartPage(c *gin.Context) {
	var in struct{ Page string }
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	if in.Page == "/map" {
		in.Page = ""
	}
	if in.Page != "" && !startPageRe.MatchString(in.Page) {
		badRequest(c, fmt.Errorf("unknown page %q", in.Page))
		return
	}
	conf.AppConfig.StartPage = in.Page
	conf.Write(conf.AppConfig)
	c.IndentedJSON(http.StatusOK, "OK")
}
