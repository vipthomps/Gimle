package gdb

import (
	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectViews - all views in tab order
func SelectViews() (views []models.View) {
	err := db.Table("views").Order("\"SORT\", \"ID\"").Find(&views).Error
	check.IfError(err)
	return views
}

// SelectView - one view by ID
func SelectView(id int) (view models.View, ok bool) {
	err := db.Table("views").First(&view, id).Error
	return view, err == nil
}

// SaveView - create or update a view
func SaveView(view *models.View) error {
	return db.Table("views").Save(view).Error
}

// DeleteView - delete a view. Tags and items stay
func DeleteView(id int) {
	err := db.Table("views").Delete(&models.View{}, id).Error
	check.IfError(err)
}

// SelectAllItems - every tagged item
func SelectAllItems() (items []models.Item) {
	err := db.Table("items").Order("\"TAG\", \"SORT\", \"ID\"").Find(&items).Error
	check.IfError(err)
	return items
}

// SelectItemsByTag - items with one tag, in order
func SelectItemsByTag(tag string) (items []models.Item) {
	err := db.Table("items").Where("\"TAG\" = ?", tag).Order("\"SORT\", \"ID\"").Find(&items).Error
	check.IfError(err)
	return items
}

// SelectItemsByMAC - tags on a host and its services
func SelectItemsByMAC(mac string) (items []models.Item) {
	err := db.Table("items").Where("\"MAC\" = ?", mac).Order("\"TAG\", \"PORT\"").Find(&items).Error
	check.IfError(err)
	return items
}

// SelectItem - one item by ID
func SelectItem(id int) (item models.Item, ok bool) {
	err := db.Table("items").First(&item, id).Error
	return item, err == nil
}

// SaveItem - create or update an item
func SaveItem(item *models.Item) error {
	return db.Table("items").Save(item).Error
}

// DeleteItem - delete one item
func DeleteItem(id int) {
	err := db.Table("items").Delete(&models.Item{}, id).Error
	check.IfError(err)
}

// DeleteItemsByMAC - remove a deleted host's tags
func DeleteItemsByMAC(mac string) {
	err := db.Table("items").Where("\"MAC\" = ?", mac).Delete(&models.Item{}).Error
	check.IfError(err)
}
