package gdb

import (
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// SelectBookmarks - every bookmark, by name
func SelectBookmarks() (list []models.Bookmark) {
	err := db.Table("bookmarks").Where("\"HIDDEN\" IS NOT TRUE").Order("\"ID\"").Find(&list).Error
	check.IfError(err)
	sort.SliceStable(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	return list
}

// SelectBookmark - one bookmark by ID
func SelectBookmark(id int) (b models.Bookmark, ok bool) {
	err := db.Table("bookmarks").First(&b, id).Error
	return b, err == nil
}

// SaveBookmark - create or update a bookmark, and set its tags. Tags it
// already had keep their place in their groups; new ones go at the end.
func SaveBookmark(b *models.Bookmark, tags []string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("bookmarks").Save(b).Error; err != nil {
			return err
		}
		var old []models.Item
		if err := tx.Table("items").Where("\"KIND\" = ? AND \"BOOKMARK\" = ?", "bookmark", b.ID).Find(&old).Error; err != nil {
			return err
		}
		keep := make(map[string]bool)
		for _, t := range tags {
			keep[t] = true
		}
		have := make(map[string]bool)
		for _, it := range old {
			if keep[it.Tag] && !have[it.Tag] {
				have[it.Tag] = true
				continue
			}
			if err := tx.Table("items").Delete(&models.Item{}, it.ID).Error; err != nil {
				return err
			}
		}
		for _, t := range tags {
			if have[t] {
				continue
			}
			have[t] = true
			var n int64
			if err := tx.Table("items").Where("\"TAG\" = ?", t).Count(&n).Error; err != nil {
				return err
			}
			it := models.Item{Tag: t, Kind: "bookmark", Bookmark: b.ID, Sort: int(n)}
			if err := tx.Table("items").Create(&it).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteBookmark - delete a bookmark and its tags. One a connector keeps in
// sync is hidden instead, so the next sync doesn't add it again.
func DeleteBookmark(id int) {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("items").Where("\"KIND\" = ? AND \"BOOKMARK\" = ?", "bookmark", id).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		var b models.Bookmark
		if err := tx.Table("bookmarks").First(&b, id).Error; err != nil {
			return nil
		}
		if b.Source != 0 {
			return tx.Table("bookmarks").Where("\"ID\" = ?", id).Update("HIDDEN", true).Error
		}
		return tx.Table("bookmarks").Delete(&models.Bookmark{}, id).Error
	})
	check.IfError(err)
}

// SelectSourceBookmarks - the bookmarks a connector keeps, hidden ones too
func SelectSourceBookmarks(source int) (list []models.Bookmark) {
	err := db.Table("bookmarks").Where("\"SOURCE\" = ?", source).Order("\"ID\"").Find(&list).Error
	check.IfError(err)
	return list
}

// SaveSourceBookmark - create or update a bookmark a connector keeps, leaving its tags alone
func SaveSourceBookmark(b *models.Bookmark) error {
	return db.Table("bookmarks").Save(b).Error
}

// DeleteSourceBookmarks - remove bookmarks a connector kept, with their tags
func DeleteSourceBookmarks(source int, ids []int) {
	if len(ids) == 0 {
		return
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("items").Where("\"KIND\" = ? AND \"BOOKMARK\" IN ?", "bookmark", ids).Delete(&models.Item{}).Error; err != nil {
			return err
		}
		return tx.Table("bookmarks").Where("\"SOURCE\" = ? AND \"ID\" IN ?", source, ids).Delete(&models.Bookmark{}).Error
	})
	check.IfError(err)
}

// migrateLinks - links used to live on their items, one item per tag. Turn
// each into a bookmark (one per name and address) tagged with the same tags.
func migrateLinks() {
	var links []models.Item
	err := db.Table("items").Where("\"KIND\" = ?", "link").Order("\"ID\"").Find(&links).Error
	check.IfError(err)

	made := make(map[[2]string]int)
	for _, it := range links {
		key := [2]string{it.Name, it.URL}
		id, ok := made[key]
		if !ok {
			b := models.Bookmark{Name: it.Name, URL: it.URL, Icon: it.Icon}
			if err := db.Table("bookmarks").Create(&b).Error; err != nil {
				check.IfError(err)
				continue
			}
			id = b.ID
			made[key] = id
		}
		err := db.Table("items").Where("\"ID\" = ?", it.ID).Updates(map[string]any{
			"KIND": "bookmark", "BOOKMARK": id, "NAME": "", "URL": "", "ICON": "",
		}).Error
		check.IfError(err)
	}
}
