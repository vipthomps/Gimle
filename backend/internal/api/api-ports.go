package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/portscan"
	"github.com/vipthomps/gimle/backend/internal/routines"
)

// getHostPorts godoc
// @Summary      Open ports of a host
// @Description  Stored open ports, when the host was last scanned, and the progress of a running scan
// @Tags         ports
// @Produce      json
// @Param        id   path      int  true  "Host ID"
// @Success      200  {object}  models.HostPorts
// @Router       /ports/{id} [get]
func getHostPorts(c *gin.Context) {
	host := getHostByID(c.Param("id")) // functions.go
	if host.ID == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	res := models.HostPorts{
		Ports: gdb.SelectPorts(host.Mac),
		Job:   routines.PortJob(host.ID),
	}
	if res.Ports == nil {
		res.Ports = []models.Port{}
	}
	res.LastScan, _ = gdb.SelectPortScan(host.Mac)

	c.IndentedJSON(http.StatusOK, res)
}

// scanHostPorts godoc
// @Summary      Start a port scan
// @Description  Scan a host in the background. Poll GET /ports/{id} for progress
// @Tags         ports
// @Produce      json
// @Param        id    path      int     true   "Host ID"
// @Param        list  query     string  false  "top (default), all, or a list like 22,80,8000-8100"
// @Success      200   {object}  models.PortJob
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /ports/{id}/scan [post]
func scanHostPorts(c *gin.Context) {
	host := getHostByID(c.Param("id")) // functions.go
	if host.ID == 0 || host.IP == "" {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}

	list := c.DefaultQuery("list", "top")

	if err := routines.StartPortScan(host, list); err != nil {
		status := http.StatusBadRequest
		if routines.PortJob(host.ID).Running {
			status = http.StatusConflict
		}
		c.IndentedJSON(status, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, routines.PortJob(host.ID))
}

// getTopPorts godoc
// @Summary      Ports in the "top" list
// @Description  The ports a "top" port scan checks, with the service usually found on each
// @Tags         ports
// @Produce      json
// @Success      200  {array}  portscan.KnownPort
// @Router       /portlist/top [get]
func getTopPorts(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, portscan.TopList())
}
