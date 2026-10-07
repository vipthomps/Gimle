package conf

import (
	"os"
	"slices"

	"strings"

	"github.com/spf13/viper"

	"github.com/vipthomps/gimle/backend/internal/check"
	"github.com/vipthomps/gimle/backend/internal/models"
)

func read(path string) (config models.Conf) {

	viper.Reset() // values set by an earlier Write would otherwise hide the file's

	viper.SetDefault("HOST", "0.0.0.0")
	viper.SetDefault("PORT", "8840")
	viper.SetDefault("THEME", "gold")
	viper.SetDefault("COLOR", "dark")
	viper.SetDefault("NODEPATH", "")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("ARP_ARGS", "")
	viper.SetDefault("ARP_STRS_JOINED", "")
	viper.SetDefault("IFACES", "")
	viper.SetDefault("TIMEOUT", 120)
	viper.SetDefault("TRIM_HIST", 48)
	viper.SetDefault("SHOUTRRR_URL", "")

	viper.SetDefault("USE_DB", "sqlite")
	viper.SetDefault("PG_CONNECT", "")

	viper.SetDefault("INFLUX_ENABLE", false)

	viper.SetDefault("PROMETHEUS_ENABLE", false)

	viper.SetDefault("PORT_SCAN", false)
	viper.SetDefault("PORT_LIST", "top")
	viper.SetDefault("PORT_INTERVAL", 360)
	viper.SetDefault("PORT_WORKERS", 64)
	viper.SetDefault("PORT_TIMEOUT", 700)

	viper.SetDefault("STATS_DAYS", 90)

	viper.SetDefault("DNS_SERVER", "")
	viper.SetDefault("NAME_MDNS", true)
	viper.SetDefault("NAME_NETBIOS", true)

	viper.SetDefault("DOCS_REPO", "")
	viper.SetDefault("DOCS_BRANCH", "")
	viper.SetDefault("DOCS_DIR", "")
	viper.SetDefault("DOCS_TOKEN", "")
	viper.SetDefault("DOCS_EDIT", false)
	viper.SetDefault("DOCS_API", "")

	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	check.IfError(err)

	viper.AutomaticEnv() // Get ENVIRONMENT variables

	config.Host = viper.Get("HOST").(string)
	config.Port = viper.Get("PORT").(string)
	config.Theme = viper.Get("THEME").(string)
	if !slices.Contains(accents, config.Theme) {
		config.Theme = "gold" // a bootswatch theme name from an older config
	}
	config.Color = viper.Get("COLOR").(string)
	config.NodePath = viper.Get("NODEPATH").(string)
	config.LogLevel = viper.Get("LOG_LEVEL").(string)
	config.ArpArgs = viper.Get("ARP_ARGS").(string)
	config.ArpStrs = viper.GetStringSlice("ARP_STRS")
	config.Ifaces = viper.Get("IFACES").(string)
	config.Timeout = viper.GetInt("TIMEOUT")
	config.TrimHist = viper.GetInt("TRIM_HIST")
	config.StatsDays = viper.GetInt("STATS_DAYS")
	config.ShoutURL = viper.Get("SHOUTRRR_URL").(string)

	config.UseDB = viper.Get("USE_DB").(string)
	config.PGConnect = viper.Get("PG_CONNECT").(string)

	config.InfluxEnable = viper.GetBool("INFLUX_ENABLE")
	config.InfluxSkipTLS = viper.GetBool("INFLUX_SKIP_TLS")
	config.InfluxAddr, _ = viper.Get("INFLUX_ADDR").(string)
	config.InfluxToken, _ = viper.Get("INFLUX_TOKEN").(string)
	config.InfluxOrg, _ = viper.Get("INFLUX_ORG").(string)
	config.InfluxBucket, _ = viper.Get("INFLUX_BUCKET").(string)

	config.PrometheusEnable = viper.GetBool("PROMETHEUS_ENABLE")

	config.PortScan = viper.GetBool("PORT_SCAN")
	config.PortList = viper.GetString("PORT_LIST")
	config.PortInterval = viper.GetInt("PORT_INTERVAL")
	config.PortWorkers = viper.GetInt("PORT_WORKERS")
	config.PortTimeout = viper.GetInt("PORT_TIMEOUT")

	config.DNSServer = viper.GetString("DNS_SERVER")
	config.NameMDNS = viper.GetBool("NAME_MDNS")
	config.NameNetBIOS = viper.GetBool("NAME_NETBIOS")
	config.StartPage = viper.GetString("START_PAGE")

	config.DocsRepo = viper.GetString("DOCS_REPO")
	config.DocsBranch = viper.GetString("DOCS_BRANCH")
	config.DocsDir = viper.GetString("DOCS_DIR")
	config.DocsToken = viper.GetString("DOCS_TOKEN")
	config.DocsEdit = viper.GetBool("DOCS_EDIT")
	config.DocsAPI = viper.GetString("DOCS_API")
	config.DocsHasToken = config.DocsToken != ""
	config.DocsTokenEnv = os.Getenv("DOCS_TOKEN") != ""

	joined := viper.Get("ARP_STRS_JOINED").(string)
	// slog.Info("ARP_STRS_JOINED: " + joined)

	if joined != "" {
		config.ArpStrs = strings.Split(joined, ",")
	}

	return config
}

// accents - the accent colours Settings offers
var accents = []string{"gold", "copper", "sea", "moss", "iris"}
