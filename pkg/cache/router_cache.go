package cache

import (
	"context"
	"sync"
	"time"

	"github.com/helloworlde/miwifi-exporter/internal/models"
)

// RouterSmartCache is a specialized smart cache for router data
type RouterSmartCache struct {
	cache      *SmartCache
	ttl        time.Duration
	preload    bool
	mu         sync.RWMutex
	background *BackgroundLoader
}

// BackgroundLoader handles background data loading
type BackgroundLoader struct {
	cache      *RouterSmartCache
	dataLoader DataLoader
	interval   time.Duration
	stop       chan struct{}
}

// DataLoader defines the interface for loading router data
type DataLoader interface {
	GetSystemStatus(ctx context.Context) (*models.SystemStatus, error)
	GetDeviceList(ctx context.Context) (*models.DeviceList, error)
	GetWanInfo(ctx context.Context) (*models.WanInfo, error)
	GetWifiDetails(ctx context.Context) (*models.WifiDetailAll, error)
}

// NewRouterSmartCache creates a new smart router cache
func NewRouterSmartCache(ttl time.Duration, sizeLimit int, preload bool) *RouterSmartCache {
	return &RouterSmartCache{
		cache:   NewSmartCache(ttl, sizeLimit),
		ttl:     ttl,
		preload: preload,
	}
}

// SetDataLoader sets the data loader for background preloading
func (rc *RouterSmartCache) SetDataLoader(loader DataLoader, interval time.Duration) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	// Stop existing background loader
	if rc.background != nil {
		rc.background.Stop()
	}
	
	rc.background = &BackgroundLoader{
		cache:      rc,
		dataLoader: loader,
		interval:   interval,
		stop:       make(chan struct{}),
	}
	
	if rc.preload {
		go rc.background.Start()
	}
}

// GetSystemStatus retrieves system status from cache
func (rc *RouterSmartCache) GetSystemStatus() (*models.SystemStatus, bool) {
	if value, found := rc.cache.Get("system_status"); found {
		return value.(*models.SystemStatus), true
	}
	return nil, false
}

// SetSystemStatus stores system status in cache
func (rc *RouterSmartCache) SetSystemStatus(value *models.SystemStatus) {
	rc.cache.Set("system_status", value, rc.ttl)
}

// GetDeviceList retrieves device list from cache
func (rc *RouterSmartCache) GetDeviceList() (*models.DeviceList, bool) {
	if value, found := rc.cache.Get("device_list"); found {
		return value.(*models.DeviceList), true
	}
	return nil, false
}

// SetDeviceList stores device list in cache
func (rc *RouterSmartCache) SetDeviceList(value *models.DeviceList) {
	rc.cache.Set("device_list", value, rc.ttl)
}

// GetWanInfo retrieves WAN info from cache
func (rc *RouterSmartCache) GetWanInfo() (*models.WanInfo, bool) {
	if value, found := rc.cache.Get("wan_info"); found {
		return value.(*models.WanInfo), true
	}
	return nil, false
}

// SetWanInfo stores WAN info in cache
func (rc *RouterSmartCache) SetWanInfo(value *models.WanInfo) {
	rc.cache.Set("wan_info", value, rc.ttl)
}

// GetWifiDetails retrieves WiFi details from cache
func (rc *RouterSmartCache) GetWifiDetails() (*models.WifiDetailAll, bool) {
	if value, found := rc.cache.Get("wifi_details"); found {
		return value.(*models.WifiDetailAll), true
	}
	return nil, false
}

// SetWifiDetails stores WiFi details in cache
func (rc *RouterSmartCache) SetWifiDetails(value *models.WifiDetailAll) {
	rc.cache.Set("wifi_details", value, rc.ttl)
}

// GetStats returns cache statistics
func (rc *RouterSmartCache) GetStats() *CacheStats {
	return rc.cache.GetStats()
}

// Clear clears all cached data
func (rc *RouterSmartCache) Clear() {
	rc.cache.Clear()
}

// Stop stops the cache and background loader
func (rc *RouterSmartCache) Stop() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	if rc.background != nil {
		rc.background.Stop()
		rc.background = nil
	}
	
	rc.cache.Stop()
}

// PreloadData preloads all data into cache
func (rc *RouterSmartCache) PreloadData(ctx context.Context, loader DataLoader) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 4)
	
	wg.Add(4)
	
	// Load system status
	go func() {
		defer wg.Done()
		if status, err := loader.GetSystemStatus(ctx); err == nil {
			rc.SetSystemStatus(status)
		} else {
			errChan <- err
		}
	}()
	
	// Load device list
	go func() {
		defer wg.Done()
		if devices, err := loader.GetDeviceList(ctx); err == nil {
			rc.SetDeviceList(devices)
		} else {
			errChan <- err
		}
	}()
	
	// Load WAN info
	go func() {
		defer wg.Done()
		if wan, err := loader.GetWanInfo(ctx); err == nil {
			rc.SetWanInfo(wan)
		} else {
			errChan <- err
		}
	}()
	
	// Load WiFi details
	go func() {
		defer wg.Done()
		if wifi, err := loader.GetWifiDetails(ctx); err == nil {
			rc.SetWifiDetails(wifi)
		} else {
			errChan <- err
		}
	}()
	
	wg.Wait()
	close(errChan)
	
	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Start starts the background loader
func (bl *BackgroundLoader) Start() {
	ticker := time.NewTicker(bl.interval)
	
	go func() {
		for {
			select {
			case <-ticker.C:
				bl.preloadData()
			case <-bl.stop:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the background loader
func (bl *BackgroundLoader) Stop() {
	close(bl.stop)
}

// preloadData preloads data in the background
func (bl *BackgroundLoader) preloadData() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Ignore errors in background loading
	_ = bl.cache.PreloadData(ctx, bl.dataLoader)
}