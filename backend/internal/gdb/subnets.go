package gdb

import (
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectSubnets - get all subnets
func SelectSubnets() (subnets []models.Subnet) {

	tab := db.Table("subnets")
	err := tab.Order("\"ID\"").Find(&subnets).Error
	check.IfError(err)

	return subnets
}

// SelectSubnet - get one subnet by ID
func SelectSubnet(id int) (subnet models.Subnet, ok bool) {

	tab := db.Table("subnets")
	err := tab.First(&subnet, id).Error

	return subnet, err == nil
}

// SaveSubnet - create or update a subnet
func SaveSubnet(subnet *models.Subnet) error {

	tab := db.Table("subnets")
	return tab.Save(subnet).Error
}

// DeleteSubnet - delete a subnet
func DeleteSubnet(id int) {

	tab := db.Table("subnets")
	result := tab.Delete(&models.Subnet{}, id)
	check.IfError(result.Error)
}
