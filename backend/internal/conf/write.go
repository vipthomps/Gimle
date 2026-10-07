package conf

import (
	"log/slog"

	"github.com/spf13/viper"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

// Write - write config to file
func Write(config models.Conf) {

	slog.Info("Writing new config to " + config.ConfPath)

	viper.SetConfigFile(config.ConfPath)
	viper.SetConfigType("yaml")

	viper.Set("HOST", config.Host)
	viper.Set("PORT", config.Port)
	viper.Set("THEME", config.Theme)
	viper.Set("COLOR", config.Color)
	viper.Set("NODEPATH", config.NodePath)
	viper.Set("LOG_LEVEL", config.LogLevel)
	viper.Set("ARP_ARGS", config.ArpArgs)
	viper.Set("ARP_STRS", config.ArpStrs)
	viper.Set("ARP_STRS_JOINED", "") // Can be set only with ENV
	viper.Set("IFACES", config.Ifaces)
	viper.Set("TIMEOUT", config.Timeout)
	viper.Set("TRIM_HIST", config.TrimHist)
	viper.Set("STATS_DAYS", config.StatsDays)
	viper.Set("SHOUTRRR_URL", config.ShoutURL)

	viper.Set("USE_DB", config.UseDB)
	viper.Set("PG_CONNECT", config.PGConnect)

	viper.Set("influx_enable", config.InfluxEnable)
	viper.Set("influx_skip_tls", config.InfluxSkipTLS)
	viper.Set("influx_addr", config.InfluxAddr)
	viper.Set("influx_token", config.InfluxToken)
	viper.Set("influx_org", config.InfluxOrg)
	viper.Set("influx_bucket", config.InfluxBucket)

	viper.Set("PROMETHEUS_ENABLE", config.PrometheusEnable)

	viper.Set("PORT_SCAN", config.PortScan)
	viper.Set("PORT_LIST", config.PortList)
	viper.Set("PORT_INTERVAL", config.PortInterval)
	viper.Set("PORT_WORKERS", config.PortWorkers)
	viper.Set("PORT_TIMEOUT", config.PortTimeout)

	viper.Set("DNS_SERVER", config.DNSServer)
	viper.Set("NAME_MDNS", config.NameMDNS)
	viper.Set("NAME_NETBIOS", config.NameNetBIOS)

	viper.Set("START_PAGE", config.StartPage)

	viper.Set("DOCS_REPO", config.DocsRepo)
	viper.Set("DOCS_BRANCH", config.DocsBranch)
	viper.Set("DOCS_DIR", config.DocsDir)
	viper.Set("DOCS_EDIT", config.DocsEdit)
	viper.Set("DOCS_API", config.DocsAPI)
	if config.DocsTokenEnv {
		viper.Set("DOCS_TOKEN", "") // a token from the environment stays out of the file
	} else {
		viper.Set("DOCS_TOKEN", config.DocsToken)
	}

	err := viper.WriteConfig()
	check.IfError(err)
}
