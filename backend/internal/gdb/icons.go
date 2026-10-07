package gdb

import (
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectIcons - every host and service icon
func SelectIcons() (icons []models.Icon) {
	err := db.Table("icons").Find(&icons).Error
	check.IfError(err)
	return icons
}

// SaveIcon - set an icon, or remove it when Icon is empty
func SaveIcon(icon models.Icon) error {
	tab := db.Table("icons").Where("\"MAC\" = ? AND \"PORT\" = ?", icon.Mac, icon.Port)
	if err := tab.Delete(&models.Icon{}).Error; err != nil {
		return err
	}
	if icon.Icon == "" {
		return nil
	}
	return db.Table("icons").Create(&icon).Error
}

// DeleteIconsByMAC - remove a deleted host's icons and chosen category
func DeleteIconsByMAC(mac string) {
	err := db.Table("icons").Where("\"MAC\" = ?", mac).Delete(&models.Icon{}).Error
	check.IfError(err)
	err = db.Table("host_categories").Where("\"MAC\" = ?", mac).Delete(&models.HostCategory{}).Error
	check.IfError(err)
}

// SelectHostCategories - categories chosen for hosts, by MAC
func SelectHostCategories() map[string]string {
	var list []models.HostCategory
	err := db.Table("host_categories").Find(&list).Error
	check.IfError(err)
	res := make(map[string]string, len(list))
	for _, c := range list {
		res[c.Mac] = c.Category
	}
	return res
}

// SaveHostCategory - choose a host's category, or go back to the suggested one when Category is empty
func SaveHostCategory(hc models.HostCategory) error {
	tab := db.Table("host_categories").Where("\"MAC\" = ?", hc.Mac)
	if err := tab.Delete(&models.HostCategory{}).Error; err != nil {
		return err
	}
	if hc.Category == "" {
		return nil
	}
	return db.Table("host_categories").Create(&hc).Error
}
