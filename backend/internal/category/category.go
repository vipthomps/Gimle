// Package category suggests a dashboard group for a host or service from
// what is known about it: page titles, service and container names, images,
// open ports and the hardware vendor. Suggestions are only a starting point;
// tags set by hand always win.
package category

import (
	"strings"
)

// Categories in the order they are shown
var Order = []string{
	"Network", "Virtualization", "Containers", "Storage", "Media", "Downloads",
	"Home automation", "Monitoring", "Security", "Databases", "Development",
	"Productivity", "AI", "Printers", "Cameras", "Computers", "Phones and tablets",
	"Other",
}

type rule struct {
	cat   string
	words []string
}

// Checked in order, so more specific rules come first
var rules = []rule{
	{"Virtualization", []string{"proxmox", "pve", "esxi", "vmware", "xcp-ng", "xen orchestra", "truenas scale apps", "incus", "libvirt", "hyper-v", "qemu"}},
	{"Containers", []string{"portainer", "dockhand", "dockge", "docker api", "komodo", "yacht", "kubernetes", "k3s", "rancher", "watchtower", "traefik dashboard"}},
	{"Network", []string{"pi-hole", "pihole", "adguard", "unbound", "bind9", "technitium", "dns", "dhcp", "opnsense", "pfsense", "openwrt", "unifi", "ubiquiti", "mikrotik", "routeros", "omada", "tp-link", "netgear", "router", "switch", "access point", "firewall", "wireguard", "openvpn", "tailscale", "headscale", "nginx proxy manager", "traefik", "caddy", "haproxy", "cloudflared", "scanopy", "netbox", "upnp", "snmp", "fortinet", "fortigate", "fortiswitch", "fortiap", "arista", "cisco", "juniper", "aruba", "eero", "asus router", "gateway"}},
	{"Storage", []string{"truenas", "freenas", "synology", "qnap", "unraid", "nextcloud", "owncloud", "seafile", "syncthing", "minio", "samba", "smb", "nfs", "afp", "rsync", "backup", "proxmox backup", "duplicati", "restic", "kopia", "filebrowser", "openmediavault", "nas"}},
	{"Media", []string{"plex", "jellyfin", "emby", "immich", "photoprism", "navidrome", "audiobookshelf", "kavita", "komga", "calibre", "tautulli", "overseerr", "jellyseerr", "ombi", "roku", "sonos", "chromecast", "apple tv", "fire tv", "shield", "kodi", "dlna", "rtsp", "tv"}},
	{"Downloads", []string{"sonarr", "radarr", "lidarr", "readarr", "prowlarr", "bazarr", "jackett", "qbittorrent", "transmission", "deluge", "sabnzbd", "nzbget", "slskd"}},
	{"Home automation", []string{"home assistant", "homeassistant", "hass", "node-red", "nodered", "zigbee2mqtt", "zwave", "z-wave", "esphome", "mosquitto", "mqtt", "homebridge", "hubitat", "openhab", "frigate", "philips hue", "hue bridge", "hue", "shelly", "tasmota", "espressif", "tuya", "ecobee", "nest", "ring", "thermostat", "doorbell", "smart plug", "plug", "sensor", "lutron", "sonoff"}},
	{"Monitoring", []string{"grafana", "prometheus", "influxdb", "uptime kuma", "uptime-kuma", "netdata", "zabbix", "checkmk", "librenms", "loki", "graylog", "glances", "node exporter", "cadvisor", "scrutiny", "healthchecks", "gotify", "ntfy", "cockpit", "webmin", "dashy", "homepage", "homarr", "heimdall", "organizr", "gimle"}},
	{"Security", []string{"vaultwarden", "bitwarden", "authentik", "authelia", "keycloak", "crowdsec", "wazuh", "step-ca", "ldap", "kerberos", "kanidm", "pocket-id"}},
	{"Databases", []string{"postgres", "postgresql", "mysql", "mariadb", "mongodb", "redis", "valkey", "ms sql", "sql server", "couchdb", "qdrant", "elasticsearch", "opensearch", "clickhouse", "pgadmin", "adminer", "surrealdb", "influx"}},
	{"Development", []string{"gitea", "forgejo", "gitlab", "jenkins", "drone", "woodpecker", "code-server", "vscode", "harbor", "registry", "sonarqube", "n8n"}},
	{"AI", []string{"ollama", "open webui", "open-webui", "openwebui", "localai", "comfyui", "stable diffusion", "automatic1111", "litellm", "open notebook", "librechat", "anythingllm"}},
	{"Productivity", []string{"paperless", "bookstack", "wiki", "outline", "trilium", "joplin", "memos", "mealie", "tandoor", "grocy", "firefly", "actual", "linkding", "wallabag", "freshrss", "miniflux", "vikunja", "nocodb", "baserow", "stirling", "onlyoffice", "collabora", "mkdocs", "docs"}},
	{"Printers", []string{"printer", "ipp printing", "cups", "brother", "epson", "canon", "lexmark", "kyocera", "xerox", "ricoh", "octoprint", "klipper", "mainsail", "fluidd", "bambu"}},
	{"Cameras", []string{"camera", "hikvision", "dahua", "reolink", "amcrest", "axis communications", "blue iris", "shinobi", "zoneminder", "scrypted", "unifi protect"}},
	{"Phones and tablets", []string{"iphone", "ipad", "android", "pixel", "galaxy", "oneplus", "tablet", "phone", "private mac"}},
	{"Computers", []string{"rdp", "vnc", "ssh", "desktop", "laptop", "macbook", "imac", "mac mini", "osx", "macos", "windows", "workstation", "netbios", "dell", "lenovo", "hewlett", "hp inc", "intel corporate", "asustek", "micro-star", "gigabyte", "raspberry pi", "apple"}},
}

// Suggest - the category whose words appear first in the given texts, most
// telling text first (a page title before a vendor name). "" if nothing matches.
func Suggest(texts ...string) string {
	for _, t := range texts {
		if c := match(t); c != "" {
			return c
		}
	}
	return ""
}

// separators - "home-assistant", "immich_server" and "ghcr.io/x" read as words
var separators = strings.NewReplacer("-", " ", "_", " ", ".", " ", "/", " ", ":", " ")

func match(text string) string {
	t := " " + separators.Replace(strings.ToLower(text)) + " "
	if strings.TrimSpace(t) == "" {
		return ""
	}
	for _, r := range rules {
		for _, w := range r.words {
			if containsWord(t, separators.Replace(w)) {
				return r.cat
			}
		}
	}
	return ""
}

// containsWord - w appears in t, not inside a longer word: "tv" is in "living room tv" but not in
// "tvheadend". Digits may follow, so "dns" is in "dns3".
func containsWord(t, w string) bool {
	for i := 0; ; {
		j := strings.Index(t[i:], w)
		if j < 0 {
			return false
		}
		j += i
		before, after := t[j-1], byte(' ')
		if j+len(w) < len(t) {
			after = t[j+len(w)]
		}
		if !isWordByte(before) && !(after >= 'a' && after <= 'z') {
			return true
		}
		i = j + 1
	}
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}
