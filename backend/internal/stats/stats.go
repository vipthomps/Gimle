// Package stats turns the scan counters into uptime and utilization figures.
package stats

import (
	"sort"
	"time"

	"github.com/vipthomps/gimle/backend/internal/models"
)

// Range - the first day and hour included when looking back a number of days
// (today counts as one day), and whether points are per hour or per day
func Range(now time.Time, days int) (fromDay, fromHour, bucket string) {
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1-days)
	bucket = "day"
	if days <= 7 {
		bucket = "hour"
	}
	return from.Format("2006-01-02"), from.Format("2006-01-02 15"), bucket
}

// Trends - average online and used counts per bucket, by subnet ID (0 = all hosts)
func Trends(hours []models.SubnetHour, bucket string) map[int][]models.TrendPoint {
	type acc struct {
		samples, online, used, total int
	}
	sums := make(map[int]map[string]*acc)
	for _, h := range hours {
		key := h.Hour
		if bucket == "day" && len(key) >= 10 {
			key = key[:10]
		}
		if sums[h.SubnetID] == nil {
			sums[h.SubnetID] = make(map[string]*acc)
		}
		a := sums[h.SubnetID][key]
		if a == nil {
			a = &acc{}
			sums[h.SubnetID][key] = a
		}
		a.samples += h.Samples
		a.online += h.OnlineSum
		a.used += h.UsedSum
		a.total = h.Total // hours come in order, so the last one wins
	}

	res := make(map[int][]models.TrendPoint)
	for id, byKey := range sums {
		keys := make([]string, 0, len(byKey))
		for k := range byKey {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			a := byKey[k]
			if a.samples == 0 {
				continue
			}
			res[id] = append(res[id], models.TrendPoint{
				Time:   k,
				Online: float64(a.online) / float64(a.samples),
				Used:   float64(a.used) / float64(a.samples),
				Total:  a.total,
			})
		}
	}
	return res
}

// Uptime - a host's uptime from its daily counters
func Uptime(h models.Host, first models.FirstSeen, days []models.HostDay) models.HostUptime {
	u := models.HostUptime{Host: h, FirstSeen: first.Date, Before: first.Seeded, Uptime: -1, Days: []models.DayUptime{}}
	for _, d := range days {
		u.Scans += d.Scans
		u.Online += d.Online
		u.Days = append(u.Days, models.DayUptime{Date: d.Date, Scans: d.Scans, Online: d.Online})
	}
	if u.Scans > 0 {
		u.Uptime = float64(u.Online) * 100 / float64(u.Scans)
	}
	return u
}

// NewPerDay - how many hosts were first seen on each day from a day on.
// Hosts known before stats existed are not new
func NewPerDay(list []models.FirstSeen, fromDay string) []models.DayCount {
	counts := make(map[string]int)
	for _, f := range list {
		if f.Seeded || len(f.Date) < 10 || f.Date[:10] < fromDay {
			continue
		}
		counts[f.Date[:10]]++
	}
	res := []models.DayCount{}
	for d, n := range counts {
		res = append(res, models.DayCount{Date: d, Count: n})
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Date < res[j].Date })
	return res
}
