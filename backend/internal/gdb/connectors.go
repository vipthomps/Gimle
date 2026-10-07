package gdb

import (
	"encoding/json"

	"gorm.io/gorm"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectConnectors - all connectors, with their tokens
func SelectConnectors() (list []models.Connector) {

	err := db.Table("connectors").Order("\"ID\"").Find(&list).Error
	check.IfError(err)

	for i := range list {
		list[i].HasToken = list[i].Token != ""
	}
	return list
}

// SelectConnector - one connector by ID
func SelectConnector(id int) (c models.Connector, ok bool) {

	err := db.Table("connectors").Where("\"ID\" = ?", id).First(&c).Error
	c.HasToken = c.Token != ""

	return c, err == nil
}

// SaveConnector - create or update a connector
func SaveConnector(c *models.Connector) error {
	if c.ID == 0 {
		return db.Table("connectors").Create(c).Error
	}
	return db.Table("connectors").Save(c).Error
}

// SetConnectorResult - record how the last sync went
func SetConnectorResult(id int, date, errText string, count int) {

	err := db.Table("connectors").Where("\"ID\" = ?", id).Updates(map[string]any{
		"LAST_SYNC": date, "LAST_ERROR": errText, "LAST_COUNT": count,
	}).Error
	check.IfError(err)
}

// DeleteConnector - delete a connector with the containers and ports it reported
func DeleteConnector(id int) {

	err := db.Table("connectors").Delete(&models.Connector{}, id).Error
	check.IfError(err)
	err = db.Table("containers").Where("\"CONNECTOR_ID\" = ?", id).Delete(&models.Container{}).Error
	check.IfError(err)
	err = db.Table("ports").Where("\"SOURCE\" = ?", id).Delete(&models.Port{}).Error
	check.IfError(err)
	err = db.Table("guests").Where("\"CONNECTOR_ID\" = ?", id).Delete(&models.Guest{}).Error
	check.IfError(err)
}

// ReplaceContainers - store what a connector reported, replacing its last report
func ReplaceContainers(connectorID int, list []models.Container) error {
	for i := range list {
		b, _ := json.Marshal(list[i].Ports)
		list[i].PortsJSON = string(b)
		list[i].ConnectorID = connectorID
		list[i].ID = 0
	}

	tx := db.Begin()
	if err := tx.Table("containers").Where("\"CONNECTOR_ID\" = ?", connectorID).Delete(&models.Container{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(list) > 0 {
		if err := tx.Table("containers").CreateInBatches(list, 100).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// SelectContainers - all containers, ports decoded
func SelectContainers() (list []models.Container) {

	err := db.Table("containers").Order("\"NAME\"").Find(&list).Error
	check.IfError(err)

	for i := range list {
		_ = json.Unmarshal([]byte(list[i].PortsJSON), &list[i].Ports)
		if list[i].Ports == nil {
			list[i].Ports = []models.ContainerPort{}
		}
	}
	return list
}

// SelectPortsBySource - ports a connector reported
func SelectPortsBySource(id int) (ports []models.Port) {

	err := db.Table("ports").Where("\"SOURCE\" = ?", id).Find(&ports).Error
	check.IfError(err)

	return ports
}

// FillHost - set a host's name and DNS name from a connector where they are empty
func FillHost(id int, name, dns string) {
	if name != "" {
		err := db.Table("now").Where("\"ID\" = ? AND \"NAME\" = ''", id).Update("NAME", name).Error
		check.IfError(err)
	}
	if dns != "" {
		err := db.Table("now").Where("\"ID\" = ? AND \"DNS\" = ''", id).Update("DNS", dns).Error
		check.IfError(err)
	}
}

// ReplaceGuests - store the VMs and LXCs a connector reported, replacing its last report
func ReplaceGuests(connectorID int, list []models.Guest) error {
	for i := range list {
		list[i].ConnectorID = connectorID
		list[i].ID = 0
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("guests").Where("\"CONNECTOR_ID\" = ?", connectorID).Delete(&models.Guest{}).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		return tx.Table("guests").CreateInBatches(list, 100).Error
	})
}

// SelectGuests - all VMs and LXCs, by node and VMID
func SelectGuests() (list []models.Guest) {

	err := db.Table("guests").Order("\"NODE\", \"VMID\"").Find(&list).Error
	check.IfError(err)

	return list
}
