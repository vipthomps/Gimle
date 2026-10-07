package routines

import (
	"time"

	"github.com/vipthomps/gimle/backend/internal/arp"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/influx"
	"github.com/vipthomps/gimle/backend/internal/models"
	"github.com/vipthomps/gimle/backend/internal/names"
	"github.com/vipthomps/gimle/backend/internal/notify"
	"github.com/vipthomps/gimle/backend/internal/prometheus"
)

func startScan(quit chan bool) {
	var lastDate, nowDate, plusDate time.Time
	var foundHosts []models.Host

	for {
		select {
		case <-quit:
			return
		default:
			nowDate = time.Now()
			plusDate = lastDate.Add(time.Duration(conf.AppConfig.Timeout) * time.Second)

			if nowDate.After(plusDate) {

				foundHosts = arp.Scan(conf.AppConfig.Ifaces, conf.AppConfig.ArpArgs, conf.AppConfig.ArpStrs, gdb.SelectSubnets())

				// Make map of found hosts
				foundHostsMap := make(map[string]models.Host)
				for _, fHost := range foundHosts {
					foundHostsMap[fHost.Mac] = fHost
				}

				compareHosts(foundHostsMap)

				lastDate = time.Now()
			}

			time.Sleep(time.Duration(1) * time.Minute)
		}
	}
}

func compareHosts(foundHostsMap map[string]models.Host) {

	allHosts, ok := gdb.Select("now")
	if !ok {
		return
	}

	scanned := make([]models.Host, 0, len(allHosts)+len(foundHostsMap))

	for _, aHost := range allHosts {

		fHost, exists := foundHostsMap[aHost.Mac]
		if exists {

			aHost.Iface = fHost.Iface
			aHost.IP = fHost.IP
			aHost.Date = fHost.Date
			aHost.Now = 1

			delete(foundHostsMap, aHost.Mac)

		} else {
			aHost.Now = 0
		}
		gdb.Update("now", aHost)
		scanned = append(scanned, aHost)

		aHost.ID = 0
		aHost.Date = time.Now().Format("2006-01-02 15:04:05")
		gdb.Update("history", aHost)

		if conf.AppConfig.InfluxEnable {
			influx.Add(conf.AppConfig, aHost)
		}
		if conf.AppConfig.PrometheusEnable {
			prometheus.Add(aHost)
		}
	}

	for _, fHost := range foundHostsMap {

		if fHost.DNS = names.Reverse(fHost.IP, conf.AppConfig.DNSServer, time.Second); fHost.DNS != "" {
			fHost.Name = names.Short(fHost.DNS)
		}
		notify.Unknown(fHost) // Log and Shoutrrr

		fHost = gdb.Update("now", fHost)
		gdb.AddFirstSeen(fHost.Mac, time.Now().Format("2006-01-02 15:04:05"), false)

		fHost.Now = 1
		scanned = append(scanned, fHost)
	}

	recordStats(scanned, time.Now())
	resolveNames(scanned)
}
