package models

// Conf - app config
type Conf struct {
	Host      string
	Port      string
	Theme     string
	Color     string
	DirPath   string
	ConfPath  string
	DBPath    string
	NodePath  string
	LogLevel  string
	Ifaces    string
	ArpArgs   string
	ArpStrs   []string
	Timeout   int
	TrimHist  int
	StatsDays int // days of uptime and utilization stats to keep
	ShoutURL  string
	Version   string
	// PostgreSQL
	UseDB     string
	PGConnect string
	// InfluxDB
	InfluxEnable  bool
	InfluxAddr    string
	InfluxToken   string
	InfluxOrg     string
	InfluxBucket  string
	InfluxSkipTLS bool
	// Prometheus
	PrometheusEnable bool
	// Port scanning
	PortScan     bool   // scheduled scans of online hosts
	PortList     string // "top", "all" or "22,80,8000-8100"
	PortInterval int    // minutes between scheduled scans
	PortWorkers  int    // ports checked at once per host
	PortTimeout  int    // milliseconds to wait for each port
	// Host names
	DNSServer   string // "" uses the system resolver; "ip" or "ip:port" asks that server
	NameMDNS    bool   // ask hosts for their own name over mDNS
	NameNetBIOS bool   // ask hosts for their own name over NetBIOS
	// Page shown at /: "" for the network map, or "/view/auto", "/view/ID", "/hosts", "/stats", "/bookmarks"
	StartPage string
	// Docs: Markdown pages read from a folder in a GitHub repository
	DocsRepo     string // owner/name
	DocsBranch   string // "" for the repository's default branch
	DocsDir      string // folder inside the repository, "" for the root
	DocsToken    string `json:"-"` // never sent to the browser
	DocsEdit     bool   // let Gimle commit edits with the token
	DocsAPI      string // GitHub API URL; only GitHub Enterprise needs it, set in the config file
	DocsHasToken bool   // a token is saved
	DocsTokenEnv bool   // the token comes from the DOCS_TOKEN environment variable
}

// Host - one host
type Host struct {
	ID    int    `gorm:"column:ID;primaryKey"`
	Name  string `gorm:"column:NAME"`
	DNS   string `gorm:"column:DNS"`
	Iface string `gorm:"column:IFACE"`
	IP    string `gorm:"column:IP"`
	Mac   string `gorm:"column:MAC"`
	Hw    string `gorm:"column:HW"`
	Date  string `gorm:"column:DATE"`
	Known int    `gorm:"column:KNOWN"`
	Now   int    `gorm:"column:NOW"`
}

// Stat - status
type Stat struct {
	Total   int
	Online  int
	Offline int
	Known   int
	Unknown int
}

// Subnet - one network to scan and track addresses in
type Subnet struct {
	ID        int    `gorm:"column:ID;primaryKey"`
	Name      string `gorm:"column:NAME"`
	CIDR      string `gorm:"column:CIDR"`
	Iface     string `gorm:"column:IFACE"`
	Method    string `gorm:"column:METHOD"`     // "arp" or "none"
	Reserved  string `gorm:"column:RESERVED"`   // comma separated IPs and ranges (a-b)
	SkipPorts bool   `gorm:"column:SKIP_PORTS"` // leave hosts here out of scheduled port scans
}

// SubnetStat - address usage of one subnet
type SubnetStat struct {
	Subnet      Subnet
	Total       int // usable addresses
	Online      int
	Offline     int // seen before, not online now
	Reserved    int // reserved and not in use
	Free        int
	NextFree    string
	Utilization int // percent of usable addresses not free
}

// Address - one address in a subnet
type Address struct {
	IP       string
	State    string // network, broadcast, online, offline, reserved, free
	Reserved bool
	HostID   int
	Name     string
	Mac      string
}

// IPAM - address map of one subnet
type IPAM struct {
	Stat      SubnetStat
	Addresses []Address // empty when the subnet is larger than the grid limit
}

// Port - one open TCP port on a host
type Port struct {
	ID      int    `gorm:"column:ID;primaryKey"`
	Mac     string `gorm:"column:MAC"`
	IP      string `gorm:"column:IP"`
	Port    int    `gorm:"column:PORT"`
	Service string `gorm:"column:SERVICE"`
	Web     string `gorm:"column:WEB"`   // "http", "https" or ""
	Title   string `gorm:"column:TITLE"` // title of the web page on this port, if any
	// Set when a connector reported this port
	Source    int    `gorm:"column:SOURCE"`    // connector ID, 0 for port scans
	Container string `gorm:"column:CONTAINER"` // container publishing the port
	Category  string `gorm:"column:CATEGORY"`  // category the connector gave the service
	First     string `gorm:"column:FIRST"`
	Last      string `gorm:"column:LAST"`
}

// PortScan - when a host was last port scanned
type PortScan struct {
	Mac  string `gorm:"column:MAC;primaryKey"`
	Date string `gorm:"column:DATE"`
	List string `gorm:"column:LIST"`
}

// PortJob - progress of a port scan
type PortJob struct {
	HostID   int
	List     string
	Running  bool
	Done     int
	Total    int
	Open     int
	Started  string
	Finished string
}

// HostPorts - open ports of a host and its scan state
type HostPorts struct {
	Ports    []Port
	LastScan PortScan
	Job      PortJob
}

// MapPos - saved position of a host on the map
type MapPos struct {
	Mac string  `gorm:"column:MAC;primaryKey"`
	X   float64 `gorm:"column:X"`
	Y   float64 `gorm:"column:Y"`
}

// MapHost - a host as drawn on the map
type MapHost struct {
	Host
	SubnetID   int  // 0 when the host is in no configured subnet
	HasPos     bool // false: the map places the host automatically
	X          float64
	Y          float64
	Web        []Port // open ports that serve a web page
	Icon       string // icon set for the host, "" if none
	Category   string // chosen category, or the suggested one, see package category
	Suggested  bool   // true when Category is a suggestion, not chosen
	Containers []Container
	Guests     []Guest    // VMs and LXCs when this host is a hypervisor node
	GuestOf    *Guest     // set when this host is a VM or LXC on a known hypervisor
	Bookmarks  []Bookmark // bookmarks linked to this host or its services
}

// Map - everything the map page draws
type Map struct {
	Subnets []Subnet
	Hosts   []MapHost
}

// View - a custom dashboard tab. Each group in it is a tag
type View struct {
	ID     int    `gorm:"column:ID;primaryKey"`
	Name   string `gorm:"column:NAME"`
	Layout string `gorm:"column:LAYOUT"` // "tiles" or "map"
	Tags   string `gorm:"column:TAGS"`   // groups in order, comma separated; empty shows every tag
	Sort   int    `gorm:"column:SORT"`
}

// Item - something carrying a tag: a host, one service on a host, or a bookmark
type Item struct {
	ID       int    `gorm:"column:ID;primaryKey"`
	Tag      string `gorm:"column:TAG"`
	Kind     string `gorm:"column:KIND"`     // "host", "service" or "bookmark" ("link" before bookmarks existed)
	Mac      string `gorm:"column:MAC"`      // host and service
	Port     int    `gorm:"column:PORT"`     // service
	Bookmark int    `gorm:"column:BOOKMARK"` // bookmark ID
	Name     string `gorm:"column:NAME"`     // optional for host and service
	URL      string `gorm:"column:URL"`      // overrides where a host or service tile goes
	Icon     string `gorm:"column:ICON"`     // unused since bookmarks; hosts and services use the icons table
	Sort     int    `gorm:"column:SORT"`
}

// Bookmark - an address added by hand, like https://immich.example.lan behind a
// reverse proxy. Mac (and Port) optionally tie it to a discovered host or service.
type Bookmark struct {
	ID   int    `gorm:"column:ID;primaryKey"`
	Name string `gorm:"column:NAME"`
	URL  string `gorm:"column:URL"`
	Icon string `gorm:"column:ICON"` // as in Icon; "" uses the linked host's or service's
	Note string `gorm:"column:NOTE"` // shown under the name instead of the address
	Mac  string `gorm:"column:MAC"`  // linked host, "" for none
	Port int    `gorm:"column:PORT"` // linked service on that host, 0 for the host itself
	// Source is the connector that keeps this bookmark in sync, 0 for one added
	// by hand. Its address and link follow the connector; deleting it hides it
	// so the next sync doesn't bring it back.
	Source int  `gorm:"column:SOURCE"`
	Hidden bool `gorm:"column:HIDDEN" json:"-"`
}

// BookmarkInfo - a bookmark with its tags and what it is linked to
type BookmarkInfo struct {
	Bookmark
	Tags     []string
	HostID   int    // 0 when not linked, or the host is gone
	HostName string // linked host's name or IP
}

// Icon - icon for a host (Port 0) or one of its services.
// Value: a dashboard-icons name like "immich", an image URL, or "none"
type Icon struct {
	Mac  string `gorm:"column:MAC;primaryKey"`
	Port int    `gorm:"column:PORT;primaryKey;autoIncrement:false"`
	Icon string `gorm:"column:ICON"`
}

// HostCategory - a category chosen for a host, used instead of the suggested one
type HostCategory struct {
	Mac      string `gorm:"column:MAC;primaryKey"`
	Category string `gorm:"column:CATEGORY"`
}

// ViewItem - an item with what the dashboard shows for it
type ViewItem struct {
	Item
	Title    string
	Subtitle string
	Link     string // where clicking the tile goes ("" opens host details)
	Status   string // "up", "down" or "unknown"
	Host     *MapHost
}

// ViewGroup - one tag's items
type ViewGroup struct {
	Tag   string
	Items []ViewItem
}

// ViewData - a view with its groups and items
type ViewData struct {
	View   View
	Groups []ViewGroup
}

// TagCount - a tag and how many items carry it
type TagCount struct {
	Tag   string
	Count int
}

// HostDay - how many scans saw a host online on one day
type HostDay struct {
	Date   string `gorm:"column:DATE;primaryKey"` // 2006-01-02
	Mac    string `gorm:"column:MAC;primaryKey"`
	Scans  int    `gorm:"column:SCANS"`
	Online int    `gorm:"column:ONLINE"`
}

// SubnetHour - scans summed over one hour for a subnet (SubnetID 0 = all hosts)
type SubnetHour struct {
	Hour      string `gorm:"column:HOUR;primaryKey"` // 2006-01-02 15
	SubnetID  int    `gorm:"column:SUBNET_ID;primaryKey;autoIncrement:false"`
	Samples   int    `gorm:"column:SAMPLES"`
	OnlineSum int    `gorm:"column:ONLINE_SUM"`
	UsedSum   int    `gorm:"column:USED_SUM"` // addresses not free (online, offline or reserved); all hosts for SubnetID 0
	Total     int    `gorm:"column:TOTAL"`    // usable addresses at the last sample
}

// FirstSeen - when a host was first discovered
type FirstSeen struct {
	Mac    string `gorm:"column:MAC;primaryKey"`
	Date   string `gorm:"column:DATE"`   // 2006-01-02 15:04:05
	Seeded bool   `gorm:"column:SEEDED"` // host was known before stats existed, so Date is an upper bound
}

// DayUptime - one host on one day
type DayUptime struct {
	Date   string
	Scans  int
	Online int
}

// HostUptime - uptime of one host over the requested days
type HostUptime struct {
	Host      Host
	FirstSeen string
	Before    bool // FirstSeen is the earliest record, the host may be older
	Scans     int
	Online    int
	Uptime    float64 // percent of scans online, -1 when there are no scans
	Days      []DayUptime
}

// TrendPoint - averages over one time bucket
type TrendPoint struct {
	Time   string // "2006-01-02 15" for hours, "2006-01-02" for days
	Online float64
	Used   float64
	Total  int
}

// SubnetTrend - online and used addresses over time for one subnet
type SubnetTrend struct {
	Subnet Subnet
	Now    SubnetStat
	Points []TrendPoint
}

// DayCount - a count per day
type DayCount struct {
	Date  string
	Count int
}

// Stats - everything the stats page draws
type Stats struct {
	Days     int
	From     string // first day included
	Bucket   string // "hour" or "day"
	All      []TrendPoint
	Subnets  []SubnetTrend
	Hosts    []HostUptime
	NewHosts []DayCount
}

// Connector - another system Gimlé reads hosts, names, services or containers from
type Connector struct {
	ID        int    `gorm:"column:ID;primaryKey"`
	Kind      string `gorm:"column:KIND"` // "docker", "dockhand", "scanopy", "unifi", "technitium", "proxmox" or "caddy"
	Name      string `gorm:"column:NAME"`
	URL       string `gorm:"column:URL"`
	Token     string `gorm:"column:TOKEN" json:"-"` // API key or token, never sent to the browser
	User      string `gorm:"column:USER"`           // for sources that log in with a user name
	Site      string `gorm:"column:SITE"`           // UniFi site, Dockhand environment; "" for all
	HostIP    string `gorm:"column:HOST_IP"`        // docker: IP of the host running Docker, if not the URL's
	Insecure  bool   `gorm:"column:INSECURE"`       // accept self-signed certificates
	Enabled   bool   `gorm:"column:ENABLED"`
	Interval  int    `gorm:"column:INTERVAL"` // minutes between syncs
	LastSync  string `gorm:"column:LAST_SYNC"`
	LastError string `gorm:"column:LAST_ERROR"`
	LastCount int    `gorm:"column:LAST_COUNT"` // hosts the last sync matched
	HasToken  bool   `gorm:"-"`
}

// ConnectorSave - a connector as the config page sends it. An empty Token keeps the saved one.
type ConnectorSave struct {
	Connector
	Token      string
	ClearToken bool
}

// Container - a Docker container a connector reported
type Container struct {
	ID          int             `gorm:"column:ID;primaryKey"`
	ConnectorID int             `gorm:"column:CONNECTOR_ID"`
	Mac         string          `gorm:"column:MAC"` // host it runs on, "" if that host is not known
	IP          string          `gorm:"column:IP"`  // IP of the Docker host
	Host        string          `gorm:"column:HOST"`
	CID         string          `gorm:"column:CID"` // container ID
	Name        string          `gorm:"column:NAME"`
	Image       string          `gorm:"column:IMAGE"`
	State       string          `gorm:"column:STATE"` // "running", "exited"...
	Status      string          `gorm:"column:STATUS"`
	Project     string          `gorm:"column:PROJECT"` // compose project
	PortsJSON   string          `gorm:"column:PORTS" json:"-"`
	Ports       []ContainerPort `gorm:"-"`
	Category    string          `gorm:"-"`
	Icon        string          `gorm:"-"` // icon suggestion from the image name
}

// ContainerPort - a port a container publishes
type ContainerPort struct {
	Private int
	Public  int // 0 when not published on the host
	Proto   string
	IP      string `json:",omitempty"`
}

// Guest - a virtual machine or container a hypervisor connector reported
type Guest struct {
	ID          int    `gorm:"column:ID;primaryKey"`
	ConnectorID int    `gorm:"column:CONNECTOR_ID"`
	Node        string `gorm:"column:NODE"`     // hypervisor node name
	NodeMac     string `gorm:"column:NODE_MAC"` // the node's host, "" if not known
	Mac         string `gorm:"column:MAC"`      // first NIC, lower case
	IP          string `gorm:"column:IP"`       // if the hypervisor knows it
	HostMac     string `gorm:"column:HOST_MAC"` // the guest's own host, "" if not discovered
	VMID        int    `gorm:"column:VMID"`
	Type        string `gorm:"column:TYPE"` // "qemu" (VM) or "lxc"
	Name        string `gorm:"column:NAME"`
	Status      string `gorm:"column:STATUS"` // "running", "stopped"...
	Tags        string `gorm:"column:TAGS"`   // as set in the hypervisor, ";" separated
	CPUs        int    `gorm:"column:CPUS"`
	MaxMem      int64  `gorm:"column:MAX_MEM"` // bytes
	Uptime      int64  `gorm:"column:UPTIME"`  // seconds
}
