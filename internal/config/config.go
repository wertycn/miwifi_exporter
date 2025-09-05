package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

type Config struct {
	Router    RouterConfig `json:"router" envPrefix:"ROUTER_"`
	Server    ServerConfig `json:"server" envPrefix:"SERVER_"`
	Cache     CacheConfig  `json:"cache" envPrefix:"CACHE_"`
	Logging   LoggingConfig `json:"logging" envPrefix:"LOGGING_"`
	Memory    MemoryConfig `json:"memory" envPrefix:"MEMORY_"`
}

type RouterConfig struct {
	IP       string `json:"ip" env:"IP" validate:"required,ip"`
	Password string `json:"password" env:"PASSWORD" validate:"required,min=1"`
	Host     string `json:"host" env:"HOST" default:"miwifi"`
	Timeout  int    `json:"timeout" env:"TIMEOUT" default:"30" validate:"min=1"`
}

type ServerConfig struct {
	Port         int           `json:"port" env:"PORT" default:"9001" validate:"min=1,max=65535"`
	MetricsPath  string        `json:"metrics_path" env:"METRICS_PATH" default:"/metrics"`
	Namespace    string        `json:"namespace" env:"NAMESPACE" default:"miwifi"`
	ReadTimeout  time.Duration `json:"read_timeout" env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout time.Duration `json:"write_timeout" env:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout  time.Duration `json:"idle_timeout" env:"IDLE_TIMEOUT" default:"60s"`
}

type CacheConfig struct {
	Enabled bool          `json:"enabled" env:"ENABLED" default:"true"`
	TTL     time.Duration `json:"ttl" env:"TTL" default:"60s"`
}

type LoggingConfig struct {
	Level  string `json:"level" env:"LEVEL" default:"info"`
	Format string `json:"format" env:"FORMAT" default:"json" validate:"oneof=json text"`
}

type MemoryConfig struct {
	Enabled           bool `json:"enabled" env:"ENABLED" default:"true"`
	OptimizeOnCollect bool `json:"optimize_on_collect" env:"OPTIMIZE_ON_COLLECT" default:"true"`
	ForceGCOnClose    bool `json:"force_gc_on_close" env:"FORCE_GC_ON_CLOSE" default:"true"`
	TrackAllocations  bool `json:"track_allocations" env:"TRACK_ALLOCATIONS" default:"true"`
	EnablePoolStats   bool `json:"enable_pool_stats" env:"ENABLE_POOL_STATS" default:"true"`
}

var (
	defaultConfig = Config{
		Router: RouterConfig{
			Host:    "miwifi",
			Timeout: 30,
		},
		Server: ServerConfig{
			Port:         9001,
			MetricsPath:  "/metrics",
			Namespace:    "miwifi",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     10 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Memory: MemoryConfig{
			Enabled:           true,
			OptimizeOnCollect: true,
			ForceGCOnClose:    true,
			TrackAllocations:  true,
			EnablePoolStats:   true,
		},
	}
	validate = validator.New()
)

func Load() (*Config, error) {
	cfg := defaultConfig

	// 首先尝试从环境变量加载
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	// 如果环境变量没有提供必要配置，尝试从配置文件加载
	if cfg.Router.IP == "" || cfg.Router.Password == "" {
		if err := loadFromFile(&cfg); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// 验证配置
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func loadFromFile(cfg *Config) error {
	configFile := "config.json"
	if envFile := os.Getenv("CONFIG_FILE"); envFile != "" {
		configFile = envFile
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("config file %s not found", configFile)
	}

	// 从配置文件加载
	file, err := os.Open(configFile)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// 这里简化处理，实际应该使用JSON解析器
	// 为了向后兼容，我们保持原有的简单逻辑
	ip := getFromFile(configFile, "ip")
	password := getFromFile(configFile, "password")
	port := getFromFile(configFile, "port")

	if ip != "" {
		cfg.Router.IP = ip
	}
	if password != "" {
		cfg.Router.Password = password
	}
	if port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	return nil
}

func getFromFile(filename, key string) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}

	// 简单的字符串查找，实际应该使用JSON解析
	strContent := string(content)
	keyPattern := `"` + key + `"`
	startIndex := strings.Index(strContent, keyPattern)
	if startIndex == -1 {
		return ""
	}

	// 查找值的位置
	valueStart := strings.Index(strContent[startIndex:], ":")
	if valueStart == -1 {
		return ""
	}

	valueStart += startIndex + 1
	valueEnd := strings.Index(strContent[valueStart:], ",")
	if valueEnd == -1 {
		valueEnd = strings.Index(strContent[valueStart:], "}")
		if valueEnd == -1 {
			valueEnd = len(strContent) - valueStart
		}
	}

	value := strings.TrimSpace(strContent[valueStart : valueStart+valueEnd])
	value = strings.Trim(value, `"`)
	return value
}

func (c *Config) Validate() error {
	return validate.Struct(c)
}

func (c *Config) GetRouterURL() string {
	return fmt.Sprintf("http://%s", c.Router.IP)
}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}