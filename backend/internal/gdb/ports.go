package gdb

import (
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectPorts - open ports of a host, by MAC
func SelectPorts(mac string) (ports []models.Port) {

	tab := db.Table("ports")
	err := tab.Where("\"MAC\" = ?", mac).Order("\"PORT\"").Find(&ports).Error
	check.IfError(err)

	return ports
}

// SelectPortScan - when a host was last port scanned
func SelectPortScan(mac string) (scan models.PortScan, ok bool) {

	tab := db.Table("port_scans")
	err := tab.Where("\"MAC\" = ?", mac).First(&scan).Error

	return scan, err == nil
}

// SavePortScan - record a finished port scan
func SavePortScan(scan models.PortScan) {

	tab := db.Table("port_scans")
	err := tab.Save(&scan).Error
	check.IfError(err)
}

// SavePort - create or update one port
func SavePort(port models.Port) {

	tab := db.Table("ports")
	err := tab.Save(&port).Error
	check.IfError(err)
}

// DeletePort - delete one port
func DeletePort(id int) {

	tab := db.Table("ports")
	err := tab.Delete(&models.Port{}, id).Error
	check.IfError(err)
}

// DeletePortsByMAC - delete all ports and scan records of a host
func DeletePortsByMAC(mac string) {

	err := db.Table("ports").Where("\"MAC\" = ?", mac).Delete(&models.Port{}).Error
	check.IfError(err)
	err = db.Table("port_scans").Where("\"MAC\" = ?", mac).Delete(&models.PortScan{}).Error
	check.IfError(err)
}
