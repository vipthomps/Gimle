package gdb

import (
	"gorm.io/gorm"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// RecordScan - add one scan to the daily host counters and hourly subnet sums
func RecordScan(days []models.HostDay, hours []models.SubnetHour) {
	err := db.Transaction(func(tx *gorm.DB) error {
		for _, d := range days {
			var old models.HostDay
			res := tx.Table("host_days").Where("\"DATE\" = ? AND \"MAC\" = ?", d.Date, d.Mac).Limit(1).Find(&old)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				if err := tx.Table("host_days").Create(&d).Error; err != nil {
					return err
				}
				continue
			}
			err := tx.Table("host_days").Where("\"DATE\" = ? AND \"MAC\" = ?", d.Date, d.Mac).
				Updates(map[string]any{"SCANS": old.Scans + d.Scans, "ONLINE": old.Online + d.Online}).Error
			if err != nil {
				return err
			}
		}
		for _, h := range hours {
			var old models.SubnetHour
			res := tx.Table("subnet_hours").Where("\"HOUR\" = ? AND \"SUBNET_ID\" = ?", h.Hour, h.SubnetID).Limit(1).Find(&old)
			if res.Error != nil {
				return res.Error
			}
			// Not Save: SubnetID 0 is a zero primary key, which gorm treats as a new row
			if res.RowsAffected == 0 {
				if err := tx.Table("subnet_hours").Create(&h).Error; err != nil {
					return err
				}
				continue
			}
			err := tx.Table("subnet_hours").Where("\"HOUR\" = ? AND \"SUBNET_ID\" = ?", h.Hour, h.SubnetID).
				Updates(map[string]any{"SAMPLES": old.Samples + h.Samples, "ONLINE_SUM": old.OnlineSum + h.OnlineSum,
					"USED_SUM": old.UsedSum + h.UsedSum, "TOTAL": h.Total}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	check.IfError(err)
}

// SelectHostDays - daily host counters from a date on
func SelectHostDays(from string) (days []models.HostDay) {
	err := db.Table("host_days").Where("\"DATE\" >= ?", from).Order("\"DATE\"").Find(&days).Error
	check.IfError(err)
	return days
}

// SelectHostDaysByMAC - one host's daily counters from a date on
func SelectHostDaysByMAC(mac, from string) (days []models.HostDay) {
	err := db.Table("host_days").Where("\"MAC\" = ? AND \"DATE\" >= ?", mac, from).Order("\"DATE\"").Find(&days).Error
	check.IfError(err)
	return days
}

// SelectSubnetHours - hourly subnet sums from an hour on
func SelectSubnetHours(from string) (hours []models.SubnetHour) {
	err := db.Table("subnet_hours").Where("\"HOUR\" >= ?", from).Order("\"HOUR\"").Find(&hours).Error
	check.IfError(err)
	return hours
}

// SelectFirstSeen - when each host was first discovered
func SelectFirstSeen() (list []models.FirstSeen) {
	err := db.Table("first_seen").Find(&list).Error
	check.IfError(err)
	return list
}

// AddFirstSeen - record a host's first discovery, keeping an existing date.
// seeded marks hosts that were known before stats existed
func AddFirstSeen(mac, date string, seeded bool) {
	var n int64
	db.Table("first_seen").Where("\"MAC\" = ?", mac).Count(&n)
	if n == 0 {
		err := db.Table("first_seen").Create(&models.FirstSeen{Mac: mac, Date: date, Seeded: seeded}).Error
		check.IfError(err)
	}
}

// OldestHistory - the earliest history date for a host, "" if none
func OldestHistory(mac string) string {
	var h models.Host
	res := db.Table("history").Where("\"MAC\" = ?", mac).Order("\"DATE\"").Limit(1).Find(&h)
	if res.Error != nil || res.RowsAffected == 0 {
		return ""
	}
	return h.Date
}

// DeleteStatsBefore - drop daily host counters before a day and subnet sums before an hour
func DeleteStatsBefore(day, hour string) {
	err := db.Table("host_days").Where("\"DATE\" < ?", day).Delete(&models.HostDay{}).Error
	check.IfError(err)
	err = db.Table("subnet_hours").Where("\"HOUR\" < ?", hour).Delete(&models.SubnetHour{}).Error
	check.IfError(err)
}

// DeleteStatsByMAC - remove a deleted host's stats
func DeleteStatsByMAC(mac string) {
	err := db.Table("host_days").Where("\"MAC\" = ?", mac).Delete(&models.HostDay{}).Error
	check.IfError(err)
	err = db.Table("first_seen").Where("\"MAC\" = ?", mac).Delete(&models.FirstSeen{}).Error
	check.IfError(err)
}
