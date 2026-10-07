import { createSignal } from "solid-js";
import { createStore } from "solid-js/store";

export interface Host {
	ID:    number;
	Name:  string;
	DNS:   string;
	Iface: string;
	IP:    string;
	Mac:   string;
	Hw:    string;
	Date:  string;
	Known: number;
	Now:   number;
};

export interface Conf {
	Host:	   string;
	Port:	   string;
	Theme:	   string;
	Color:     string;
	DirPath:   string;
	Timeout:   number;
	NodePath:  string;
	LogLevel:  string;
	Ifaces:	   string;
	ArpArgs:   string;
	ArpStrs:   string[];
	TrimHist:  number;
	StatsDays: number;
	ShoutURL:  string;
	UseDB:     string;
	PGConnect: string;
	// InfluxDB
	InfluxEnable:  boolean;
	InfluxAddr:    string;
	InfluxToken:   string;
	InfluxOrg:     string;
	InfluxBucket:  string;
	InfluxSkipTLS: boolean;
	// Prometheus
	PrometheusEnable: boolean;
	// Port scanning
	PortScan:     boolean;
	PortList:     string;
	PortInterval: number;
	PortWorkers:  number;
	PortTimeout:  number;
	// Host names
	DNSServer:   string;
	NameMDNS:    boolean;
	NameNetBIOS: boolean;
	StartPage:   string; // "" for the network map, or a path like /view/3
	// Docs from a GitHub repository; the token itself is never sent
	DocsRepo:     string;
	DocsBranch:   string;
	DocsDir:      string;
	DocsEdit:     boolean;
	DocsHasToken: boolean;
	DocsTokenEnv: boolean;
};

export interface DocNav {
	Title:     string;
	Path?:     string;
	Children?: DocNav[];
};

export interface DocsIndex {
	Repo:    string;
	Branch:  string;
	Dir:     string;
	CanEdit: boolean;
	Nav:     DocNav[] | null;
	Error?:  string;
};

export interface DocPage {
	Path:     string;
	Title:    string;
	HTML:     string;
	Markdown: string;
	Sha:      string;
	GitHub:   string;
};

export const emptyHost:Host = {
	ID:    0,
	Name:  "",
	DNS:   "",
	Iface: "",
	IP:    "",
	Mac:   "",
	Hw:    "",
	Date:  "",
	Known: 0,
	Now:   0,
};

export const emptyConf:Conf = {
	Host:	 "",
	Port:	 "",
	Theme:	 "",
	Color:   "",
	DirPath: "",
	Timeout: 120,
	NodePath: "",
	LogLevel: "",
	Ifaces:	 "",
	ArpArgs: "",
	ArpStrs: [],
	TrimHist: 48,
	StatsDays: 90,
	ShoutURL: "",
	UseDB: "",
	PGConnect: "",
	InfluxEnable:  false,
	InfluxAddr:    "",
	InfluxToken:   "",
	InfluxOrg:     "",
	InfluxBucket:  "",
	InfluxSkipTLS: false,
	PrometheusEnable: false,
	PortScan:     false,
	PortList:     "top",
	PortInterval: 360,
	PortWorkers:  64,
	PortTimeout:  700,
	DNSServer:    "",
	NameMDNS:     true,
	NameNetBIOS:  true,
	StartPage:    "",
	DocsRepo:     "",
	DocsBranch:   "",
	DocsDir:      "",
	DocsEdit:     false,
	DocsHasToken: false,
	DocsTokenEnv: false,
};

export const [allHosts, setAllHosts] = createStore<Host[]>([]);
export const [bkpHosts, setBkpHosts] = createSignal<Host[]>([]);

export const [ifaces, setIfaces] = createSignal<string[]>([]);

export const [appConfig, setAppConfig] = createSignal<Conf>(emptyConf);

export const [editNames, setEditNames] = createSignal(false);

export const [show, setShow] = createSignal<number>(200);

export const [histUpdOnFilter, setHistUpdOnFilter] = createSignal(false);

export const [selectedIDs, setSelectedIDs] = createSignal<number[]>([]);
export interface Subnet {
	ID:       number;
	Name:     string;
	CIDR:     string;
	Iface:    string;
	Method:   string;
	Reserved:  string;
	SkipPorts: boolean;
};

export interface SubnetStat {
	Subnet:      Subnet;
	Total:       number;
	Online:      number;
	Offline:     number;
	Reserved:    number;
	Free:        number;
	NextFree:    string;
	Utilization: number;
};

export interface Address {
	IP:       string;
	State:    string;
	Reserved: boolean;
	HostID:   number;
	Name:     string;
	Mac:      string;
};

export interface IPAM {
	Stat:      SubnetStat;
	Addresses: Address[] | null;
};

export const emptySubnet:Subnet = {
	ID:       0,
	Name:     "",
	CIDR:     "",
	Iface:    "",
	Method:   "arp",
	Reserved: "",
	SkipPorts: false,
};

export interface Port {
	ID:      number;
	Mac:     string;
	IP:      string;
	Port:    number;
	Service: string;
	Web:     string;
	Title:   string;
	Source:    number;
	Container: string;
	Category:  string;
	First:   string;
	Last:    string;
};

export interface PortJob {
	HostID:   number;
	List:     string;
	Running:  boolean;
	Done:     number;
	Total:    number;
	Open:     number;
	Started:  string;
	Finished: string;
};

export interface HostPorts {
	Ports:    Port[];
	LastScan: { Mac: string; Date: string; List: string };
	Job:      PortJob;
};

export interface MapHost extends Host {
	SubnetID: number;
	HasPos:   boolean;
	X:        number;
	Y:        number;
	Web:      Port[];
	Icon:     string;
	Category:   string;
	Suggested:  boolean;
	Containers: Container[];
	Guests:     Guest[];
	GuestOf:    Guest | null;
	Bookmarks:  Bookmark[];
};

export interface Guest {
	ID:      number;
	Node:    string;
	NodeMac: string;
	Mac:     string;
	IP:      string;
	HostMac: string;
	VMID:    number;
	Type:    string;
	Name:    string;
	Status:  string;
	Tags:    string;
	CPUs:    number;
	MaxMem:  number;
	Uptime:  number;
};

export interface ContainerPort {
	Private: number;
	Public:  number;
	Proto:   string;
};

export interface Container {
	ID:          number;
	ConnectorID: number;
	Mac:         string;
	IP:          string;
	Host:        string;
	CID:         string;
	Name:        string;
	Image:       string;
	State:       string;
	Status:      string;
	Project:     string;
	Ports:       ContainerPort[];
	Category:    string;
};

export interface Connector {
	ID:        number;
	Kind:      string;
	Name:      string;
	URL:       string;
	User:      string;
	Site:      string;
	HostIP:    string;
	Insecure:  boolean;
	Enabled:   boolean;
	Interval:  number;
	LastSync:  string;
	LastError: string;
	LastCount: number;
	HasToken:  boolean;
};

export interface ConnectorKind {
	Label:   string;
	Token:   string;
	Example: string;
};

export interface MapData {
	Subnets: Subnet[];
	Hosts:   MapHost[];
};

export interface View {
	ID:     number;
	Name:   string;
	Layout: string; // tiles or map
	Tags:   string; // comma separated, empty means every tag
	Sort:   number;
};

export interface Item {
	ID:   number;
	Tag:  string;
	Kind: string; // host, service or bookmark
	Mac:  string;
	Port: number;
	Bookmark?: number;
	Name: string;
	URL:  string;
	Icon: string;
	Sort: number;
};

export interface Bookmark {
	ID:   number;
	Name: string;
	URL:  string;
	Icon: string;
	Note: string;
	Mac:  string; // linked host, "" for none
	Port: number; // linked service, 0 for the host itself
};

export interface BookmarkInfo extends Bookmark {
	Tags:     string[];
	HostID:   number;
	HostName: string;
};

export interface ViewItem extends Item {
	Title:    string;
	Subtitle: string;
	Link:     string;
	Status:   string; // up, down or unknown
	Host:     MapHost | null;
};

export interface ViewGroup {
	Tag:   string;
	Items: ViewItem[];
};

export interface ViewData {
	View:   View;
	Groups: ViewGroup[];
};

export interface TagCount {
	Tag:   string;
	Count: number;
};

export interface DayUptime {
	Date:   string;
	Scans:  number;
	Online: number;
};

export interface HostUptime {
	Host:      Host;
	FirstSeen: string;
	Before:    boolean; // known before stats began, so FirstSeen is a latest date
	Scans:     number;
	Online:    number;
	Uptime:    number; // percent, -1 without scans
	Days:      DayUptime[];
};

export interface TrendPoint {
	Time:   string;
	Online: number;
	Used:   number;
	Total:  number;
};

export interface SubnetTrend {
	Subnet: Subnet;
	Now:    SubnetStat;
	Points: TrendPoint[];
};

export interface Stats {
	Days:     number;
	From:     string;
	Bucket:   string; // hour or day
	All:      TrendPoint[];
	Subnets:  SubnetTrend[];
	Hosts:    HostUptime[];
	NewHosts: { Date: string, Count: number }[];
};
