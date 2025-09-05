package models

// SystemStatus represents the system status from miwifi
type SystemStatus struct {
	Dev []DeviceInfo `json:"dev"`
	Code int        `json:"code"`
	Mem  MemoryInfo `json:"mem"`
	Temperature int        `json:"temperature"`
	Count       DeviceCount `json:"count"`
	Hardware    HardwareInfo `json:"hardware"`
	UpTime      string      `json:"upTime"`
	CPU         CPUInfo     `json:"cpu"`
	Wan         WanStatus   `json:"wan"`
}

type DeviceInfo struct {
	Mac              string      `json:"mac"`
	MaxDownloadSpeed string      `json:"maxdownloadspeed"`
	Upload           interface{} `json:"upload"`
	UpSpeed          interface{} `json:"upspeed"`
	DownSpeed        interface{} `json:"downspeed"`
	Online           string      `json:"online"`
	DevName          string      `json:"devname"`
	MaxUploadSpeed   string      `json:"maxuploadspeed"`
	Download         interface{} `json:"download"`
}

type MemoryInfo struct {
	Usage float64 `json:"usage"`
	Total string  `json:"total"`
	Hz    string  `json:"hz"`
	Type  string  `json:"type"`
}

type DeviceCount struct {
	All               int `json:"all"`
	Online            int `json:"online"`
	AllWithoutMash    int `json:"all_without_mash"`
	OnlineWithoutMash int `json:"online_without_mash"`
}

type HardwareInfo struct {
	Mac      string `json:"mac"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Channel  string `json:"channel"`
	Sn       string `json:"sn"`
}

type CPUInfo struct {
	Core int     `json:"core"`
	Hz   string  `json:"hz"`
	Load float64 `json:"load"`
}

type WanStatus struct {
	DownSpeed        string `json:"downspeed"`
	MaxDownloadSpeed string `json:"maxdownloadspeed"`
	History          string `json:"history"`
	DevName          string `json:"devname"`
	Upload           string `json:"upload"`
	UpSpeed          string `json:"upspeed"`
	MaxUploadSpeed   string `json:"maxuploadspeed"`
	Download         string `json:"download"`
}

// DeviceList represents the device list from miwifi
type DeviceList struct {
	Mac  string        `json:"mac"`
	List []DeviceEntry `json:"list"`
	Code int           `json:"code"`
}

type DeviceEntry struct {
	Mac       string           `json:"mac"`
	OnNme     string           `json:"oname"`
	IsAP      int              `json:"isap"`
	Parent    string           `json:"parent"`
	Authority AuthorityInfo    `json:"authority"`
	Push      int              `json:"push"`
	Online    int              `json:"online"`
	Name      string           `json:"name"`
	Times     int              `json:"times"`
	IP        []IPInfo         `json:"ip"`
	Statistics DeviceStatistics `json:"statistics"`
	Icon      string           `json:"icon"`
	Type      int              `json:"type"`
}

type AuthorityInfo struct {
	Wan     int `json:"wan"`
	PriDisk int `json:"pridisk"`
	Admin   int `json:"admin"`
	Lan     int `json:"lan"`
}

type IPInfo struct {
	DownSpeed string `json:"downspeed"`
	Online    string `json:"online"`
	Active    int    `json:"active"`
	UpSpeed   string `json:"upspeed"`
	IP        string `json:"ip"`
}

type DeviceStatistics struct {
	DownSpeed string `json:"downspeed"`
	Online    string `json:"online"`
	UpSpeed   string `json:"upspeed"`
}

// WanInfo represents WAN information
type WanInfo struct {
	Info WanInfoDetails `json:"info"`
	Code int            `json:"code"`
}

type WanInfoDetails struct {
	Mac     string    `json:"mac"`
	Mtu     string    `json:"mtu"`
	Details WanConfig `json:"details"`
	GateWay string    `json:"gateWay"`
	DnsAddr1 string   `json:"dnsAddrs1"`
	Status   int      `json:"status"`
	Uptime   int      `json:"uptime"`
	DNSAddr  string   `json:"dnsAddrs"`
	Ipv6Info IPv6Info `json:"ipv6_info"`
	Ipv6Show int      `json:"ipv6_show"`
	Link     int      `json:"link"`
	Ipv4     []IPv4   `json:"ipv4"`
}

type WanConfig struct {
	Username string `json:"username"`
	IfName   string `json:"ifname"`
	WanType  string `json:"wanType"`
	Service  string `json:"service"`
	Password string `json:"password"`
	PeerDns  string `json:"peerdns"`
}

type IPv6Info struct {
	WanType      string        `json:"wanType"`
	IfName       string        `json:"ifname"`
	DNS          []interface{} `json:"dns"`
	IP6Addr      []string      `json:"ip6addr"`
	PeerDns      string        `json:"peerdns"`
	LanIP6Prefix []interface{} `json:"lan_ip6prefix"`
	LanIP6Addr   []interface{} `json:"lan_ip6addr"`
}

type IPv4 struct {
	Mask string `json:"mask"`
	IP   string `json:"ip"`
}

// WifiDetailAll represents WiFi information
type WifiDetailAll struct {
	Bsd  int           `json:"bsd"`
	Info []WifiDetails `json:"info"`
	Code int           `json:"code"`
}

type WifiDetails struct {
	IfName      string      `json:"ifname"`
	ChannelInfo ChannelInfo `json:"channelInfo"`
	Encryption  string      `json:"encryption"`
	Bandwidth   string      `json:"bandwidth"`
	KickThreshold string    `json:"kickthreshold"`
	Status      string      `json:"status"`
	Mode        string      `json:"mode"`
	Bsd         string      `json:"bsd"`
	Ssid        string      `json:"ssid"`
	WeakThreshold string    `json:"weakthreshold"`
	Device      string      `json:"device"`
	Ax          string      `json:"ax"`
	Hidden      string      `json:"hidden"`
	Password    string      `json:"password"`
	Channel     string      `json:"channel"`
	TxPWR       string      `json:"txpwr"`
	WeakEnable  string      `json:"weakenable"`
	TxBF        string      `json:"txbf"`
	Signal      int         `json:"signal"`
}

type ChannelInfo struct {
	Bandwidth string   `json:"bandwidth"`
	BandList  []string `json:"bandList"`
	Channel   int      `json:"channel"`
}

// Auth represents authentication information
type Auth struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	Code  int    `json:"code"`
}

// Router represents router configuration
type Router struct {
	IP       string
	Password string
	Headers  map[string]string
	Session  interface{}
	Data     map[string]string
	Token    string
	Path     string
	Stok     string
}

// InitInfo represents initialization information
type InitInfo struct {
	Hardware       string `json:"hardware"`
	RomVersion     string `json:"romversion"`
	SerialNumber   string `json:"id"`
	RouterName     string `json:"routername"`
	NewEncryptMode int    `json:"newEncryptMode"`
}