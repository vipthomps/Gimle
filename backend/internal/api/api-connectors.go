package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/connectors"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/routines"
)

// getConnectors godoc
// @Summary      List connectors
// @Description  Docker, Dockhand, Scanopy, UniFi and Technitium sources. Tokens are never returned.
// @Tags         connectors
// @Produce      json
// @Success      200  {array}   models.Connector
// @Router       /connectors [get]
func getConnectors(c *gin.Context) {
	list := gdb.SelectConnectors()
	if list == nil {
		list = []models.Connector{}
	}
	c.IndentedJSON(http.StatusOK, list)
}

// getConnectorKinds godoc
// @Summary      Connector kinds
// @Tags         connectors
// @Produce      json
// @Success      200  {object}  map[string]connectors.KindInfo
// @Router       /connectors/kinds [get]
func getConnectorKinds(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, connectors.Kinds)
}

// saveConnector godoc
// @Summary      Create or update a connector
// @Description  ID 0 creates one. An empty Token keeps the saved token; ClearToken removes it.
// @Tags         connectors
// @Accept       json
// @Produce      json
// @Param        connector  body      models.ConnectorSave  true  "Connector"
// @Success      200        {object}  models.Connector
// @Router       /connectors [post]
func saveConnector(c *gin.Context) {
	var in models.ConnectorSave
	if err := c.ShouldBindJSON(&in); err != nil {
		badRequest(c, err)
		return
	}
	conn := in.Connector
	conn.Name = strings.TrimSpace(conn.Name)
	conn.URL = strings.TrimSpace(conn.URL)
	conn.HostIP = strings.TrimSpace(conn.HostIP)
	conn.Site = strings.TrimSpace(conn.Site)
	if conn.Name == "" {
		conn.Name = connectors.Kinds[conn.Kind].Label
	}
	if conn.Interval <= 0 {
		conn.Interval = 15
	}
	if err := connectors.Check(conn); err != nil {
		badRequest(c, err)
		return
	}

	conn.Token = strings.TrimSpace(in.Token)
	if conn.ID != 0 {
		old, ok := gdb.SelectConnector(conn.ID)
		if !ok {
			c.IndentedJSON(http.StatusNotFound, gin.H{"error": "connector not found"})
			return
		}
		if conn.Token == "" && !in.ClearToken {
			conn.Token = old.Token
		}
		conn.LastSync, conn.LastError, conn.LastCount = old.LastSync, old.LastError, old.LastCount
	}
	if conn.Token == "" && connectors.Kinds[conn.Kind].Token == "required" {
		badRequest(c, fmt.Errorf("%s needs an API token", connectors.Kinds[conn.Kind].Label))
		return
	}
	if err := gdb.SaveConnector(&conn); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	conn.HasToken = conn.Token != ""
	if conn.Enabled {
		go func() { _ = routines.SyncConnector(conn) }()
	}
	c.IndentedJSON(http.StatusOK, conn)
}

// deleteConnector godoc
// @Summary      Delete a connector
// @Description  Also removes the containers and ports it reported
// @Tags         connectors
// @Produce      json
// @Param        id   path      int  true  "Connector ID"
// @Success      200  {string}  string  "OK"
// @Router       /connectors/{id} [delete]
func deleteConnector(c *gin.Context) {
	gdb.DeleteConnector(idParam(c))
	c.IndentedJSON(http.StatusOK, "OK")
}

// syncConnector godoc
// @Summary      Sync a connector now
// @Description  Waits for the sync and returns the connector with its result
// @Tags         connectors
// @Produce      json
// @Param        id   path      int  true  "Connector ID"
// @Success      200  {object}  models.Connector
// @Router       /connectors/{id}/sync [post]
func syncConnector(c *gin.Context) {
	conn, ok := gdb.SelectConnector(idParam(c))
	if !ok {
		c.IndentedJSON(http.StatusNotFound, gin.H{"error": "connector not found"})
		return
	}
	_ = routines.SyncConnector(conn)
	conn, _ = gdb.SelectConnector(conn.ID)
	c.IndentedJSON(http.StatusOK, conn)
}

// getContainers godoc
// @Summary      Containers
// @Description  Containers the connectors reported, optionally for one host
// @Tags         connectors
// @Produce      json
// @Param        mac  query     string  false  "Host MAC"
// @Success      200  {array}   models.Container
// @Router       /containers [get]
func getContainers(c *gin.Context) {
	mac := c.Query("mac")
	list := []models.Container{}
	for _, ct := range gdb.SelectContainers() {
		if mac == "" || ct.Mac == mac {
			list = append(list, describeContainer(ct))
		}
	}
	c.IndentedJSON(http.StatusOK, list)
}
