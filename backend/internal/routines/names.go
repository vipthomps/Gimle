package routines

import (
	"log/slog"
	"sync"
	"time"

	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/names"
)

const (
	nameRetry   = time.Hour // wait this long before asking again for a host with no name
	nameWorkers = 16
)

var (
	namesMu   sync.Mutex
	nameTried = make(map[string]time.Time) // MAC -> last failed lookup
	namesBusy bool
)

// NameOptions - lookups to try, from the config
func NameOptions() names.Options {
	return names.Options{
		DNSServer: conf.AppConfig.DNSServer,
		MDNS:      conf.AppConfig.NameMDNS,
		NetBIOS:   conf.AppConfig.NameNetBIOS,
		Timeout:   time.Second,
	}
}

// RetryNames - forget failed lookups, so the next scan asks every unnamed host again
func RetryNames() {
	namesMu.Lock()
	nameTried = make(map[string]time.Time)
	namesMu.Unlock()
}

// resolveNames - look up names for online hosts that have none, in the
// background. A host that gave no name is asked again after nameRetry.
func resolveNames(hosts []models.Host) {
	now := time.Now()

	namesMu.Lock()
	if namesBusy {
		namesMu.Unlock()
		return
	}
	var todo []models.Host
	for _, h := range hosts {
		if h.Now != 1 || h.DNS != "" || h.IP == "" {
			continue
		}
		if t, ok := nameTried[h.Mac]; ok && now.Sub(t) < nameRetry {
			continue
		}
		nameTried[h.Mac] = now
		todo = append(todo, h)
	}
	if len(todo) == 0 {
		namesMu.Unlock()
		return
	}
	namesBusy = true
	namesMu.Unlock()

	go func() {
		defer func() {
			namesMu.Lock()
			namesBusy = false
			namesMu.Unlock()
		}()
		lookupAll(todo, NameOptions())
	}()
}

func lookupAll(hosts []models.Host, o names.Options) {
	ch := make(chan models.Host)
	var wg sync.WaitGroup
	for range nameWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for h := range ch {
				name, source := names.Lookup(h.IP, o)
				if name == "" {
					continue
				}
				gdb.SetName(h.ID, name, h.Name == "")
				namesMu.Lock()
				delete(nameTried, h.Mac)
				namesMu.Unlock()
				slog.Debug("Found host name", "ip", h.IP, "name", name, "source", source)
			}
		}()
	}
	for _, h := range hosts {
		ch <- h
	}
	close(ch)
	wg.Wait()
}
