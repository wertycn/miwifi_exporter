package memory

import (
	"sync"
	"time"
)

// MemoryPool implements a sync.Pool for reusing memory allocations
type MemoryPool struct {
	pool     sync.Pool
	maxSize  int
	created  int64
	reused   int64
	mu       sync.Mutex
}

// NewMemoryPool creates a new memory pool with optimal sizing
func NewMemoryPool(maxSize int) *MemoryPool {
	return &MemoryPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, maxSize)
			},
		},
		maxSize: maxSize,
	}
}

// Get returns a byte slice from the pool
func (mp *MemoryPool) Get() []byte {
	mp.mu.Lock()
	mp.reused++
	mp.mu.Unlock()
	return mp.pool.Get().([]byte)
}

// Put returns a byte slice to the pool
func (mp *MemoryPool) Put(buf []byte) {
	if cap(buf) <= mp.maxSize {
		mp.mu.Lock()
		mp.created++
		mp.mu.Unlock()
		mp.pool.Put(buf[:0])
	}
}

// Stats returns memory pool statistics
func (mp *MemoryPool) Stats() (created int64, reused int64) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.created, mp.reused
}

// BufferPool manages reusable buffers for different sizes
type BufferPool struct {
	small   *MemoryPool // 1KB
	medium  *MemoryPool // 8KB
	large   *MemoryPool // 64KB
	xlarge  *MemoryPool // 512KB
	created int64
	reused  int64
	mu      sync.Mutex
}

// NewBufferPool creates a new buffer pool with multiple size categories
func NewBufferPool() *BufferPool {
	return &BufferPool{
		small:  NewMemoryPool(1024),     // 1KB
		medium: NewMemoryPool(8192),     // 8KB
		large:  NewMemoryPool(65536),    // 64KB
		xlarge: NewMemoryPool(524288),   // 512KB
	}
}

// GetBuffer returns a buffer of appropriate size
func (bp *BufferPool) GetBuffer(size int) []byte {
	var buf []byte
	
	switch {
	case size <= 1024:
		buf = bp.small.Get()
	case size <= 8192:
		buf = bp.medium.Get()
	case size <= 65536:
		buf = bp.large.Get()
	default:
		buf = bp.xlarge.Get()
	}
	
	bp.mu.Lock()
	bp.reused++
	bp.mu.Unlock()
	
	return buf
}

// PutBuffer returns a buffer to the appropriate pool
func (bp *BufferPool) PutBuffer(buf []byte) {
	capacity := cap(buf)
	
	bp.mu.Lock()
	bp.created++
	bp.mu.Unlock()
	
	switch {
	case capacity <= 1024:
		bp.small.Put(buf)
	case capacity <= 8192:
		bp.medium.Put(buf)
	case capacity <= 65536:
		bp.large.Put(buf)
	default:
		bp.xlarge.Put(buf)
	}
}

// Stats returns buffer pool statistics
func (bp *BufferPool) Stats() (created int64, reused int64) {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.created, bp.reused
}

// ObjectPool provides generic object pooling
type ObjectPool struct {
	pool     sync.Pool
	created  int64
	reused   int64
	mu       sync.Mutex
	newFunc  func() interface{}
}

// NewObjectPool creates a new object pool
func NewObjectPool(newFunc func() interface{}) *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: newFunc,
		},
		newFunc: newFunc,
	}
}

// Get returns an object from the pool
func (op *ObjectPool) Get() interface{} {
	op.mu.Lock()
	op.reused++
	op.mu.Unlock()
	return op.pool.Get()
}

// Put returns an object to the pool
func (op *ObjectPool) Put(obj interface{}) {
	op.mu.Lock()
	op.created++
	op.mu.Unlock()
	op.pool.Put(obj)
}

// Stats returns object pool statistics
func (op *ObjectPool) Stats() (created int64, reused int64) {
	op.mu.Lock()
	defer op.mu.Unlock()
	return op.created, op.reused
}

// MemoryTracker tracks memory usage over time
type MemoryTracker struct {
	allocations map[string]int64
	timestamps  map[string]time.Time
	mu          sync.RWMutex
}

// NewMemoryTracker creates a new memory tracker
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		allocations: make(map[string]int64),
		timestamps:  make(map[string]time.Time),
	}
}

// TrackAllocation tracks a memory allocation
func (mt *MemoryTracker) TrackAllocation(key string, size int64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	mt.allocations[key] = size
	mt.timestamps[key] = time.Now()
}

// TrackDeallocation tracks a memory deallocation
func (mt *MemoryTracker) TrackDeallocation(key string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	delete(mt.allocations, key)
	delete(mt.timestamps, key)
}

// GetTotalAllocated returns total allocated memory
func (mt *MemoryTracker) GetTotalAllocated() int64 {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	var total int64
	for _, size := range mt.allocations {
		total += size
	}
	return total
}

// GetAllocationCount returns number of active allocations
func (mt *MemoryTracker) GetAllocationCount() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return len(mt.allocations)
}

// GetStats returns memory tracking statistics
func (mt *MemoryTracker) GetStats() (total int64, count int, oldest time.Time) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	for key, size := range mt.allocations {
		total += size
		count++
		if timestamp, exists := mt.timestamps[key]; exists && (oldest.IsZero() || timestamp.Before(oldest)) {
			oldest = timestamp
		}
	}
	
	return total, count, oldest
}

// OptimizedStringSlice provides memory-efficient string slice operations
type OptimizedStringSlice struct {
	data []string
	mu   sync.RWMutex
}

// NewOptimizedStringSlice creates a new optimized string slice
func NewOptimizedStringSlice(initialCapacity int) *OptimizedStringSlice {
	return &OptimizedStringSlice{
		data: make([]string, 0, initialCapacity),
	}
}

// Append adds strings with efficient capacity management
func (oss *OptimizedStringSlice) Append(items ...string) {
	oss.mu.Lock()
	defer oss.mu.Unlock()
	
	newLen := len(oss.data) + len(items)
	if cap(oss.data) < newLen {
		// Grow by 50% to reduce frequent allocations
		newCap := newLen * 3 / 2
		if newCap < 16 {
			newCap = 16
		}
		newData := make([]string, len(oss.data), newCap)
		copy(newData, oss.data)
		oss.data = newData
	}
	
	oss.data = append(oss.data, items...)
}

// Get returns the string slice
func (oss *OptimizedStringSlice) Get() []string {
	oss.mu.RLock()
	defer oss.mu.RUnlock()
	return oss.data
}

// Clear resets the slice while retaining capacity
func (oss *OptimizedStringSlice) Clear() {
	oss.mu.Lock()
	defer oss.mu.Unlock()
	oss.data = oss.data[:0]
}

// Len returns the length of the slice
func (oss *OptimizedStringSlice) Len() int {
	oss.mu.RLock()
	defer oss.mu.RUnlock()
	return len(oss.data)
}

// Cap returns the capacity of the slice
func (oss *OptimizedStringSlice) Cap() int {
	oss.mu.RLock()
	defer oss.mu.RUnlock()
	return cap(oss.data)
}