package api

import (
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Routes - start API routes
func Routes(router *gin.Engine) {

	r0 := router.Group("/api")
	{
		r0.GET("/all", getAllHosts)                // api-hosts.go
		r0.GET("/edit/:id/:name/*known", editHost) // api-hosts.go
		r0.GET("/host/:id", getHost)               // api-hosts.go
		r0.GET("/host/del/:id", delHost)           // api-hosts.go
		r0.GET("/host/add/:mac", addHost)          // api-hosts.go

		r0.GET("/config", getConfig)        // api-system.go
		r0.GET("/notify_test", notifyTest)  // api-system.go
		r0.GET("/status/*iface", getStatus) // api-system.go
		r0.GET("/version", getVersion)      // api-system.go
		r0.GET("/rescan", triggerRescan)    // api-system.go

		r0.GET("/history", getHistory)                  // api-history.go
		r0.GET("/history/:mac", getHistoryByMAC)        // api-history.go
		r0.GET("/history/:mac/:date", getHistoryByDate) // api-history.go

		r0.GET("/port/:addr/:port", getPortState) // api-network.go
		r0.GET("/wol/:mac", sendWOL)              // api-network.go

		r0.GET("/subnets", getSubnets)             // api-subnets.go
		r0.GET("/subnets/detect", detectSubnets)   // api-subnets.go
		r0.GET("/subnets/:id/ipam", getSubnetIPAM) // api-subnets.go
		r0.POST("/subnets", saveSubnet)            // api-subnets.go
		r0.DELETE("/subnets/:id", deleteSubnet)

		r0.GET("/ports/:id", getHostPorts)        // api-ports.go
		r0.POST("/ports/:id/scan", scanHostPorts) // api-ports.go
		r0.GET("/portlist/top", getTopPorts)      // api-ports.go

		r0.GET("/map", getMap)          // api-map.go
		r0.POST("/map/pos", saveMapPos) // api-map.go
		r0.DELETE("/map/pos", resetMap) // api-map.go

		r0.GET("/views", getViews)         // api-views.go
		r0.POST("/views", saveView)        // api-views.go
		r0.GET("/views/auto", getAutoView) // api-views.go
		r0.POST("/views/auto/apply", applyAutoView)
		r0.GET("/views/:id", getView)          // api-views.go
		r0.DELETE("/views/:id", deleteView)    // api-views.go
		r0.POST("/views/:id/move", moveView)   // api-views.go
		r0.GET("/tags", getTags)               // api-views.go
		r0.GET("/tags/host/:mac", getHostTags) // api-views.go
		r0.GET("/tags/suggest/:mac", getSuggestedTags)
		r0.POST("/items", saveItem)         // api-views.go
		r0.DELETE("/items/:id", deleteItem) // api-views.go
		r0.POST("/items/:id/move", moveItem)
		r0.POST("/icons", saveIcon)
		r0.GET("/bookmarks", getBookmarks)          // api-bookmarks.go
		r0.POST("/bookmarks", saveBookmark)         // api-bookmarks.go
		r0.DELETE("/bookmarks/:id", deleteBookmark) // api-bookmarks.go

		r0.GET("/docs", getDocs)                  // api-docs.go
		r0.GET("/docs/page", getDocPage)          // api-docs.go
		r0.POST("/docs/page", saveDocPage)        // api-docs.go
		r0.POST("/docs/preview", previewDoc)      // api-docs.go
		r0.GET("/docs/file", getDocFile)          // api-docs.go
		r0.POST("/config/docs", saveDocsSettings) // api-docs.go

		r0.GET("/backups", getBackups)                   // api-backups.go
		r0.POST("/backups", createBackup)                // api-backups.go
		r0.POST("/backups/:name/restore", restoreBackup) // api-backups.go

		r0.GET("/stats", getStats)               // api-stats.go
		r0.GET("/stats/host/:mac", getHostStats) // api-stats.go // api-views.go

		r0.GET("/connectors", getConnectors)           // api-connectors.go
		r0.GET("/connectors/kinds", getConnectorKinds) // api-connectors.go
		r0.POST("/connectors", saveConnector)          // api-connectors.go
		r0.DELETE("/connectors/:id", deleteConnector)  // api-connectors.go
		r0.POST("/connectors/:id/sync", syncConnector) // api-connectors.go
		r0.GET("/containers", getContainers)           // api-connectors.go
		r0.POST("/categories/host", saveHostCategory)  // api-map.go
		r0.GET("/categories", getCategories)           // api-map.go

		r0.POST("/config/", saveConfigHandler)                // config.go
		r0.POST("/config_settings/", saveSettingsHandler)     // config.go
		r0.POST("/config_influx/", saveInfluxHandler)         // config.go
		r0.POST("/config_prometheus/", savePrometheusHandler) // config.go
		r0.POST("/config_ports/", savePortsHandler)           // config.go
		r0.POST("/config_names/", saveNamesHandler)           // config.go
		r0.POST("/config/start", saveStartPage)               // config.go
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
