package routines

import (
	"log/slog"
	"time"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/subnet"
)

// recordStats - add one scan to the uptime and utilization counters
func recordStats(hosts []models.Host, now time.Time) {
	day := now.Format("2006-01-02")
	hour := now.Format("2006-01-02 15")

	days := make([]models.HostDay, 0, len(hosts))
	all := models.SubnetHour{Hour: hour, SubnetID: 0, Samples: 1, UsedSum: len(hosts)}
	for _, h := range hosts {
		days = append(days, models.HostDay{Date: day, Mac: h.Mac, Scans: 1, Online: h.Now})
		all.OnlineSum += h.Now
	}

	hours := []models.SubnetHour{all}
	for _, s := range gdb.SelectSubnets() {
		ipam, err := subnet.Map(s, hosts)
		if err != nil {
			continue
		}
		st := ipam.Stat
		hours = append(hours, models.SubnetHour{
			Hour: hour, SubnetID: s.ID, Samples: 1,
			OnlineSum: st.Online, UsedSum: st.Total - st.Free, Total: st.Total,
		})
	}

	gdb.RecordScan(days, hours)
}

// SeedFirstSeen - give hosts found before stats existed a first-seen date
// from their oldest history, or their last-seen date
func SeedFirstSeen() {
	known := make(map[string]bool)
	for _, f := range gdb.SelectFirstSeen() {
		known[f.Mac] = true
	}
	hosts, _ := gdb.Select("now")
	for _, h := range hosts {
		if known[h.Mac] {
			continue
		}
		date := gdb.OldestHistory(h.Mac)
		if date == "" || (h.Date != "" && h.Date < date) {
			date = h.Date
		}
		gdb.AddFirstSeen(h.Mac, date, true)
	}
}

// StatsTrim - drop stats older than STATS_DAYS, once an hour
func StatsTrim() {
	go func() {
		for {
			days := conf.AppConfig.StatsDays
			if days > 0 {
				from := time.Now().AddDate(0, 0, -days)
				gdb.DeleteStatsBefore(from.Format("2006-01-02"), from.Format("2006-01-02 15"))
				slog.Debug("Removed stats before", "date", from.Format("2006-01-02"))
			}
			time.Sleep(time.Hour)
		}
	}()
}
