package gdb

import (
	"strings"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/names"
)

// Update - update or create host. Returns the host with its ID set.
func Update(table string, oneHost models.Host) models.Host {

	tab := db.Table(table)
	result := tab.Save(&oneHost)
	check.IfError(result.Error)

	return oneHost
}

// Delete - delete host from DB
func Delete(table string, id int) {

	tab := db.Table(table)
	result := tab.Delete(&models.Host{}, id)
	check.IfError(result.Error)
}

// DeleteOldHistory - delete a list of hosts from History
func DeleteOldHistory(date string) int64 {

	tab := db.Table("history")
	result := tab.Where("\"DATE\" < ?", date).Delete(&models.Host{})
	check.IfError(result.Error)

	return result.RowsAffected
}

// Clear - delete all hosts from table
func Clear(table string) {

	tab := db.Table(table)
	result := tab.Where("1 = 1").Delete(&models.Host{})
	check.IfError(result.Error)
}

// SetName - store a looked up name for a host. Only the DNS column changes,
// plus the display name when the host has none yet, so a scan running at
// the same time keeps its own updates.
func SetName(id int, dns string, setName bool) {

	cols := map[string]any{"DNS": dns}
	if setName {
		cols["NAME"] = names.Short(dns)
	}
	err := db.Table("now").Where("\"ID\" = ? AND \"DNS\" = ''", id).Updates(cols).Error
	check.IfError(err)
}

// TidyNames - clean up rows written by older versions: arp-scan's
// "(Unknown...)" vendor text, and display names that are a full DNS name
func TidyNames() {
	for _, table := range []string{"now", "history"} {
		tab := db.Table(table)
		err := tab.Where("\"HW\" LIKE ?", "(Unknown: locally administered%").Update("HW", "Private MAC").Error
		check.IfError(err)
		err = db.Table(table).Where("\"HW\" LIKE ?", "(Unknown%").Update("HW", "").Error
		check.IfError(err)
	}

	var hosts []models.Host
	err := db.Table("now").Where("\"NAME\" LIKE ?", "%.%").Find(&hosts).Error
	check.IfError(err)
	for _, h := range hosts {
		fqdn := strings.TrimSuffix(h.Name, ".")
		if h.DNS == "" || strings.Fields(h.DNS)[0] != fqdn {
			continue // a name someone typed
		}
		err = db.Table("now").Where("\"ID\" = ?", h.ID).Update("NAME", names.Short(fqdn)).Error
		check.IfError(err)
	}
}
