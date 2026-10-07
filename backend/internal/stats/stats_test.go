package stats

import (
	"testing"
	"time"

	"github.com/vipthomps/gimle/backend/internal/models"
)

func TestRange(t *testing.T) {
	now := time.Date(2026, 10, 6, 18, 30, 0, 0, time.UTC)
	day, hour, bucket := Range(now, 1)
	if day != "2026-10-06" || hour != "2026-10-06 00" || bucket != "hour" {
		t.Errorf("1 day: %s %s %s", day, hour, bucket)
	}
	day, _, bucket = Range(now, 30)
	if day != "2026-09-07" || bucket != "day" {
		t.Errorf("30 days: %s %s", day, bucket)
	}
}

func TestTrends(t *testing.T) {
	hours := []models.SubnetHour{
		{Hour: "2026-10-05 10", SubnetID: 1, Samples: 2, OnlineSum: 10, UsedSum: 20, Total: 254},
		{Hour: "2026-10-05 11", SubnetID: 1, Samples: 2, OnlineSum: 6, UsedSum: 20, Total: 254},
		{Hour: "2026-10-06 09", SubnetID: 1, Samples: 1, OnlineSum: 4, UsedSum: 12, Total: 126},
		{Hour: "2026-10-06 09", SubnetID: 0, Samples: 1, OnlineSum: 9, UsedSum: 30},
	}
	byHour := Trends(hours, "hour")
	if len(byHour[1]) != 3 || byHour[1][0].Online != 5 || byHour[1][1].Online != 3 {
		t.Errorf("hourly: %+v", byHour[1])
	}
	byDay := Trends(hours, "day")
	p := byDay[1]
	if len(p) != 2 || p[0].Time != "2026-10-05" || p[0].Online != 4 || p[0].Used != 10 || p[1].Total != 126 {
		t.Errorf("daily: %+v", p)
	}
	if len(byDay[0]) != 1 || byDay[0][0].Online != 9 {
		t.Errorf("all hosts: %+v", byDay[0])
	}
}

func TestUptime(t *testing.T) {
	u := Uptime(models.Host{Mac: "m"}, models.FirstSeen{}, nil)
	if u.Uptime != -1 {
		t.Errorf("no scans: %v", u.Uptime)
	}
	u = Uptime(models.Host{Mac: "m"}, models.FirstSeen{Date: "x", Seeded: true}, []models.HostDay{{Date: "a", Scans: 3, Online: 3}, {Date: "b", Scans: 1, Online: 0}})
	if u.Uptime != 75 || len(u.Days) != 2 || !u.Before {
		t.Errorf("uptime: %+v", u)
	}
}

func TestNewPerDay(t *testing.T) {
	got := NewPerDay([]models.FirstSeen{
		{Mac: "a", Date: "2026-10-01 10:00:00"},
		{Mac: "b", Date: "2026-10-03 10:00:00"},
		{Mac: "c", Date: "2026-10-03 12:00:00"},
		{Mac: "d", Date: ""},
		{Mac: "e", Date: "2026-10-04 09:00:00", Seeded: true},
	}, "2026-10-02")
	if len(got) != 1 || got[0].Date != "2026-10-03" || got[0].Count != 2 {
		t.Errorf("new per day: %+v", got)
	}
}
