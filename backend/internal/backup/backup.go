// Package backup keeps rolling copies of the database and settings inside the
// data folder, so a change anyone on the network made (Gimle has no login) can
// be rolled back from Settings without restoring the whole container.
package backup

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
)

// Kinds of backup and how many of each are kept. Each kind has its own
// allowance, so pressing "Back up now" many times never pushes out the
// automatic ones.
const (
	Auto       = "auto"       // every 4 hours
	Nightly    = "nightly"    // once a day
	Manual     = "manual"     // Back up now
	PreRestore = "prerestore" // taken just before a restore, so the restore can be undone
)

var keep = map[string]int{Auto: 6, Nightly: 7, Manual: 10, PreRestore: 5}

const (
	autoEvery   = 4 * time.Hour
	nightlyHour = 3 // local time
	stamp       = "20060102-150405"
	dbName      = "scan.db"
	confName    = "config_v2.yaml"
)

var nameRe = regexp.MustCompile(`^gimle-(auto|nightly|manual|prerestore)-(\d{8}-\d{6})\.zip$`)

// Backup - one saved copy
type Backup struct {
	Name string
	Kind string
	Time time.Time
	Size int64
}

var mu sync.Mutex // one backup or restore at a time

// Dir - where backups are kept
func Dir() string {
	return filepath.Join(conf.AppConfig.DirPath, "backups")
}

// List - every backup, newest first
func List() ([]Backup, error) {
	entries, err := os.ReadDir(Dir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Backup
	for _, e := range entries {
		m := nameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		t, err := time.ParseInLocation(stamp, m[2], time.Local)
		if err != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Backup{Name: e.Name(), Kind: m[1], Time: t, Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.After(out[j].Time) })
	return out, nil
}

// Create - save the database and settings now
func Create(kind string) (Backup, error) {
	mu.Lock()
	defer mu.Unlock()
	return create(kind)
}

func create(kind string) (Backup, error) {
	if _, ok := keep[kind]; !ok {
		return Backup{}, fmt.Errorf("unknown backup kind %q", kind)
	}
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return Backup{}, err
	}

	now := time.Now()
	name := fmt.Sprintf("gimle-%s-%s.zip", kind, now.Format(stamp))
	final := filepath.Join(Dir(), name)

	work, err := os.MkdirTemp(Dir(), ".work-")
	if err != nil {
		return Backup{}, err
	}
	defer os.RemoveAll(work)

	dbCopy := filepath.Join(work, dbName)
	if err := gdb.Snapshot(dbCopy); err != nil {
		return Backup{}, err
	}

	tmp := final + ".part"
	if err := writeZip(tmp, map[string]string{dbName: dbCopy, confName: conf.AppConfig.ConfPath}); err != nil {
		os.Remove(tmp)
		return Backup{}, err
	}
	if err := os.Rename(tmp, final); err != nil {
		return Backup{}, err
	}
	prune(kind)

	info, _ := os.Stat(final)
	b := Backup{Name: name, Kind: kind, Time: now.Truncate(time.Second)}
	if info != nil {
		b.Size = info.Size()
	}
	slog.Info("Backup saved", "name", name)
	return b, nil
}

func writeZip(path string, files map[string]string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for name, src := range files {
		in, err := os.Open(src)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			f.Close()
			return err
		}
		w, err := zw.Create(name)
		if err == nil {
			_, err = io.Copy(w, in)
		}
		in.Close()
		if err != nil {
			f.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// prune - drop the oldest backups of a kind beyond its allowance
func prune(kind string) {
	all, err := List()
	if err != nil {
		return
	}
	n := 0
	for _, b := range all {
		if b.Kind != kind {
			continue
		}
		n++
		if n > keep[kind] {
			_ = os.Remove(filepath.Join(Dir(), b.Name))
		}
	}
}

// Restore - put back the database and settings from a backup. The current
// state is saved first, so a restore can itself be undone. The address, port
// and node path in use are kept, so a restore never moves the web page.
func Restore(name string) error {
	if nameRe.FindStringSubmatch(name) == nil {
		return fmt.Errorf("no such backup")
	}
	mu.Lock()
	defer mu.Unlock()

	src := filepath.Join(Dir(), name)
	zr, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("could not open %s: %w", name, err)
	}
	defer zr.Close()

	var dbFile, confFile *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case dbName:
			dbFile = f
		case confName:
			confFile = f
		}
	}
	if dbFile == nil {
		return fmt.Errorf("%s holds no database", name)
	}

	if _, err := create(PreRestore); err != nil {
		return fmt.Errorf("could not save the current state first, so nothing was restored: %w", err)
	}

	dbPath := conf.AppConfig.DBPath
	if err := extract(dbFile, dbPath+".restore"); err != nil {
		return err
	}

	gdb.Close()
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(dbPath + suffix)
	}
	err = os.Rename(dbPath+".restore", dbPath)
	if err == nil && confFile != nil {
		err = restoreConfig(confFile)
	}
	gdb.Reopen()
	if err != nil {
		return err
	}
	slog.Info("Restored backup", "name", name)
	return nil
}

func extract(f *zip.File, path string) error {
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func restoreConfig(f *zip.File) error {
	cur := conf.AppConfig
	if err := extract(f, cur.ConfPath+".restore"); err != nil {
		return err
	}
	if err := os.Rename(cur.ConfPath+".restore", cur.ConfPath); err != nil {
		return err
	}
	conf.Start(cur.DirPath, "")
	conf.AppConfig.Host, conf.AppConfig.Port, conf.AppConfig.NodePath = cur.Host, cur.Port, cur.NodePath
	conf.AppConfig.Version = cur.Version
	conf.Write(conf.AppConfig)
	return nil
}

// due - which automatic backups are owed now, given the ones that exist
func due(all []Backup, now time.Time) []string {
	var lastAuto, lastNightly time.Time
	for _, b := range all {
		switch {
		case b.Kind == Auto && b.Time.After(lastAuto):
			lastAuto = b.Time
		case b.Kind == Nightly && b.Time.After(lastNightly):
			lastNightly = b.Time
		}
	}
	var out []string
	if now.Sub(lastAuto) >= autoEvery {
		out = append(out, Auto)
	}
	y, m, d := now.Date()
	ly, lm, ld := lastNightly.Date()
	if now.Hour() >= nightlyHour && (y != ly || m != lm || d != ld) {
		out = append(out, Nightly)
	}
	return out
}

// Start - take automatic backups: every 4 hours, keeping a day of them, and
// one each night after 03:00, keeping a week
func Start() {
	go func() {
		for {
			if gdb.UsingSQLite() {
				all, err := List()
				if err == nil {
					for _, kind := range due(all, time.Now()) {
						if _, err := Create(kind); err != nil {
							slog.Error("Backup failed", "kind", kind, "err", err)
						}
					}
				}
			}
			time.Sleep(5 * time.Minute)
		}
	}()
}
