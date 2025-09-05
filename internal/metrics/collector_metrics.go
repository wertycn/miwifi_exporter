package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CollectorMetrics tracks the performance of the metrics collector
type CollectorMetrics struct {
	// Collection metrics
	collectionDuration *prometheus.HistogramVec
	collectionErrors   *prometheus.CounterVec
	collectionSuccess  *prometheus.CounterVec
	
	// Cache metrics
	cacheHits         *prometheus.CounterVec
	cacheMisses       *prometheus.CounterVec
	cacheEvictions    *prometheus.CounterVec
	cacheSize         *prometheus.GaugeVec
	
	// HTTP client metrics
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpRequestErrors   *prometheus.CounterVec
	
	// Data fetcher metrics
	dataFetchDuration   *prometheus.HistogramVec
	dataFetchSuccess    *prometheus.CounterVec
	dataFetchErrors     *prometheus.CounterVec
	dataFetchTimeouts   *prometheus.CounterVec
	
	// System metrics
	memoryUsage     *prometheus.GaugeVec
	goroutines      *prometheus.GaugeVec
	uptime          *prometheus.GaugeVec
	startTime       time.Time
}

// NewCollectorMetrics creates new collector metrics
func NewCollectorMetrics(namespace string) *CollectorMetrics {
	return &CollectorMetrics{
		// Collection metrics
		collectionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "collection_duration_seconds",
				Help:      "Duration of metrics collection",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		collectionErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "collection_errors_total",
				Help:      "Total number of collection errors",
			},
			[]string{"operation", "error_type"},
		),
		collectionSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "collection_success_total",
				Help:      "Total number of successful collections",
			},
			[]string{"operation"},
		),
		
		// Cache metrics
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_type"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_type"},
		),
		cacheEvictions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_evictions_total",
				Help:      "Total number of cache evictions",
			},
			[]string{"cache_type"},
		),
		cacheSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_size",
				Help:      "Current cache size",
			},
			[]string{"cache_type"},
		),
		
		// HTTP client metrics
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests",
				Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "endpoint", "status_code"},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "Size of HTTP requests",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "Size of HTTP responses",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),
		httpRequestErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_request_errors_total",
				Help:      "Total number of HTTP request errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),
		
		// Data fetcher metrics
		dataFetchDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "data_fetch_duration_seconds",
				Help:      "Duration of data fetch operations",
				Buckets:   []float64{0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
			},
			[]string{"data_type", "source"},
		),
		dataFetchSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_success_total",
				Help:      "Total number of successful data fetches",
			},
			[]string{"data_type"},
		),
		dataFetchErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_errors_total",
				Help:      "Total number of data fetch errors",
			},
			[]string{"data_type", "error_type"},
		),
		dataFetchTimeouts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_timeouts_total",
				Help:      "Total number of data fetch timeouts",
			},
			[]string{"data_type"},
		),
		
		// System metrics
		memoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_usage_bytes",
				Help:      "Memory usage in bytes",
			},
			[]string{"type"},
		),
		goroutines: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "goroutines",
				Help:      "Number of goroutines",
			},
			[]string{},
		),
		uptime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "uptime_seconds",
				Help:      "Uptime in seconds",
			},
			[]string{},
		),
		startTime: time.Now(),
	}
}

// Describe implements prometheus.Collector
func (cm *CollectorMetrics) Describe(ch chan<- *prometheus.Desc) {
	cm.collectionDuration.Describe(ch)
	cm.collectionErrors.Describe(ch)
	cm.collectionSuccess.Describe(ch)
	cm.cacheHits.Describe(ch)
	cm.cacheMisses.Describe(ch)
	cm.cacheEvictions.Describe(ch)
	cm.cacheSize.Describe(ch)
	cm.httpRequestDuration.Describe(ch)
	cm.httpRequestSize.Describe(ch)
	cm.httpResponseSize.Describe(ch)
	cm.httpRequestErrors.Describe(ch)
	cm.dataFetchDuration.Describe(ch)
	cm.dataFetchSuccess.Describe(ch)
	cm.dataFetchErrors.Describe(ch)
	cm.dataFetchTimeouts.Describe(ch)
	cm.memoryUsage.Describe(ch)
	cm.goroutines.Describe(ch)
	cm.uptime.Describe(ch)
}

// Collect implements prometheus.Collector
func (cm *CollectorMetrics) Collect(ch chan<- prometheus.Metric) {
	cm.collectionDuration.Collect(ch)
	cm.collectionErrors.Collect(ch)
	cm.collectionSuccess.Collect(ch)
	cm.cacheHits.Collect(ch)
	cm.cacheMisses.Collect(ch)
	cm.cacheEvictions.Collect(ch)
	cm.cacheSize.Collect(ch)
	cm.httpRequestDuration.Collect(ch)
	cm.httpRequestSize.Collect(ch)
	cm.httpResponseSize.Collect(ch)
	cm.httpRequestErrors.Collect(ch)
	cm.dataFetchDuration.Collect(ch)
	cm.dataFetchSuccess.Collect(ch)
	cm.dataFetchErrors.Collect(ch)
	cm.dataFetchTimeouts.Collect(ch)
	cm.memoryUsage.Collect(ch)
	cm.goroutines.Collect(ch)
	cm.uptime.Collect(ch)
}

// RecordCollectionDuration records the duration of a collection operation
func (cm *CollectorMetrics) RecordCollectionDuration(operation string, duration time.Duration) {
	cm.collectionDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordCollectionError records a collection error
func (cm *CollectorMetrics) RecordCollectionError(operation, errorType string) {
	cm.collectionErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordCollectionSuccess records a successful collection
func (cm *CollectorMetrics) RecordCollectionSuccess(operation string) {
	cm.collectionSuccess.WithLabelValues(operation).Inc()
}

// RecordCacheHit records a cache hit
func (cm *CollectorMetrics) RecordCacheHit(cacheType string) {
	cm.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (cm *CollectorMetrics) RecordCacheMiss(cacheType string) {
	cm.cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordCacheEviction records a cache eviction
func (cm *CollectorMetrics) RecordCacheEviction(cacheType string) {
	cm.cacheEvictions.WithLabelValues(cacheType).Inc()
}

// SetCacheSize sets the current cache size
func (cm *CollectorMetrics) SetCacheSize(cacheType string, size float64) {
	cm.cacheSize.WithLabelValues(cacheType).Set(size)
}

// RecordHTTPRequestDuration records HTTP request duration
func (cm *CollectorMetrics) RecordHTTPRequestDuration(method, endpoint, statusCode string, duration time.Duration) {
	cm.httpRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration.Seconds())
}

// RecordHTTPRequestSize records HTTP request size
func (cm *CollectorMetrics) RecordHTTPRequestSize(method, endpoint string, size int64) {
	cm.httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordHTTPResponseSize records HTTP response size
func (cm *CollectorMetrics) RecordHTTPResponseSize(method, endpoint string, size int64) {
	cm.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordHTTPRequestError records HTTP request error
func (cm *CollectorMetrics) RecordHTTPRequestError(method, endpoint, errorType string) {
	cm.httpRequestErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordDataFetchDuration records data fetch duration
func (cm *CollectorMetrics) RecordDataFetchDuration(dataType, source string, duration time.Duration) {
	cm.dataFetchDuration.WithLabelValues(dataType, source).Observe(duration.Seconds())
}

// RecordDataFetchSuccess records successful data fetch
func (cm *CollectorMetrics) RecordDataFetchSuccess(dataType string) {
	cm.dataFetchSuccess.WithLabelValues(dataType).Inc()
}

// RecordDataFetchError records data fetch error
func (cm *CollectorMetrics) RecordDataFetchError(dataType, errorType string) {
	cm.dataFetchErrors.WithLabelValues(dataType, errorType).Inc()
}

// RecordDataFetchTimeout records data fetch timeout
func (cm *CollectorMetrics) RecordDataFetchTimeout(dataType string) {
	cm.dataFetchTimeouts.WithLabelValues(dataType).Inc()
}

// UpdateSystemMetrics updates system metrics
func (cm *CollectorMetrics) UpdateSystemMetrics() {
	// Update uptime
	uptime := time.Since(cm.startTime).Seconds()
	cm.uptime.WithLabelValues().Set(uptime)
	
	// Memory usage will be updated by the caller
}

// SetMemoryUsage sets memory usage metrics
func (cm *CollectorMetrics) SetMemoryUsage(memType string, bytes float64) {
	cm.memoryUsage.WithLabelValues(memType).Set(bytes)
}

// SetGoroutines sets the number of goroutines
func (cm *CollectorMetrics) SetGoroutines(count float64) {
	cm.goroutines.WithLabelValues().Set(count)
}

// RecordCollectionStart records the start of a collection operation
func (cm *CollectorMetrics) RecordCollectionStart() {
	// This method can be extended to track collection start time
	// For now, it's a placeholder for future timing enhancements
}