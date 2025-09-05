package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// MockServer 模拟小米路由器的API服务
type MockServer struct {
	server     *http.Server
	authToken  string
	port       int
	devices    []MockDevice
	wifiInfo   MockWiFiInfo
	wanInfo    MockWanInfo
	systemInfo MockSystemInfo
}

// MockDevice 模拟设备信息
type MockDevice struct {
	Mac         string            `json:"mac"`
	IP          []MockIP          `json:"ip"`
	Name        string            `json:"name"`
	IsAP        int               `json:"is_ap"`
	Statistics  MockDeviceStats   `json:"statistics"`
	Upload      interface{}       `json:"upload"`
	Download    interface{}       `json:"download"`
}

type MockIP struct {
	IP string `json:"ip"`
}

type MockDeviceStats struct {
	Online    interface{} `json:"online"`
	UpSpeed   interface{} `json:"upspeed"`
	DownSpeed interface{} `json:"downspeed"`
}

// MockWiFiInfo 模拟WiFi信息
type MockWiFiInfo struct {
	Info []MockWiFiDetail `json:"info"`
}

type MockWiFiDetail struct {
	Ssid        string            `json:"ssid"`
	Status      string            `json:"status"`
	ChannelInfo MockChannelInfo   `json:"channelInfo"`
}

type MockChannelInfo struct {
	BandList []string `json:"bandList"`
	Channel  int      `json:"channel"`
}

// MockWanInfo 模拟WAN信息
type MockWanInfo struct {
	Info MockWanDetail `json:"info"`
}

type MockWanDetail struct {
	Ipv4     []MockIPv4     `json:"ipv4"`
	Ipv6Info MockIPv6Info   `json:"ipv6Info"`
}

type MockIPv4 struct {
	IP   string `json:"ip"`
	Mask string `json:"mask"`
}

type MockIPv6Info struct {
	IP6Addr []string `json:"ip6Addr"`
}

// MockSystemInfo 模拟系统信息
type MockSystemInfo struct {
	CPU       MockCPU       `json:"cpu"`
	Mem       MockMemory    `json:"mem"`
	Dev       []MockDev     `json:"dev"`
	Wan       MockWan       `json:"wan"`
	Count     MockCount     `json:"count"`
	UpTime    string        `json:"upTime"`
	Hardware  MockHardware  `json:"hardware"`
	Code      int           `json:"code"`
}

type MockCPU struct {
	Core int     `json:"core"`
	Hz   string  `json:"hz"`
	Load float64 `json:"load"`
}

type MockMemory struct {
	Total string  `json:"total"`
	Usage float64 `json:"usage"`
}

type MockDev struct {
	Mac       string      `json:"mac"`
	Upload    interface{} `json:"upload"`
	Download  interface{} `json:"download"`
}

type MockWan struct {
	UpSpeed   string `json:"upSpeed"`
	DownSpeed string `json:"downSpeed"`
	Upload    string `json:"upload"`
	Download  string `json:"download"`
}

type MockCount struct {
	All             int `json:"all"`
	Online          int `json:"online"`
	AllWithoutMash  int `json:"allWithoutMash"`
	OnlineWithoutMash int `json:"onlineWithoutMash"`
}

type MockHardware struct {
	Platform     string `json:"platform"`
	Version      string `json:"version"`
	Sn           string `json:"sn"`
	Mac          string `json:"mac"`
}

// InitInfo 初始化信息
type InitInfo struct {
	Hardware      string `json:"hardware"`
	RomVersion    string `json:"rom_version"`
	SerialNumber  string `json:"serial_number"`
	RouterName    string `json:"router_name"`
	NewEncryptMode int   `json:"new_encrypt_mode"`
}

// NewMockServer 创建新的mock服务器
func NewMockServer(port int) *MockServer {
	mockServer := &MockServer{
		port:      port,
		authToken: generateMockToken(),
	}

	// 初始化模拟数据
	mockServer.initializeMockData()

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/cgi-bin/luci/web", mockServer.handleWebPage)
	mux.HandleFunc("/cgi-bin/luci/api/xqsystem/init_info", mockServer.handleInitInfo)
	mux.HandleFunc("/cgi-bin/luci/api/xqsystem/login", mockServer.handleLogin)
	mux.HandleFunc("/cgi-bin/luci/;stok=", mockServer.handleAuthRequest)
	mux.HandleFunc("/cgi-bin/luci/api/misystem/status", mockServer.handleSystemStatus)
	mux.HandleFunc("/cgi-bin/luci/api/misystem/devicelist", mockServer.handleDeviceList)
	mux.HandleFunc("/cgi-bin/luci/api/xqnetwork/wan_info", mockServer.handleWanInfo)
	mux.HandleFunc("/cgi-bin/luci/api/xqnetwork/wifi_detail_all", mockServer.handleWifiDetails)

	mockServer.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return mockServer
}

// initializeMockData 初始化模拟数据
func (ms *MockServer) initializeMockData() {
	// 模拟设备数据
	ms.devices = []MockDevice{
		{
			Mac:  "aa:bb:cc:dd:ee:ff",
			IP:   []MockIP{{IP: "192.168.31.100"}},
			Name: "iPhone-13",
			IsAP: 0,
			Statistics: MockDeviceStats{
				Online:    "3600",
				UpSpeed:   "1024",
				DownSpeed: "2048",
			},
			Upload:   "1048576",
			Download: "2097152",
		},
		{
			Mac:  "ff:ee:dd:cc:bb:aa",
			IP:   []MockIP{{IP: "192.168.31.101"}},
			Name: "MacBook-Pro",
			IsAP: 0,
			Statistics: MockDeviceStats{
				Online:    "7200",
				UpSpeed:   "512",
				DownSpeed: "1024",
			},
			Upload:   "524288",
			Download: "1048576",
		},
		{
			Mac:  "11:22:33:44:55:66",
			IP:   []MockIP{{IP: "192.168.31.102"}},
			Name: "Android-Phone",
			IsAP: 0,
			Statistics: MockDeviceStats{
				Online:    "1800",
				UpSpeed:   "256",
				DownSpeed: "512",
			},
			Upload:   "262144",
			Download: "524288",
		},
	}

	// 模拟WiFi信息
	ms.wifiInfo = MockWiFiInfo{
		Info: []MockWiFiDetail{
			{
				Ssid:   "MiWiFi_5G",
				Status: "on",
				ChannelInfo: MockChannelInfo{
					BandList: []string{"5"},
					Channel:  149,
				},
			},
			{
				Ssid:   "MiWiFi_2.4G",
				Status: "on",
				ChannelInfo: MockChannelInfo{
					BandList: []string{"2.4"},
					Channel:  6,
				},
			},
		},
	}

	// 模拟WAN信息
	ms.wanInfo = MockWanInfo{
		Info: MockWanDetail{
			Ipv4: []MockIPv4{
				{
					IP:   "100.100.100.100",
					Mask: "255.255.255.0",
				},
			},
			Ipv6Info: MockIPv6Info{
				IP6Addr: []string{"2001:db8::1"},
			},
		},
	}

	// 模拟系统信息
	ms.systemInfo = MockSystemInfo{
		CPU: MockCPU{
			Core: 4,
			Hz:   "800000000",
			Load: 25.5,
		},
		Mem: MockMemory{
			Total: "256MB",
			Usage: 0.65,
		},
		Dev: []MockDev{
			{
				Mac:      "aa:bb:cc:dd:ee:ff",
				Upload:   "1048576",
				Download: "2097152",
			},
			{
				Mac:      "ff:ee:dd:cc:bb:aa",
				Upload:   "524288",
				Download: "1048576",
			},
			{
				Mac:      "11:22:33:44:55:66",
				Upload:   "262144",
				Download: "524288",
			},
		},
		Wan: MockWan{
			UpSpeed:   "100.5",
			DownSpeed: "200.8",
			Upload:    "1073741824",
			Download:  "2147483648",
		},
		Count: MockCount{
			All:             3,
			Online:          3,
			AllWithoutMash:  3,
			OnlineWithoutMash: 3,
		},
		UpTime: "86400",
		Hardware: MockHardware{
			Platform: "miwifi_r3p",
			Version:  "2.28.123",
			Sn:       "1234567890",
			Mac:      "aa:bb:cc:dd:ee:ff",
		},
		Code: 0,
	}
}

// generateMockToken 生成模拟token
func generateMockToken() string {
	return fmt.Sprintf("mock_token_%d", time.Now().Unix())
}

// Start 启动mock服务器
func (ms *MockServer) Start() error {
	log.Printf("Mock MiWiFi server starting on port %d", ms.port)
	log.Printf("Auth token: %s", ms.authToken)
	return ms.server.ListenAndServe()
}

// Stop 停止mock服务器
func (ms *MockServer) Stop() error {
	return ms.server.Close()
}

// handleWebPage 处理web页面请求
func (ms *MockServer) handleWebPage(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>MiWiFi Login</title>
</head>
<body>
    <script>
        var key = 'mock_key_123456';
        var deviceId = 'mock_device_id_789012';
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleInitInfo 处理初始化信息请求
func (ms *MockServer) handleInitInfo(w http.ResponseWriter, r *http.Request) {
	initInfo := InitInfo{
		Hardware:      "miwifi_r3p",
		RomVersion:    "2.28.123",
		SerialNumber:  "1234567890",
		RouterName:    "MiWiFi-Test",
		NewEncryptMode: 1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(initInfo)
}

// handleLogin 处理登录请求
func (ms *MockServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	
	response := map[string]interface{}{
		"token": ms.authToken,
		"url":   fmt.Sprintf("/cgi-bin/luci/;stok=%s/web", ms.authToken),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleAuthRequest 处理需要认证的请求
func (ms *MockServer) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	// 检查URL路径以确定具体的API端点
	path := r.URL.Path
	
	if strings.Contains(path, "api/misystem/status") {
		ms.handleSystemStatus(w, r)
	} else if strings.Contains(path, "api/misystem/devicelist") {
		ms.handleDeviceList(w, r)
	} else if strings.Contains(path, "api/xqnetwork/wan_info") {
		ms.handleWanInfo(w, r)
	} else if strings.Contains(path, "api/xqnetwork/wifi_detail_all") {
		ms.handleWifiDetails(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// handleSystemStatus 处理系统状态请求
func (ms *MockServer) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	// 随机变化一些数据以模拟真实环境
	ms.systemInfo.CPU.Load = 20 + rand.Float64()*20
	ms.systemInfo.Mem.Usage = 0.5 + rand.Float64()*0.3
	ms.systemInfo.Wan.UpSpeed = fmt.Sprintf("%.1f", 80+rand.Float64()*40)
	ms.systemInfo.Wan.DownSpeed = fmt.Sprintf("%.1f", 150+rand.Float64()*100)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ms.systemInfo)
}

// handleDeviceList 处理设备列表请求
func (ms *MockServer) handleDeviceList(w http.ResponseWriter, r *http.Request) {
	// 随机更新设备统计数据
	for i := range ms.devices {
		device := &ms.devices[i]
		device.Statistics.UpSpeed = fmt.Sprintf("%d", 500+rand.Intn(1000))
		device.Statistics.DownSpeed = fmt.Sprintf("%d", 1000+rand.Intn(2000))
		
		// 更新上传下载量
		if upload, ok := device.Upload.(string); ok {
			if uploadVal, err := strconv.ParseFloat(upload, 64); err == nil {
				device.Upload = fmt.Sprintf("%.0f", uploadVal+rand.Float64()*1000)
			}
		}
		if download, ok := device.Download.(string); ok {
			if downloadVal, err := strconv.ParseFloat(download, 64); err == nil {
				device.Download = fmt.Sprintf("%.0f", downloadVal+rand.Float64()*2000)
			}
		}
	}

	response := map[string]interface{}{
		"code": 0,
		"list": ms.devices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWanInfo 处理WAN信息请求
func (ms *MockServer) handleWanInfo(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"code": 0,
		"info": ms.wanInfo.Info,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWifiDetails 处理WiFi详情请求
func (ms *MockServer) handleWifiDetails(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"code": 0,
		"info": ms.wifiInfo.Info,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	port := 8080
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	mockServer := NewMockServer(port)
	
	log.Printf("Mock MiWiFi Server for Testing")
	log.Printf("=================================")
	log.Printf("Server running on http://localhost:%d", port)
	log.Printf("Use this server to test miwifi-exporter")
	log.Printf("Configuration for exporter:")
	log.Printf("  IP: localhost")
	log.Printf("  Port: %d", port)
	log.Printf("  Password: any_password")
	log.Printf("=================================")

	if err := mockServer.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Mock server failed: %v", err)
	}
}