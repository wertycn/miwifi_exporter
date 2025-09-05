package http

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Config represents HTTP client configuration
type Config struct {
	MaxIdleConns        int           `json:"max_idle_conns" default:"100"`
	MaxIdleConnsPerHost int           `json:"max_idle_conns_per_host" default:"10"`
	IdleConnTimeout     time.Duration `json:"idle_conn_timeout" default:"90s"`
	Timeout             time.Duration `json:"timeout" default:"30s"`
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" default:"10s"`
	DisableKeepAlives   bool          `json:"disable_keep_alives" default:"false"`
	MaxConnsPerHost     int           `json:"max_conns_per_host" default:"100"`
	DisableCompression  bool          `json:"disable_compression" default:"false"`
}

// DefaultConfig returns default HTTP client configuration
func DefaultConfig() *Config {
	return &Config{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		Timeout:             30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		MaxConnsPerHost:     100,
		DisableCompression:  false,
	}
}

// NewOptimizedClient creates an optimized HTTP client with connection pooling
func NewOptimizedClient(cfg *Config) *http.Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   cfg.Timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		DisableKeepAlives:     cfg.DisableKeepAlives,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		DisableCompression:    cfg.DisableCompression,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
		ForceAttemptHTTP2:     true,
		MaxResponseHeaderBytes: 1 << 20, // 1MB
	}

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
		Jar:       nil, // Don't use cookies by default
	}
}

// NewMetricsClient creates an HTTP client with metrics collection
func NewMetricsClient(cfg *Config, metricsCollector MetricsCollector) *http.Client {
	client := NewOptimizedClient(cfg)
	
	if transport, ok := client.Transport.(*http.Transport); ok {
		// Wrap transport with metrics
		client.Transport = NewMetricsTransport(transport, metricsCollector)
	}
	
	return client
}

// MetricsCollector defines the interface for HTTP metrics collection
type MetricsCollector interface {
	RecordRequestDuration(method, url string, duration time.Duration, statusCode int)
	RecordRequestSize(method, url string, size int64)
	RecordResponseSize(method, url string, size int64)
}

// MetricsTransport wraps http.Transport to collect metrics
type MetricsTransport struct {
	transport       http.RoundTripper
	metricsCollector MetricsCollector
}

// NewMetricsTransport creates a new metrics transport
func NewMetricsTransport(transport http.RoundTripper, metricsCollector MetricsCollector) *MetricsTransport {
	return &MetricsTransport{
		transport:       transport,
		metricsCollector: metricsCollector,
	}
}

// RoundTrip implements http.RoundTripper
func (m *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	
	// Record request size
	if req.ContentLength > 0 {
		m.metricsCollector.RecordRequestSize(req.Method, req.URL.String(), req.ContentLength)
	}
	
	resp, err := m.transport.RoundTrip(req)
	duration := time.Since(start)
	
	if err == nil {
		// Record successful request
		m.metricsCollector.RecordRequestDuration(req.Method, req.URL.String(), duration, resp.StatusCode)
		
		// Record response size
		if resp.ContentLength > 0 {
			m.metricsCollector.RecordResponseSize(req.Method, req.URL.String(), resp.ContentLength)
		}
	} else {
		// Record failed request
		m.metricsCollector.RecordRequestDuration(req.Method, req.URL.String(), duration, 0)
	}
	
	return resp, err
}