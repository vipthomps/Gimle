package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/vipthomps/gimle/backend/internal/backup"
	"github.com/vipthomps/gimle/backend/internal/docsource"
	"github.com/vipthomps/gimle/backend/internal/gdb"
)

type backupList struct {
	Enabled bool // false when the data is in PostgreSQL
	Dir     string
	Backups []backup.Backup
	Error   string `json:",omitempty"`
}

// getBackups godoc
// @Summary      Saved backups
// @Description  Copies of the database and settings, newest first. They stay in the data folder and are never offered for download, since they hold connector and docs tokens.
// @Tags         backups
// @Produce      json
// @Success      200  {object}  backupList
// @Router       /backups [get]
func getBackups(c *gin.Context) {
	out := backupList{Enabled: gdb.UsingSQLite(), Dir: backup.Dir()}
	list, err := backup.List()
	if err != nil {
		out.Error = err.Error()
	}
	out.Backups = list
	c.IndentedJSON(http.StatusOK, out)
}

// createBackup godoc
// @Summary      Back up now
// @Tags         backups
// @Produce      json
// @Success      200  {object}  backup.Backup
// @Router       /backups [post]
func createBackup(c *gin.Context) {
	b, err := backup.Create(backup.Manual)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, b)
}

// restoreBackup godoc
// @Summary      Restore a backup
// @Description  Puts back the database and settings, keeping the address and port in use. The current state is backed up first, so a restore can be undone.
// @Tags         backups
// @Produce      json
// @Param        name  path      string  true  "Backup file name"
// @Success      200   {string}  string  "OK"
// @Router       /backups/{name}/restore [post]
func restoreBackup(c *gin.Context) {
	if err := backup.Restore(c.Param("name")); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	docsource.Forget()
	c.IndentedJSON(http.StatusOK, "OK")
}
