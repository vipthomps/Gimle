package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
)

func TestDue(t *testing.T) {
	at := func(s string) time.Time {
		v, _ := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
		return v
	}
	cases := []struct {
		name string
		have []Backup
		now  string
		want []string
	}{
		{"nothing yet, before 3am", nil, "2026-10-07 01:00", []string{Auto}},
		{"nothing yet, after 3am", nil, "2026-10-07 09:00", []string{Auto, Nightly}},
		{"recent of both", []Backup{{Kind: Auto, Time: at("2026-10-07 08:00")}, {Kind: Nightly, Time: at("2026-10-07 03:05")}}, "2026-10-07 11:59", nil},
		{"four hours on", []Backup{{Kind: Auto, Time: at("2026-10-07 08:00")}, {Kind: Nightly, Time: at("2026-10-07 03:05")}}, "2026-10-07 12:00", []string{Auto}},
		{"next night", []Backup{{Kind: Auto, Time: at("2026-10-08 02:00")}, {Kind: Nightly, Time: at("2026-10-07 03:05")}}, "2026-10-08 03:00", []string{Nightly}},
		{"manual ones don't count", []Backup{{Kind: Manual, Time: at("2026-10-07 08:00")}}, "2026-10-07 08:30", []string{Auto, Nightly}},
	}
	for _, c := range cases {
		got := due(c.have, at(c.now))
		if len(got) != len(c.want) || (len(got) > 0 && got[0] != c.want[0]) || (len(got) > 1 && got[1] != c.want[1]) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestCreateRestorePrune(t *testing.T) {
	dir := t.TempDir()
	conf.Start(dir, "")
	gdb.Start()
	defer gdb.Close()

	view := models.View{Name: "Before"}
	if err := gdb.SaveView(&view); err != nil {
		t.Fatal(err)
	}
	conf.AppConfig.StartPage = "/stats"
	conf.Write(conf.AppConfig)

	saved, err := Create(Manual)
	if err != nil {
		t.Fatal(err)
	}

	// someone renames the view and changes the start page
	view.Name = "Vandalised"
	if err := gdb.SaveView(&view); err != nil {
		t.Fatal(err)
	}
	conf.AppConfig.StartPage = "/hosts"
	conf.AppConfig.Port = "9999" // the port in use survives a restore
	conf.Write(conf.AppConfig)

	if err := Restore(saved.Name); err != nil {
		t.Fatal(err)
	}
	views := gdb.SelectViews()
	if len(views) != 1 || views[0].Name != "Before" {
		t.Errorf("views after restore: %+v", views)
	}
	if conf.AppConfig.StartPage != "/stats" || conf.AppConfig.Port != "9999" {
		t.Errorf("config after restore: start %q port %q", conf.AppConfig.StartPage, conf.AppConfig.Port)
	}

	all, _ := List()
	var pre int
	for _, b := range all {
		if b.Kind == PreRestore {
			pre++
		}
	}
	if pre != 1 {
		t.Errorf("want one backup taken before the restore, got %d of %d", pre, len(all))
	}

	// each kind keeps its own allowance
	for i := 0; i < keep[Manual]+3; i++ {
		b, err := Create(Manual)
		if err != nil {
			t.Fatal(err)
		}
		// names are per second; spread them out so each is distinct
		old := filepath.Join(Dir(), b.Name)
		renamed := filepath.Join(Dir(), "gimle-manual-"+time.Now().Add(-time.Duration(i+1)*time.Hour).Format(stamp)+".zip")
		os.Rename(old, renamed)
	}
	Create(Manual)
	all, _ = List()
	var manual int
	for _, b := range all {
		if b.Kind == Manual {
			manual++
		}
	}
	if manual != keep[Manual] {
		t.Errorf("manual backups kept: %d, want %d", manual, keep[Manual])
	}
	if pre != 1 {
		t.Errorf("pruning manual backups touched others")
	}

	if err := Restore("../scan.db"); err == nil {
		t.Errorf("restore accepted a path outside the backups")
	}
}
