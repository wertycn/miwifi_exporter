package collector

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/helloworlde/miwifi-exporter/internal/client"
	"github.com/helloworlde/miwifi-exporter/internal/config"
	"github.com/helloworlde/miwifi-exporter/internal/logger"
	"github.com/helloworlde/miwifi-exporter/internal/metrics"
	"github.com/helloworlde/miwifi-exporter/internal/models"
	"github.com/helloworlde/miwifi-exporter/pkg/cache"
	"github.com/helloworlde/miwifi-exporter/pkg/concurrent"
	"github.com/helloworlde/miwifi-exporter/pkg/memory"
	"github.com/helloworlde/miwifi-exporter/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricsCollector struct {
	client         client.RouterClient
	config         *config.Config
	cache          *cache.RouterSmartCache
	dataFetcher    *concurrent.DataFetcher
	metrics        *prometheus.Registry
	descriptors    map[string]*prometheus.Desc
	collectorMetrics *metrics.CollectorMetrics
	memoryMonitor  *memory.MemoryMonitor
	mutex          sync.RWMutex
}

type Metrics struct {
	CPUCore         *prometheus.Desc
	CPUMHz          *prometheus.Desc
	CPULoad         *prometheus.Desc
	MemoryTotal     *prometheus.Desc
	MemoryUsage     *prometheus.Desc
	MemoryUsageMB   *prometheus.Desc
	DeviceCount     *prometheus.Desc
	DeviceOnline    *prometheus.Desc
	Uptime          *prometheus.Desc
	Platform        *prometheus.Desc
	Version         *prometheus.Desc
	SerialNumber    *prometheus.Desc
	MACAddress      *prometheus.Desc
	IPv4Address    *prometheus.Desc
	IPv4Mask        *prometheus.Desc
	IPv6Address     *prometheus.Desc
	WANUpSpeed      *prometheus.Desc
	WANDownSpeed    *prometheus.Desc
	WANUpload       *prometheus.Desc
	WANDownload     *prometheus.Desc
	DeviceUpload    *prometheus.Desc
	DeviceDownload  *prometheus.Desc
	DeviceUpSpeed   *prometheus.Desc
	DeviceDownSpeed *prometheus.Desc
	DeviceOnlineTime *prometheus.Desc
	WifiDetail      *prometheus.Desc
}

func NewMetricsCollector(cfg *config.Config) *MetricsCollector {
	mc := &MetricsCollector{
		config:      cfg,
		cache:       cache.NewRouterSmartCache(cfg.Cache.TTL, 1000, true),
		dataFetcher: concurrent.NewDataFetcher(
			time.Duration(cfg.Router.Timeout)*time.Second,
			3,
			5*time.Second,
		),
		collectorMetrics: metrics.NewCollectorMetrics(cfg.Server.Namespace),
		memoryMonitor:   memory.NewMemoryMonitor(cfg.Server.Namespace),
	}

	mc.initializeMetrics()
	mc.initializeDescriptors()
	
	// Configure memory monitor
	if mc.memoryMonitor != nil {
		mc.memoryMonitor.Configure(
			cfg.Memory.Enabled,
			cfg.Memory.OptimizeOnCollect,
			cfg.Memory.ForceGCOnClose,
			cfg.Memory.TrackAllocations,
			cfg.Memory.EnablePoolStats,
		)
	}

	return mc
}

func (mc *MetricsCollector) initializeMetrics() {
	mc.metrics = prometheus.NewRegistry()
	mc.metrics.MustRegister(mc)
	mc.metrics.MustRegister(mc.collectorMetrics)
	mc.metrics.MustRegister(mc.memoryMonitor)
}

func (mc *MetricsCollector) initializeDescriptors() {
	namespace := mc.config.Server.Namespace

	mc.descriptors = map[string]*prometheus.Desc{
		"cpu_cores": prometheus.NewDesc(
			fmt.Sprintf("%s_cpu_cores", namespace),
			"Number of CPU cores",
			[]string{"host"}, nil,
		),
		"cpu_mhz": prometheus.NewDesc(
			fmt.Sprintf("%s_cpu_mhz", namespace),
			"CPU frequency in MHz",
			[]string{"host"}, nil,
		),
		"cpu_load": prometheus.NewDesc(
			fmt.Sprintf("%s_cpu_load", namespace),
			"CPU load percentage",
			[]string{"host"}, nil,
		),
		"memory_total_mb": prometheus.NewDesc(
			fmt.Sprintf("%s_memory_total_mb", namespace),
			"Total memory in MB",
			[]string{"host"}, nil,
		),
		"memory_usage_mb": prometheus.NewDesc(
			fmt.Sprintf("%s_memory_usage_mb", namespace),
			"Memory usage in MB",
			[]string{"host"}, nil,
		),
		"memory_usage": prometheus.NewDesc(
			fmt.Sprintf("%s_memory_usage", namespace),
			"Memory usage percentage",
			[]string{"host"}, nil,
		),
		"count_all": prometheus.NewDesc(
			fmt.Sprintf("%s_count_all", namespace),
			"Total number of devices",
			[]string{"host"}, nil,
		),
		"count_online": prometheus.NewDesc(
			fmt.Sprintf("%s_count_online", namespace),
			"Number of online devices",
			[]string{"host"}, nil,
		),
		"count_all_without_mash": prometheus.NewDesc(
			fmt.Sprintf("%s_count_all_without_mash", namespace),
			"Total number of devices without mesh",
			[]string{"host"}, nil,
		),
		"count_online_without_mash": prometheus.NewDesc(
			fmt.Sprintf("%s_count_online_without_mash", namespace),
			"Number of online devices without mesh",
			[]string{"host"}, nil,
		),
		"uptime": prometheus.NewDesc(
			fmt.Sprintf("%s_uptime", namespace),
			"Router uptime in seconds",
			[]string{"host"}, nil,
		),
		"platform": prometheus.NewDesc(
			fmt.Sprintf("%s_platform", namespace),
			"Router platform information",
			[]string{"platform"}, nil,
		),
		"version": prometheus.NewDesc(
			fmt.Sprintf("%s_version", namespace),
			"Router firmware version",
			[]string{"version"}, nil,
		),
		"sn": prometheus.NewDesc(
			fmt.Sprintf("%s_sn", namespace),
			"Router serial number",
			[]string{"sn"}, nil,
		),
		"mac": prometheus.NewDesc(
			fmt.Sprintf("%s_mac", namespace),
			"Router MAC address",
			[]string{"mac"}, nil,
		),
		"ipv4": prometheus.NewDesc(
			fmt.Sprintf("%s_ipv4", namespace),
			"Router IPv4 address",
			[]string{"ipv4"}, nil,
		),
		"ipv4_mask": prometheus.NewDesc(
			fmt.Sprintf("%s_ipv4_mask", namespace),
			"Router IPv4 subnet mask",
			[]string{"ipv4"}, nil,
		),
		"ipv6": prometheus.NewDesc(
			fmt.Sprintf("%s_ipv6", namespace),
			"Router IPv6 address",
			[]string{"ipv6"}, nil,
		),
		"wan_upload_speed": prometheus.NewDesc(
			fmt.Sprintf("%s_wan_upload_speed", namespace),
			"WAN upload speed",
			[]string{"host"}, nil,
		),
		"wan_download_speed": prometheus.NewDesc(
			fmt.Sprintf("%s_wan_download_speed", namespace),
			"WAN download speed",
			[]string{"host"}, nil,
		),
		"wan_upload_traffic": prometheus.NewDesc(
			fmt.Sprintf("%s_wan_upload_traffic", namespace),
			"WAN upload traffic",
			[]string{"host"}, nil,
		),
		"wan_download_traffic": prometheus.NewDesc(
			fmt.Sprintf("%s_wan_download_traffic", namespace),
			"WAN download traffic",
			[]string{"host"}, nil,
		),
		"device_upload_traffic": prometheus.NewDesc(
			fmt.Sprintf("%s_device_upload_traffic", namespace),
			"Device upload traffic",
			[]string{"ip", "mac", "device_name", "is_ap"}, nil,
		),
		"device_upload_speed": prometheus.NewDesc(
			fmt.Sprintf("%s_device_upload_speed", namespace),
			"Device upload speed",
			[]string{"ip", "mac", "device_name", "is_ap"}, nil,
		),
		"device_download_traffic": prometheus.NewDesc(
			fmt.Sprintf("%s_device_download_traffic", namespace),
			"Device download traffic",
			[]string{"ip", "mac", "device_name", "is_ap"}, nil,
		),
		"device_download_speed": prometheus.NewDesc(
			fmt.Sprintf("%s_device_download_speed", namespace),
			"Device download speed",
			[]string{"ip", "mac", "device_name", "is_ap"}, nil,
		),
		"device_online_time": prometheus.NewDesc(
			fmt.Sprintf("%s_device_online_time", namespace),
			"Device online time",
			[]string{"ip", "mac", "device_name", "is_ap"}, nil,
		),
		"wifi_detail": prometheus.NewDesc(
			fmt.Sprintf("%s_wifi_detail", namespace),
			"WiFi network details",
			[]string{"ssid", "status", "band_list", "channel"}, nil,
		),
	}
}

func (mc *MetricsCollector) SetClient(client client.RouterClient) {
	mc.client = client
	// Set data loader for background preloading
	mc.cache.SetDataLoader(client, mc.config.Cache.TTL/2)
}

func (mc *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	for _, desc := range mc.descriptors {
		ch <- desc
	}
}

func (mc *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	start := time.Now()
	
	// Record collection start
	mc.collectorMetrics.RecordCollectionStart()
	
	// Optimize memory before collection if enabled
	if mc.config.Memory.OptimizeOnCollect {
		mc.memoryMonitor.OptimizeMemory()
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(mc.config.Router.Timeout)*time.Second)
	defer cancel()

	if mc.client == nil {
		logger.Default.Error("Router client not initialized")
		mc.collectorMetrics.RecordCollectionError("collect", "client_not_initialized")
		return
	}

	// Collect data from router
	data, err := mc.collectRouterData(ctx)
	if err != nil {
		logger.Default.Errorf("Failed to collect router data: %v", err)
		mc.collectorMetrics.RecordCollectionError("collect", "data_fetch_failed")
		return
	}

	// Export metrics
	mc.exportSystemMetrics(ch, data)
	mc.exportDeviceMetrics(ch, data)
	mc.exportWANMetrics(ch, data)
	mc.exportWiFiMetrics(ch, data)
	
	// Update memory metrics
	mc.memoryMonitor.UpdateSystemMetrics()
	
	// Record collection completion
	duration := time.Since(start)
	mc.collectorMetrics.RecordCollectionDuration("collect", duration)
	mc.collectorMetrics.RecordCollectionSuccess("collect")
}

type RouterData struct {
	SystemStatus *models.SystemStatus
	DeviceList   *models.DeviceList
	WanInfo      *models.WanInfo
	WifiDetails  *models.WifiDetailAll
}

func (mc *MetricsCollector) collectRouterData(ctx context.Context) (*RouterData, error) {
	start := time.Now()
	
	// Check cache first if enabled
	if mc.config.Cache.Enabled {
		if cachedData := mc.getDataFromCache(); cachedData != nil {
			mc.collectorMetrics.RecordCacheHit("router_data")
			mc.memoryMonitor.RecordOptimization("cache_hit", 0)
			return cachedData, nil
		}
		mc.collectorMetrics.RecordCacheMiss("router_data")
	}
	
	// Use concurrent data fetcher
	result, err := mc.dataFetcher.FetchData(ctx, mc.client)
	if err != nil {
		mc.collectorMetrics.RecordDataFetchError("router_data", "fetch_failed")
		return nil, fmt.Errorf("failed to fetch router data: %w", err)
	}
	
	// Update cache if enabled
	if mc.config.Cache.Enabled {
		mc.updateCache(result)
	}
	
	// Convert to our RouterData type
	data := &RouterData{
		SystemStatus: result.SystemStatus,
		DeviceList:   result.DeviceList,
		WanInfo:      result.WanInfo,
		WifiDetails:  result.WifiDetails,
	}
	
	// Record performance metrics
	duration := time.Since(start)
	mc.collectorMetrics.RecordDataFetchDuration("router_data", "api", duration)
	mc.collectorMetrics.RecordDataFetchSuccess("router_data")
	
	return data, nil
}

// getDataFromCache attempts to get all data from cache
func (mc *MetricsCollector) getDataFromCache() *RouterData {
	data := &RouterData{}
	
	if status, found := mc.cache.GetSystemStatus(); found {
		data.SystemStatus = status
	} else {
		return nil
	}
	
	if devices, found := mc.cache.GetDeviceList(); found {
		data.DeviceList = devices
	} else {
		return nil
	}
	
	if wan, found := mc.cache.GetWanInfo(); found {
		data.WanInfo = wan
	} else {
		return nil
	}
	
	if wifi, found := mc.cache.GetWifiDetails(); found {
		data.WifiDetails = wifi
	} else {
		return nil
	}
	
	return data
}

// updateCache updates the cache with new data
func (mc *MetricsCollector) updateCache(data *concurrent.RouterData) {
	if data.SystemStatus != nil {
		mc.cache.SetSystemStatus(data.SystemStatus)
	}
	if data.DeviceList != nil {
		mc.cache.SetDeviceList(data.DeviceList)
	}
	if data.WanInfo != nil {
		mc.cache.SetWanInfo(data.WanInfo)
	}
	if data.WifiDetails != nil {
		mc.cache.SetWifiDetails(data.WifiDetails)
	}
}

func (mc *MetricsCollector) exportSystemMetrics(ch chan<- prometheus.Metric, data *RouterData) {
	if data.SystemStatus == nil {
		return
	}
	
	host := mc.config.Router.Host
	
	// CPU metrics
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["cpu_cores"],
		prometheus.GaugeValue,
		float64(data.SystemStatus.CPU.Core),
		host,
	)
	
	cpuFreq := utils.ParseCPUFrequency(data.SystemStatus.CPU.Hz)
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["cpu_mhz"],
		prometheus.GaugeValue,
		cpuFreq,
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["cpu_load"],
		prometheus.GaugeValue,
		data.SystemStatus.CPU.Load,
		host,
	)
	
	// Memory metrics
	memTotal := utils.ParseMemorySize(data.SystemStatus.Mem.Total)
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["memory_total_mb"],
		prometheus.GaugeValue,
		memTotal,
		host,
	)
	
	memUsage := data.SystemStatus.Mem.Usage * memTotal
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["memory_usage_mb"],
		prometheus.GaugeValue,
		memUsage,
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["memory_usage"],
		prometheus.GaugeValue,
		data.SystemStatus.Mem.Usage,
		host,
	)
	
	// Device count metrics
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["count_all"],
		prometheus.GaugeValue,
		float64(data.SystemStatus.Count.All),
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["count_online"],
		prometheus.GaugeValue,
		float64(data.SystemStatus.Count.Online),
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["count_all_without_mash"],
		prometheus.GaugeValue,
		float64(data.SystemStatus.Count.AllWithoutMash),
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["count_online_without_mash"],
		prometheus.GaugeValue,
		float64(data.SystemStatus.Count.OnlineWithoutMash),
		host,
	)
	
	// Uptime
	if uptime, err := strconv.ParseFloat(data.SystemStatus.UpTime, 64); err == nil {
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["uptime"],
			prometheus.GaugeValue,
			uptime,
			host,
		)
	}
	
	// Hardware info
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["platform"],
		prometheus.GaugeValue,
		1,
		data.SystemStatus.Hardware.Platform,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["version"],
		prometheus.GaugeValue,
		1,
		data.SystemStatus.Hardware.Version,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["sn"],
		prometheus.GaugeValue,
		1,
		data.SystemStatus.Hardware.Sn,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["mac"],
		prometheus.GaugeValue,
		1,
		data.SystemStatus.Hardware.Mac,
	)
}

func (mc *MetricsCollector) exportDeviceMetrics(ch chan<- prometheus.Metric, data *RouterData) {
	if data.SystemStatus == nil || data.DeviceList == nil {
		return
	}
	
	// Process device traffic from system status
	for _, dev := range data.SystemStatus.Dev {
		devUpload, _ := utils.InterfaceToFloat64(dev.Upload)
		devDownload, _ := utils.InterfaceToFloat64(dev.Download)
		
		var devIP, devName, devIsAP string
		devMac := dev.Mac
		
		// Find device info from device list
		for _, device := range data.DeviceList.List {
			if device.Mac == dev.Mac && len(device.IP) > 0 {
				devIP = device.IP[0].IP
				devName = device.Name
				devIsAP = strconv.Itoa(device.IsAP)
				break
			}
		}
		
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["device_upload_traffic"],
			prometheus.GaugeValue,
			devUpload,
			devIP, devMac, devName, devIsAP,
		)
		
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["device_download_traffic"],
			prometheus.GaugeValue,
			devDownload,
			devIP, devMac, devName, devIsAP,
		)
	}
	
	// Process device speed and online time from device list
	for _, dev := range data.DeviceList.List {
		if len(dev.IP) > 0 {
			devIP := dev.IP[0].IP
			devMac := dev.Mac
			devName := dev.Name
			devIsAP := strconv.Itoa(dev.IsAP)
			
			devOnlineTime, _ := utils.InterfaceToFloat64(dev.Statistics.Online)
			devUpSpeed, _ := utils.InterfaceToFloat64(dev.Statistics.UpSpeed)
			devDownSpeed, _ := utils.InterfaceToFloat64(dev.Statistics.DownSpeed)
			
			ch <- prometheus.MustNewConstMetric(
				mc.descriptors["device_upload_speed"],
				prometheus.GaugeValue,
				devUpSpeed,
				devIP, devMac, devName, devIsAP,
			)
			
			ch <- prometheus.MustNewConstMetric(
				mc.descriptors["device_download_speed"],
				prometheus.GaugeValue,
				devDownSpeed,
				devIP, devMac, devName, devIsAP,
			)
			
			ch <- prometheus.MustNewConstMetric(
				mc.descriptors["device_online_time"],
				prometheus.GaugeValue,
				devOnlineTime,
				devIP, devMac, devName, devIsAP,
			)
		}
	}
}

func (mc *MetricsCollector) exportWANMetrics(ch chan<- prometheus.Metric, data *RouterData) {
	if data.SystemStatus == nil || data.WanInfo == nil {
		return
	}
	
	host := mc.config.Router.Host
	
	// WAN speed and traffic from system status
	wanUpSpeed, _ := strconv.ParseFloat(data.SystemStatus.Wan.UpSpeed, 64)
	wanDownSpeed, _ := strconv.ParseFloat(data.SystemStatus.Wan.DownSpeed, 64)
	wanUpload, _ := strconv.ParseFloat(data.SystemStatus.Wan.Upload, 64)
	wanDownload, _ := strconv.ParseFloat(data.SystemStatus.Wan.Download, 64)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["wan_upload_speed"],
		prometheus.GaugeValue,
		wanUpSpeed,
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["wan_download_speed"],
		prometheus.GaugeValue,
		wanDownSpeed,
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["wan_upload_traffic"],
		prometheus.GaugeValue,
		wanUpload,
		host,
	)
	
	ch <- prometheus.MustNewConstMetric(
		mc.descriptors["wan_download_traffic"],
		prometheus.GaugeValue,
		wanDownload,
		host,
	)
	
	// IP addresses from WAN info
	for _, ipv4 := range data.WanInfo.Info.Ipv4 {
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["ipv4"],
			prometheus.GaugeValue,
			1,
			ipv4.IP,
		)
		
		if mask, err := utils.SubNetMaskToLen(ipv4.Mask); err == nil {
			ch <- prometheus.MustNewConstMetric(
				mc.descriptors["ipv4_mask"],
				prometheus.GaugeValue,
				float64(mask),
				ipv4.IP,
			)
		}
	}
	
	for _, ipv6 := range data.WanInfo.Info.Ipv6Info.IP6Addr {
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["ipv6"],
			prometheus.GaugeValue,
			1,
			ipv6,
		)
	}
}

func (mc *MetricsCollector) exportWiFiMetrics(ch chan<- prometheus.Metric, data *RouterData) {
	if data.WifiDetails == nil {
		return
	}
	
	for _, info := range data.WifiDetails.Info {
		status, _ := utils.InterfaceToFloat64(info.Status)
		
		bandList := ""
		for i, band := range info.ChannelInfo.BandList {
			bandList += band
			if i != len(info.ChannelInfo.BandList)-1 {
				bandList += "/"
			} else {
				bandList += "MHz"
			}
		}
		
		channel := strconv.Itoa(info.ChannelInfo.Channel)
		
		ch <- prometheus.MustNewConstMetric(
			mc.descriptors["wifi_detail"],
			prometheus.GaugeValue,
			status,
			info.Ssid, info.Status, bandList, channel,
		)
	}
}

func (mc *MetricsCollector) GetRegistry() *prometheus.Registry {
	return mc.metrics
}

func (mc *MetricsCollector) Close() error {
	if mc.cache != nil {
		mc.cache.Stop()
	}
	
	// Final memory optimization before shutdown if enabled
	if mc.memoryMonitor != nil && mc.config.Memory.ForceGCOnClose {
		mc.memoryMonitor.OptimizeMemory()
		mc.memoryMonitor.ForceGC()
	}
	
	return nil
}