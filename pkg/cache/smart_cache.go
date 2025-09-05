package cache

import (
	"context"
	"sync"
	"time"
)

// SmartCache represents an intelligent caching system
type SmartCache struct {
	items      map[string]*SmartCacheItem
	mu         sync.RWMutex
	ttl        time.Duration
	cleanup    *time.Ticker
	stop       chan struct{}
	stats      *CacheStats
	hits       int64
	misses     int64
	evictions  int64
	sizeLimit  int
	cleanupCtx context.Context
	cleanupCancel context.CancelFunc
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits      int64 `json:"hits"`
	Misses    int64 `json:"misses"`
	Evictions int64 `json:"evictions"`
	Size      int   `json:"size"`
	HitRate   float64 `json:"hit_rate"`
}

// SmartCacheItem represents a cached item with metadata
type SmartCacheItem struct {
	value       interface{}
	expiration  time.Time
	accessed    time.Time
	accessCount int64
	size        int
}

// NewSmartCache creates a new smart cache with automatic cleanup
func NewSmartCache(ttl time.Duration, sizeLimit int) *SmartCache {
	ctx, cancel := context.WithCancel(context.Background())
	
	sc := &SmartCache{
		items:      make(map[string]*SmartCacheItem),
		ttl:        ttl,
		stop:       make(chan struct{}),
		stats:      &CacheStats{},
		sizeLimit:  sizeLimit,
		cleanupCtx: ctx,
		cleanupCancel: cancel,
	}
	
	// Start cleanup routine
	sc.cleanup = time.NewTicker(ttl / 4)
	go sc.cleanupRoutine()
	
	return sc
}

// Get retrieves a value from cache with access tracking
func (sc *SmartCache) Get(key string) (interface{}, bool) {
	sc.mu.RLock()
	
	item, found := sc.items[key]
	if !found {
		sc.mu.RUnlock()
		sc.misses++
		return nil, false
	}
	
	// Check if expired
	if item.IsExpired() {
		sc.mu.RUnlock()
		sc.deleteExpired(key)
		sc.misses++
		return nil, false
	}
	
	// Update access statistics
	item.accessed = time.Now()
	item.accessCount++
	sc.hits++
	
	sc.mu.RUnlock()
	return item.value, true
}

// Set stores a value in cache with intelligent eviction
func (sc *SmartCache) Set(key string, value interface{}, ttl time.Duration) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	// Calculate item size (rough estimate)
	size := estimateSize(value)
	
	// Check if we need to evict items
	if sc.sizeLimit > 0 && len(sc.items) >= sc.sizeLimit {
		sc.evictLRU(1)
	}
	
	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	} else {
		expiration = time.Now().Add(sc.ttl)
	}
	
	sc.items[key] = &SmartCacheItem{
		value:       value,
		expiration:  expiration,
		accessed:    time.Now(),
		accessCount: 1,
		size:        size,
	}
}

// Delete removes a value from cache
func (sc *SmartCache) Delete(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if _, exists := sc.items[key]; exists {
		delete(sc.items, key)
		sc.evictions++
	}
}

// Clear removes all items from cache
func (sc *SmartCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.items = make(map[string]*SmartCacheItem)
	sc.evictions += int64(len(sc.items))
}

// GetStats returns cache statistics
func (sc *SmartCache) GetStats() *CacheStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	
	total := sc.hits + sc.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(sc.hits) / float64(total)
	}
	
	return &CacheStats{
		Hits:      sc.hits,
		Misses:    sc.misses,
		Evictions: sc.evictions,
		Size:      len(sc.items),
		HitRate:   hitRate,
	}
}

// cleanupRoutine periodically cleans up expired items
func (sc *SmartCache) cleanupRoutine() {
	for {
		select {
		case <-sc.cleanup.C:
			sc.cleanupExpired()
		case <-sc.cleanupCtx.Done():
			return
		case <-sc.stop:
			return
		}
	}
}

// cleanupExpired removes expired items from cache
func (sc *SmartCache) cleanupExpired() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	for key, item := range sc.items {
		if item.IsExpired() {
			delete(sc.items, key)
			sc.evictions++
		}
	}
}

// evictLRU evicts least recently used items
func (sc *SmartCache) evictLRU(count int) {
	if count <= 0 {
		return
	}
	
	// Find items with oldest access time
	var oldestKeys []string
	oldestTime := time.Now()
	
	for key, item := range sc.items {
		if item.accessed.Before(oldestTime) || len(oldestKeys) == 0 {
			oldestKeys = []string{key}
			oldestTime = item.accessed
		} else if item.accessed.Equal(oldestTime) {
			oldestKeys = append(oldestKeys, key)
		}
	}
	
	// Evict items
	for i := 0; i < len(oldestKeys) && i < count; i++ {
		delete(sc.items, oldestKeys[i])
		sc.evictions++
	}
}

// Stop stops the cache cleanup routine
func (sc *SmartCache) Stop() {
	sc.cleanupCancel()
	close(sc.stop)
	sc.cleanup.Stop()
}

// deleteExpired safely deletes an expired item
func (sc *SmartCache) deleteExpired(key string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	if _, exists := sc.items[key]; exists {
		delete(sc.items, key)
		sc.evictions++
	}
}

// estimateSize provides a rough estimate of the value size
func estimateSize(value interface{}) int {
	// This is a simplified size estimation
	// In a real implementation, you might want to use reflection
	// for more accurate size calculation
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case int, int8, int16, int32, int64:
		return 8
	case uint, uint8, uint16, uint32, uint64:
		return 8
	case float32, float64:
		return 8
	case bool:
		return 1
	default:
		return 64 // Default estimate for complex types
	}
}

// IsExpired checks if the cache item has expired
func (item *SmartCacheItem) IsExpired() bool {
	return !item.expiration.IsZero() && time.Now().After(item.expiration)
}