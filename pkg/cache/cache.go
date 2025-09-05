package cache

import (
	"sync"
	"time"
)

// Cache represents a generic cache interface
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
}

// CacheItem represents a cached item with expiration
type CacheItem struct {
	value      interface{}
	expiration time.Time
}

// IsExpired checks if the cache item has expired
func (item *CacheItem) IsExpired() bool {
	return !item.expiration.IsZero() && time.Now().After(item.expiration)
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
	items map[string]*CacheItem
	mu    sync.RWMutex
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]*CacheItem),
	}
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return nil, false
	}
	
	if item.IsExpired() {
		return nil, false
	}
	
	return item.value, true
}

// Set stores a value in cache with TTL
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}
	
	c.items[key] = &CacheItem{
		value:      value,
		expiration: expiration,
	}
}

// Delete removes a value from cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.items, key)
}

// Clear removes all items from cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.items = make(map[string]*CacheItem)
}

// Cleanup removes expired items from cache
func (c *MemoryCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	for key, item := range c.items {
		if item.IsExpired() {
			delete(c.items, key)
		}
	}
}

// RouterCache is a specialized cache for router data
type RouterCache struct {
	cache     Cache
	ttl       time.Duration
	cleanup   *time.Ticker
	stop      chan struct{}
}

// NewRouterCache creates a new router cache with automatic cleanup
func NewRouterCache(ttl time.Duration) *RouterCache {
	rc := &RouterCache{
		cache: NewMemoryCache(),
		ttl:   ttl,
		stop:  make(chan struct{}),
	}
	
	// Start cleanup routine
	rc.cleanup = time.NewTicker(ttl / 2)
	go func() {
		for {
			select {
			case <-rc.cleanup.C:
				rc.cache.(*MemoryCache).Cleanup()
			case <-rc.stop:
				return
			}
		}
	}()
	
	return rc
}

// GetSystemStatus retrieves system status from cache
func (rc *RouterCache) GetSystemStatus() (interface{}, bool) {
	return rc.cache.Get("system_status")
}

// SetSystemStatus stores system status in cache
func (rc *RouterCache) SetSystemStatus(value interface{}) {
	rc.cache.Set("system_status", value, rc.ttl)
}

// GetDeviceList retrieves device list from cache
func (rc *RouterCache) GetDeviceList() (interface{}, bool) {
	return rc.cache.Get("device_list")
}

// SetDeviceList stores device list in cache
func (rc *RouterCache) SetDeviceList(value interface{}) {
	rc.cache.Set("device_list", value, rc.ttl)
}

// GetWanInfo retrieves WAN info from cache
func (rc *RouterCache) GetWanInfo() (interface{}, bool) {
	return rc.cache.Get("wan_info")
}

// SetWanInfo stores WAN info in cache
func (rc *RouterCache) SetWanInfo(value interface{}) {
	rc.cache.Set("wan_info", value, rc.ttl)
}

// GetWifiDetails retrieves WiFi details from cache
func (rc *RouterCache) GetWifiDetails() (interface{}, bool) {
	return rc.cache.Get("wifi_details")
}

// SetWifiDetails stores WiFi details in cache
func (rc *RouterCache) SetWifiDetails(value interface{}) {
	rc.cache.Set("wifi_details", value, rc.ttl)
}

// Stop stops the cache cleanup routine
func (rc *RouterCache) Stop() {
	close(rc.stop)
	rc.cleanup.Stop()
}