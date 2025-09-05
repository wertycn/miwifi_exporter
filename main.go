package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/helloworlde/miwifi-exporter/internal/client"
	"github.com/helloworlde/miwifi-exporter/internal/collector"
	"github.com/helloworlde/miwifi-exporter/internal/config"
	"github.com/helloworlde/miwifi-exporter/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		configFile  = flag.String("config", "", "Path to configuration file")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("miwifi-exporter %s\n", version)
		fmt.Printf("commit: %s\n", commit)
		fmt.Printf("built: %s\n", date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadConfiguration(*configFile)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.Logging.Level, cfg.Logging.Format)
	logger.Default.Info("Starting miwifi-exporter")
	logger.Default.Infof("Configuration loaded - Router: %s, Server Port: %d", cfg.Router.IP, cfg.Server.Port)

	// Create router client
	routerClient := client.NewMiWiFiClient(cfg)

	// Create metrics collector
	metricsCollector := collector.NewMetricsCollector(cfg)
	metricsCollector.SetClient(routerClient)

	// Setup HTTP server
	server := setupHTTPServer(cfg, metricsCollector.GetRegistry())

	// Start server
	startServer(server, cfg, routerClient, metricsCollector)
}

func loadConfiguration(configFile string) (*config.Config, error) {
	if configFile != "" {
		os.Setenv("CONFIG_FILE", configFile)
	}
	
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	
	return cfg, nil
}

func setupHTTPServer(cfg *config.Config, registry *prometheus.Registry) *http.Server {
	mux := http.NewServeMux()
	
	// Metrics endpoint
	mux.Handle(cfg.Server.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>MiWiFi Exporter</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .header { text-align: center; margin-bottom: 30px; }
        .metrics { background: #f5f5f5; padding: 20px; border-radius: 5px; }
        .metric-link { display: inline-block; margin: 10px; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 3px; }
        .metric-link:hover { background: #0056b3; }
        .footer { text-align: center; margin-top: 30px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>MiWiFi Exporter</h1>
            <p>Prometheus exporter for Xiaomi WiFi routers</p>
        </div>
        
        <div class="metrics">
            <h2>Available Endpoints</h2>
            <a href="` + cfg.Server.MetricsPath + `" class="metric-link">Metrics</a>
            <a href="/health" class="metric-link">Health Check</a>
        </div>
        
        <div class="footer">
            <p>Version: ` + version + ` | Commit: ` + commit + `</p>
        </div>
    </div>
</body>
</html>`))
	})
	
	return &http.Server{
		Addr:         cfg.GetServerAddress(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

func startServer(server *http.Server, cfg *config.Config, routerClient client.RouterClient, metricsCollector *collector.MetricsCollector) {
	// Setup graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	
	// Start server in goroutine
	go func() {
		logger.Default.Infof("Starting server on %s", server.Addr)
		logger.Default.Infof("Metrics available at http://localhost:%d%s", cfg.Server.Port, cfg.Server.MetricsPath)
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Default.Fatalf("Failed to start server: %v", err)
		}
	}()
	
	// Test initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	logger.Default.Info("Testing router connection...")
	if err := routerClient.Authenticate(ctx); err != nil {
		logger.Default.Errorf("Failed to authenticate with router: %v", err)
		logger.Default.Warn("Please check your router IP and password in configuration")
	}
	
	// Wait for shutdown signal
	<-done
	logger.Default.Info("Shutting down server...")
	
	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Default.Errorf("Server shutdown error: %v", err)
	}
	
	// Cleanup resources
	if err := metricsCollector.Close(); err != nil {
		logger.Default.Errorf("Error closing metrics collector: %v", err)
	}
	
	logger.Default.Info("Server stopped")
}