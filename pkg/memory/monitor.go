package memory

import (
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MemoryMonitor provides comprehensive memory usage monitoring
type MemoryMonitor struct {
	// Metrics
	allocGauge        *prometheus.GaugeVec
	sysMemoryGauge    *prometheus.GaugeVec
	gcGauge           *prometheus.GaugeVec
	poolStats         *prometheus.GaugeVec
	allocationCounter *prometheus.CounterVec
	
	// Memory pools
	bufferPool      *BufferPool
	jsonPool        *ObjectPool
	requestPool     *ObjectPool
	responsePool    *ObjectPool
	
	// Tracking
	mu              sync.RWMutex
	lastGC          time.Time
	gcCount         uint32
	allocations     map[string]int64
	optimizations   map[string]int64
	
	// Configuration
	trackAllocations bool
	enableGCStats    bool
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(namespace string) *MemoryMonitor {
	return &MemoryMonitor{
		allocGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_alloc_bytes",
				Help:      "Current memory allocation in bytes",
			},
			[]string{"type"},
		),
		sysMemoryGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_system_bytes",
				Help:      "System memory usage in bytes",
			},
			[]string{"type"},
		),
		gcGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_gc_stats",
				Help:      "Garbage collection statistics",
			},
			[]string{"stat"},
		),
		poolStats: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_pool_stats",
				Help:      "Memory pool statistics",
			},
			[]string{"pool", "stat"},
		),
		allocationCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "memory_allocations_total",
				Help:      "Total number of memory allocations",
			},
			[]string{"type", "action"},
		),
		bufferPool:      NewBufferPool(),
		jsonPool:        NewObjectPool(func() interface{} { return map[string]interface{}{} }),
		requestPool:     NewObjectPool(func() interface{} { return []byte{} }),
		responsePool:    NewObjectPool(func() interface{} { return []byte{} }),
		allocations:     make(map[string]int64),
		optimizations:   make(map[string]int64),
		trackAllocations: true,
		enableGCStats:    true,
	}
}

// Configure configures the memory monitor with settings
func (mm *MemoryMonitor) Configure(enabled, optimizeOnCollect, forceGCOnClose, trackAllocations, enablePoolStats bool) {
	mm.trackAllocations = trackAllocations
	mm.enableGCStats = enablePoolStats
}

// Describe implements prometheus.Collector
func (mm *MemoryMonitor) Describe(ch chan<- *prometheus.Desc) {
	mm.allocGauge.Describe(ch)
	mm.sysMemoryGauge.Describe(ch)
	mm.gcGauge.Describe(ch)
	mm.poolStats.Describe(ch)
	mm.allocationCounter.Describe(ch)
}

// Collect implements prometheus.Collector
func (mm *MemoryMonitor) Collect(ch chan<- prometheus.Metric) {
	mm.updateMetrics()
	
	mm.allocGauge.Collect(ch)
	mm.sysMemoryGauge.Collect(ch)
	mm.gcGauge.Collect(ch)
	mm.poolStats.Collect(ch)
	mm.allocationCounter.Collect(ch)
}

// updateMetrics updates all memory metrics
func (mm *MemoryMonitor) updateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Allocation metrics
	mm.allocGauge.WithLabelValues("heap").Set(float64(m.HeapAlloc))
	mm.allocGauge.WithLabelValues("stack").Set(float64(m.StackInuse))
	mm.allocGauge.WithLabelValues("total").Set(float64(m.TotalAlloc))
	
	// System memory metrics
	mm.sysMemoryGauge.WithLabelValues("sys").Set(float64(m.Sys))
	mm.sysMemoryGauge.WithLabelValues("heap_sys").Set(float64(m.HeapSys))
	mm.sysMemoryGauge.WithLabelValues("stack_sys").Set(float64(m.StackSys))
	
	// GC stats
	if mm.enableGCStats {
		mm.gcGauge.WithLabelValues("num_gc").Set(float64(m.NumGC))
		mm.gcGauge.WithLabelValues("pause_total_us").Set(float64(m.PauseTotalNs) / 1000)
		mm.gcGauge.WithLabelValues("next_gc_mb").Set(float64(m.NextGC) / 1024 / 1024)
	}
	
	// Pool stats
	mm.updatePoolStats()
}

// updatePoolStats updates memory pool statistics
func (mm *MemoryMonitor) updatePoolStats() {
	// Buffer pool stats
	smallCreated, smallReused := mm.bufferPool.small.Stats()
	mediumCreated, mediumReused := mm.bufferPool.medium.Stats()
	largeCreated, largeReused := mm.bufferPool.large.Stats()
	xlargeCreated, xlargeReused := mm.bufferPool.xlarge.Stats()
	
	mm.poolStats.WithLabelValues("buffer_small", "created").Set(float64(smallCreated))
	mm.poolStats.WithLabelValues("buffer_small", "reused").Set(float64(smallReused))
	mm.poolStats.WithLabelValues("buffer_medium", "created").Set(float64(mediumCreated))
	mm.poolStats.WithLabelValues("buffer_medium", "reused").Set(float64(mediumReused))
	mm.poolStats.WithLabelValues("buffer_large", "created").Set(float64(largeCreated))
	mm.poolStats.WithLabelValues("buffer_large", "reused").Set(float64(largeReused))
	mm.poolStats.WithLabelValues("buffer_xlarge", "created").Set(float64(xlargeCreated))
	mm.poolStats.WithLabelValues("buffer_xlarge", "reused").Set(float64(xlargeReused))
	
	// Object pool stats
	jsonCreated, jsonReused := mm.jsonPool.Stats()
	requestCreated, requestReused := mm.requestPool.Stats()
	responseCreated, responseReused := mm.responsePool.Stats()
	
	mm.poolStats.WithLabelValues("json", "created").Set(float64(jsonCreated))
	mm.poolStats.WithLabelValues("json", "reused").Set(float64(jsonReused))
	mm.poolStats.WithLabelValues("request", "created").Set(float64(requestCreated))
	mm.poolStats.WithLabelValues("request", "reused").Set(float64(requestReused))
	mm.poolStats.WithLabelValues("response", "created").Set(float64(responseCreated))
	mm.poolStats.WithLabelValues("response", "reused").Set(float64(responseReused))
}

// TrackAllocation tracks a memory allocation
func (mm *MemoryMonitor) TrackAllocation(key string, size int64) {
	if !mm.trackAllocations {
		return
	}
	
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.allocations[key] += size
	mm.allocationCounter.WithLabelValues(key, "allocate").Inc()
}

// TrackDeallocation tracks a memory deallocation
func (mm *MemoryMonitor) TrackDeallocation(key string, size int64) {
	if !mm.trackAllocations {
		return
	}
	
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	if current, exists := mm.allocations[key]; exists {
		if current > size {
			mm.allocations[key] = current - size
		} else {
			delete(mm.allocations, key)
		}
	}
	mm.allocationCounter.WithLabelValues(key, "deallocate").Inc()
}

// RecordOptimization records a memory optimization
func (mm *MemoryMonitor) RecordOptimization(typeName string, bytesSaved int64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	mm.optimizations[typeName] += bytesSaved
}

// GetBuffer returns a buffer from the pool
func (mm *MemoryMonitor) GetBuffer(size int) []byte {
	return mm.bufferPool.GetBuffer(size)
}

// PutBuffer returns a buffer to the pool
func (mm *MemoryMonitor) PutBuffer(buf []byte) {
	mm.bufferPool.PutBuffer(buf)
}

// GetJSONObject returns a JSON object from the pool
func (mm *MemoryMonitor) GetJSONObject() map[string]interface{} {
	return mm.jsonPool.Get().(map[string]interface{})
}

// PutJSONObject returns a JSON object to the pool
func (mm *MemoryMonitor) PutJSONObject(obj map[string]interface{}) {
	// Clear the object before returning to pool
	for k := range obj {
		delete(obj, k)
	}
	mm.jsonPool.Put(obj)
}

// GetRequestBuffer returns a request buffer from the pool
func (mm *MemoryMonitor) GetRequestBuffer() []byte {
	return mm.requestPool.Get().([]byte)
}

// PutRequestBuffer returns a request buffer to the pool
func (mm *MemoryMonitor) PutRequestBuffer(buf []byte) {
	mm.requestPool.Put(buf[:0])
}

// GetResponseBuffer returns a response buffer from the pool
func (mm *MemoryMonitor) GetResponseBuffer() []byte {
	return mm.responsePool.Get().([]byte)
}

// PutResponseBuffer returns a response buffer to the pool
func (mm *MemoryMonitor) PutResponseBuffer(buf []byte) {
	mm.responsePool.Put(buf[:0])
}

// ForceGC forces garbage collection and updates stats
func (mm *MemoryMonitor) ForceGC() {
	runtime.GC()
	mm.mu.Lock()
	mm.lastGC = time.Now()
	mm.gcCount++
	mm.mu.Unlock()
}

// GetStats returns memory statistics
func (mm *MemoryMonitor) GetStats() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	stats := map[string]interface{}{
		"heap_alloc":        m.HeapAlloc,
		"heap_sys":          m.HeapSys,
		"total_alloc":       m.TotalAlloc,
		"sys":               m.Sys,
		"num_gc":            m.NumGC,
		"pause_total_ns":    m.PauseTotalNs,
		"next_gc":           m.NextGC,
		"gc_count":          mm.gcCount,
		"last_gc":           mm.lastGC,
		"allocations":       len(mm.allocations),
		"optimizations":     mm.optimizations,
	}
	
	// Add pool stats
	bufferCreated, bufferReused := mm.bufferPool.Stats()
	stats["buffer_pool_created"] = bufferCreated
	stats["buffer_pool_reused"] = bufferReused
	
	jsonCreated, jsonReused := mm.jsonPool.Stats()
	stats["json_pool_created"] = jsonCreated
	stats["json_pool_reused"] = jsonReused
	
	return stats
}

// OptimizeMemory performs memory optimizations
func (mm *MemoryMonitor) OptimizeMemory() {
	// Force garbage collection
	mm.ForceGC()
	
	// Reset pools if they're too large
	mm.resetPoolsIfNeeded()
	
	// Record optimization
	mm.RecordOptimization("gc_optimization", 0)
}

// resetPoolsIfNeeded resets pools if they've grown too large
func (mm *MemoryMonitor) resetPoolsIfNeeded() {
	// This could be enhanced with logic to track pool sizes
	// and reset them when they exceed certain thresholds
}

// MemoryUsageSnapshot captures a snapshot of current memory usage
type MemoryUsageSnapshot struct {
	Timestamp       time.Time              `json:"timestamp"`
	HeapAlloc       uint64                 `json:"heap_alloc"`
	HeapSys         uint64                 `json:"heap_sys"`
	TotalAlloc      uint64                 `json:"total_alloc"`
	Sys             uint64                 `json:"sys"`
	NumGC           uint32                 `json:"num_gc"`
	PauseTotalNs    uint64                 `json:"pause_total_ns"`
	NextGC          uint64                 `json:"next_gc"`
	Allocations     map[string]int64       `json:"allocations"`
	Optimizations   map[string]int64       `json:"optimizations"`
	PoolStats       map[string]interface{} `json:"pool_stats"`
}

// TakeSnapshot takes a memory usage snapshot
func (mm *MemoryMonitor) TakeSnapshot() *MemoryUsageSnapshot {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	snapshot := &MemoryUsageSnapshot{
		Timestamp:     time.Now(),
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		NumGC:         m.NumGC,
		PauseTotalNs:  m.PauseTotalNs,
		NextGC:        m.NextGC,
		Allocations:   make(map[string]int64),
		Optimizations: make(map[string]int64),
		PoolStats:     make(map[string]interface{}),
	}
	
	// Copy allocations
	for k, v := range mm.allocations {
		snapshot.Allocations[k] = v
	}
	
	// Copy optimizations
	for k, v := range mm.optimizations {
		snapshot.Optimizations[k] = v
	}
	
	// Add pool stats
	bufferCreated, bufferReused := mm.bufferPool.Stats()
	snapshot.PoolStats["buffer_created"] = bufferCreated
	snapshot.PoolStats["buffer_reused"] = bufferReused
	
	return snapshot
}

// UpdateSystemMetrics updates system metrics
func (mm *MemoryMonitor) UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Update memory metrics
	mm.sysMemoryGauge.WithLabelValues("heap_alloc").Set(float64(m.HeapAlloc))
	mm.sysMemoryGauge.WithLabelValues("heap_sys").Set(float64(m.HeapSys))
	mm.sysMemoryGauge.WithLabelValues("total_alloc").Set(float64(m.TotalAlloc))
	mm.sysMemoryGauge.WithLabelValues("sys").Set(float64(m.Sys))
	
	// Update GC metrics
	mm.gcGauge.WithLabelValues("num_gc").Set(float64(m.NumGC))
	mm.gcGauge.WithLabelValues("pause_total_us").Set(float64(m.PauseTotalNs) / 1000)
	mm.gcGauge.WithLabelValues("next_gc_mb").Set(float64(m.NextGC) / 1024 / 1024)
}