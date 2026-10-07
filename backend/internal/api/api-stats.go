package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/stats"
	"github.com/vipthomps/gimle/backend/internal/subnet"
)

func daysParam(c *gin.Context) int {
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 {
		return 30
	}
	return min(days, 366)
}

// getStats godoc
// @Summary      Uptime and utilization stats
// @Description  Online hosts and used addresses over time (hourly up to 7 days, daily beyond), uptime per host, and new hosts per day
// @Tags         stats
// @Produce      json
// @Param        days  query     int  false  "Days to look back, today included (default 30)"
// @Success      200   {object}  models.Stats
// @Router       /stats [get]
func getStats(c *gin.Context) {
	days := daysParam(c)
	fromDay, fromHour, bucket := stats.Range(time.Now(), days)

	res := models.Stats{Days: days, From: fromDay, Bucket: bucket,
		Subnets: []models.SubnetTrend{}, Hosts: []models.HostUptime{}}

	trends := stats.Trends(gdb.SelectSubnetHours(fromHour), bucket)
	res.All = orEmpty(trends[0])

	hosts, _ := gdb.Select("now")
	for _, s := range gdb.SelectSubnets() {
		st := models.SubnetTrend{Subnet: s, Points: orEmpty(trends[s.ID])}
		if ipam, err := subnet.Map(s, hosts); err == nil {
			st.Now = ipam.Stat
		}
		res.Subnets = append(res.Subnets, st)
	}

	byMac := make(map[string][]models.HostDay)
	for _, d := range gdb.SelectHostDays(fromDay) {
		byMac[d.Mac] = append(byMac[d.Mac], d)
	}
	firstSeen := gdb.SelectFirstSeen()
	first := make(map[string]models.FirstSeen)
	for _, f := range firstSeen {
		first[f.Mac] = f
	}
	for _, h := range hosts {
		res.Hosts = append(res.Hosts, stats.Uptime(h, first[h.Mac], byMac[h.Mac]))
	}
	res.NewHosts = stats.NewPerDay(firstSeen, fromDay)

	c.IndentedJSON(http.StatusOK, res)
}

// getHostStats godoc
// @Summary      Uptime of one host
// @Tags         stats
// @Produce      json
// @Param        mac   path      string  true   "Host MAC"
// @Param        days  query     int     false  "Days to look back, today included (default 30)"
// @Success      200   {object}  models.HostUptime
// @Router       /stats/host/{mac} [get]
func getHostStats(c *gin.Context) {
	mac := c.Param("mac")
	hosts := gdb.SelectByMAC("now", mac)
	if len(hosts) == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "host not found"})
		return
	}
	fromDay, _, _ := stats.Range(time.Now(), daysParam(c))
	var first models.FirstSeen
	for _, f := range gdb.SelectFirstSeen() {
		if f.Mac == mac {
			first = f
		}
	}
	c.IndentedJSON(http.StatusOK, stats.Uptime(hosts[0], first, gdb.SelectHostDaysByMAC(mac, fromDay)))
}

func orEmpty(p []models.TrendPoint) []models.TrendPoint {
	if p == nil {
		return []models.TrendPoint{}
	}
	return p
}
