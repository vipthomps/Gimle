package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/subnet"
)

// getSubnets godoc
// @Summary      List subnets
// @Description  All configured subnets with their address usage
// @Tags         subnets
// @Produce      json
// @Success      200  {array}   models.SubnetStat
// @Router       /subnets [get]
func getSubnets(c *gin.Context) {
	hosts, _ := gdb.Select("now")
	stats := []models.SubnetStat{}

	for _, s := range gdb.SelectSubnets() {
		ipam, err := subnet.Map(s, hosts)
		if err != nil {
			slog.Error("Bad subnet in DB", "subnet", s.CIDR, "err", err)
			stats = append(stats, models.SubnetStat{Subnet: s})
			continue
		}
		stats = append(stats, ipam.Stat)
	}
	c.IndentedJSON(http.StatusOK, stats)
}

// getSubnetIPAM godoc
// @Summary      Subnet address map
// @Description  Usage and the state of every address in a subnet (addresses omitted above 4096)
// @Tags         subnets
// @Produce      json
// @Param        id   path      int  true  "Subnet ID"
// @Success      200  {object}  models.IPAM
// @Router       /subnets/{id}/ipam [get]
func getSubnetIPAM(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	s, ok := gdb.SelectSubnet(id)
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "subnet not found"})
		return
	}
	hosts, _ := gdb.Select("now")
	ipam, err := subnet.Map(s, hosts)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, ipam)
}

// saveSubnet godoc
// @Summary      Create or update a subnet
// @Description  ID 0 creates a new subnet. Method is "arp" (scan with arp-scan) or "none" (track addresses only)
// @Tags         subnets
// @Accept       json
// @Produce      json
// @Param        subnet  body      models.Subnet  true  "Subnet"
// @Success      200     {object}  models.Subnet
// @Failure      400     {object}  map[string]string
// @Router       /subnets [post]
func saveSubnet(c *gin.Context) {
	var s models.Subnet

	if err := c.ShouldBindJSON(&s); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := subnet.Validate(&s); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, other := range gdb.SelectSubnets() {
		if other.CIDR == s.CIDR && other.ID != s.ID {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": s.CIDR + " already exists"})
			return
		}
	}
	if err := gdb.SaveSubnet(&s); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	slog.Info("Saved subnet", "subnet", s.CIDR)
	c.IndentedJSON(http.StatusOK, s)
}

// deleteSubnet godoc
// @Summary      Delete a subnet
// @Tags         subnets
// @Produce      json
// @Param        id   path      int  true  "Subnet ID"
// @Success      200  {string}  string  "OK"
// @Router       /subnets/{id} [delete]
func deleteSubnet(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	gdb.DeleteSubnet(id)
	c.IndentedJSON(http.StatusOK, "OK")
}

// detectSubnets godoc
// @Summary      Detect subnets
// @Description  Suggest subnets from the IPv4 addresses on this machine's interfaces
// @Tags         subnets
// @Produce      json
// @Success      200  {array}   models.Subnet
// @Router       /subnets/detect [get]
func detectSubnets(c *gin.Context) {
	found := subnet.Detect("")
	if found == nil {
		found = []models.Subnet{}
	}
	c.IndentedJSON(http.StatusOK, found)
}
