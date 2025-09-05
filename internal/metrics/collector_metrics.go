package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CollectorMetrics 跟踪指标收集器的性能
type CollectorMetrics struct {
	// 收集指标
	collectionDuration *prometheus.HistogramVec
	collectionErrors   *prometheus.CounterVec
	collectionSuccess  *prometheus.CounterVec
	
	// 缓存指标
	cacheHits         *prometheus.CounterVec
	cacheMisses       *prometheus.CounterVec
	cacheEvictions    *prometheus.CounterVec
	cacheSize         *prometheus.GaugeVec
	
	// HTTP客户端指标
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec
	httpRequestErrors   *prometheus.CounterVec
	
	// 数据获取指标
	dataFetchDuration   *prometheus.HistogramVec
	dataFetchSuccess    *prometheus.CounterVec
	dataFetchErrors     *prometheus.CounterVec
	dataFetchTimeouts   *prometheus.CounterVec
	
	// 系统指标
	memoryUsage     *prometheus.GaugeVec
	goroutines      *prometheus.GaugeVec
	uptime          *prometheus.GaugeVec
	startTime       time.Time
}

// NewCollectorMetrics 创建新的收集器指标
func NewCollectorMetrics(namespace string) *CollectorMetrics {
	return &CollectorMetrics{
		// 收集指标
		collectionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "collection_duration_seconds",
				Help:      "指标收集持续时间",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		collectionErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "collection_errors_total",
				Help:      "收集错误总数",
			},
			[]string{"operation", "error_type"},
		),
		collectionSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "collection_success_total",
				Help:      "成功收集总数",
			},
			[]string{"operation"},
		),
		
		// 缓存指标
		cacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "缓存命中总数",
			},
			[]string{"cache_type"},
		),
		cacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "缓存未命中总数",
			},
			[]string{"cache_type"},
		),
		cacheEvictions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_evictions_total",
				Help:      "缓存淘汰总数",
			},
			[]string{"cache_type"},
		),
		cacheSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_size",
				Help:      "当前缓存大小",
			},
			[]string{"cache_type"},
		),
		
		// HTTP客户端指标
		httpRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP请求持续时间",
				Buckets:   []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"method", "endpoint", "status_code"},
		),
		httpRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP请求大小",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),
		httpResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP响应大小",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "endpoint"},
		),
		httpRequestErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_request_errors_total",
				Help:      "HTTP请求错误总数",
			},
			[]string{"method", "endpoint", "error_type"},
		),
		
		// 数据获取指标
		dataFetchDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "data_fetch_duration_seconds",
				Help:      "数据获取操作持续时间",
				Buckets:   []float64{0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
			},
			[]string{"data_type", "source"},
		),
		dataFetchSuccess: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_success_total",
				Help:      "成功数据获取总数",
			},
			[]string{"data_type"},
		),
		dataFetchErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_errors_total",
				Help:      "数据获取错误总数",
			},
			[]string{"data_type", "error_type"},
		),
		dataFetchTimeouts: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "data_fetch_timeouts_total",
				Help:      "数据获取超时总数",
			},
			[]string{"data_type"},
		),
		
		// 系统指标
		memoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_usage_bytes",
				Help:      "内存使用量(字节)",
			},
			[]string{"type"},
		),
		goroutines: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "goroutines",
				Help:      "Goroutine数量",
			},
			[]string{},
		),
		uptime: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "uptime_seconds",
				Help:      "运行时间(秒)",
			},
			[]string{},
		),
		startTime: time.Now(),
	}
}

// Describe 实现 prometheus.Collector 接口
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

// Collect 实现 prometheus.Collector 接口
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

// RecordCollectionDuration 记录收集操作的持续时间
func (cm *CollectorMetrics) RecordCollectionDuration(operation string, duration time.Duration) {
	cm.collectionDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordCollectionError 记录收集错误
func (cm *CollectorMetrics) RecordCollectionError(operation, errorType string) {
	cm.collectionErrors.WithLabelValues(operation, errorType).Inc()
}

// RecordCollectionSuccess 记录成功的收集
func (cm *CollectorMetrics) RecordCollectionSuccess(operation string) {
	cm.collectionSuccess.WithLabelValues(operation).Inc()
}

// RecordCacheHit 记录缓存命中
func (cm *CollectorMetrics) RecordCacheHit(cacheType string) {
	cm.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (cm *CollectorMetrics) RecordCacheMiss(cacheType string) {
	cm.cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordCacheEviction 记录缓存淘汰
func (cm *CollectorMetrics) RecordCacheEviction(cacheType string) {
	cm.cacheEvictions.WithLabelValues(cacheType).Inc()
}

// SetCacheSize 设置当前缓存大小
func (cm *CollectorMetrics) SetCacheSize(cacheType string, size float64) {
	cm.cacheSize.WithLabelValues(cacheType).Set(size)
}

// RecordHTTPRequestDuration 记录HTTP请求持续时间
func (cm *CollectorMetrics) RecordHTTPRequestDuration(method, endpoint, statusCode string, duration time.Duration) {
	cm.httpRequestDuration.WithLabelValues(method, endpoint, statusCode).Observe(duration.Seconds())
}

// RecordHTTPRequestSize 记录HTTP请求大小
func (cm *CollectorMetrics) RecordHTTPRequestSize(method, endpoint string, size int64) {
	cm.httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordHTTPResponseSize 记录HTTP响应大小
func (cm *CollectorMetrics) RecordHTTPResponseSize(method, endpoint string, size int64) {
	cm.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// RecordHTTPRequestError 记录HTTP请求错误
func (cm *CollectorMetrics) RecordHTTPRequestError(method, endpoint, errorType string) {
	cm.httpRequestErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordDataFetchDuration 记录数据获取持续时间
func (cm *CollectorMetrics) RecordDataFetchDuration(dataType, source string, duration time.Duration) {
	cm.dataFetchDuration.WithLabelValues(dataType, source).Observe(duration.Seconds())
}

// RecordDataFetchSuccess 记录成功的数据获取
func (cm *CollectorMetrics) RecordDataFetchSuccess(dataType string) {
	cm.dataFetchSuccess.WithLabelValues(dataType).Inc()
}

// RecordDataFetchError 记录数据获取错误
func (cm *CollectorMetrics) RecordDataFetchError(dataType, errorType string) {
	cm.dataFetchErrors.WithLabelValues(dataType, errorType).Inc()
}

// RecordDataFetchTimeout 记录数据获取超时
func (cm *CollectorMetrics) RecordDataFetchTimeout(dataType string) {
	cm.dataFetchTimeouts.WithLabelValues(dataType).Inc()
}

// UpdateSystemMetrics 更新系统指标
func (cm *CollectorMetrics) UpdateSystemMetrics() {
	// 更新运行时间
	uptime := time.Since(cm.startTime).Seconds()
	cm.uptime.WithLabelValues().Set(uptime)
	
	// 内存使用量将由调用者更新
}

// SetMemoryUsage 设置内存使用量指标
func (cm *CollectorMetrics) SetMemoryUsage(memType string, bytes float64) {
	cm.memoryUsage.WithLabelValues(memType).Set(bytes)
}

// SetGoroutines 设置Goroutine数量
func (cm *CollectorMetrics) SetGoroutines(count float64) {
	cm.goroutines.WithLabelValues().Set(count)
}

// RecordCollectionStart 记录收集操作的开始
func (cm *CollectorMetrics) RecordCollectionStart() {
	// 此方法可以扩展以跟踪收集开始时间
	// 目前是未来时间增强功能的占位符
}