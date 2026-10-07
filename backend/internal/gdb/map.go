package gdb

import (
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectMapPos - all saved map positions
func SelectMapPos() (pos []models.MapPos) {

	tab := db.Table("map_pos")
	err := tab.Find(&pos).Error
	check.IfError(err)

	return pos
}

// SaveMapPos - save the map position of one host
func SaveMapPos(pos models.MapPos) error {

	tab := db.Table("map_pos")
	return tab.Save(&pos).Error
}

// ClearMapPos - forget all map positions
func ClearMapPos() {

	tab := db.Table("map_pos")
	err := tab.Where("1 = 1").Delete(&models.MapPos{}).Error
	check.IfError(err)
}

// SelectWebPorts - all stored ports that serve a web page
func SelectWebPorts() (ports []models.Port) {

	tab := db.Table("ports")
	err := tab.Where("\"WEB\" <> ''").Order("\"PORT\"").Find(&ports).Error
	check.IfError(err)

	return ports
}

// DeleteMapPos - forget the map position of one host
func DeleteMapPos(mac string) {

	tab := db.Table("map_pos")
	err := tab.Where("\"MAC\" = ?", mac).Delete(&models.MapPos{}).Error
	check.IfError(err)
}

// SelectAllPorts - every stored port
func SelectAllPorts() (ports []models.Port) {

	err := db.Table("ports").Order("\"PORT\"").Find(&ports).Error
	check.IfError(err)

	return ports
}
