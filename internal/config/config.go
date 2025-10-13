package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	ClickHouse ClickHouseConfig `mapstructure:"clickhouse"`
	Proto      ProtoConfig      `mapstructure:"proto"`
	API        APIConfig        `mapstructure:"api"`
	Telemetry  TelemetryConfig  `mapstructure:"telemetry"`
}

// ProtoConfig holds Protocol Buffer generation configuration.
type ProtoConfig struct {
	OutputDir       string `mapstructure:"output_dir"`
	Package         string `mapstructure:"package"`
	GoPackage       string `mapstructure:"go_package"`
	IncludeComments bool   `mapstructure:"include_comments"`
}

// APIConfig holds API exposure configuration.
type APIConfig struct {
	BasePath       string   `mapstructure:"base_path"`
	ExposePrefixes []string `mapstructure:"expose_prefixes"`
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`

	// HTTP server timeouts
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
}

// ClickHouseConfig holds ClickHouse database configuration.
type ClickHouseConfig struct {
	DSN       string               `mapstructure:"dsn"`
	Database  string               `mapstructure:"database"`
	UseFinal  bool                 `mapstructure:"use_final"`
	Discovery TableDiscoveryConfig `mapstructure:"discovery"`

	// Connection pool settings
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`

	// Timeouts
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	MaxExecutionTime int           `mapstructure:"max_execution_time"`

	// TLS settings
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// TableDiscoveryConfig holds table discovery configuration for proto generation.
type TableDiscoveryConfig struct {
	Prefixes []string `mapstructure:"prefixes"`
	Exclude  []string `mapstructure:"exclude"`
}

// TelemetryConfig holds OpenTelemetry configuration.
type TelemetryConfig struct {
	Enabled bool `mapstructure:"enabled"`

	// OTLP exporter settings
	Endpoint string            `mapstructure:"endpoint"` // e.g., "tempo.example.com:443"
	Insecure bool              `mapstructure:"insecure"` // Use insecure connection (dev only)
	Headers  map[string]string `mapstructure:"headers"`  // Auth headers (e.g., Authorization: Bearer <token>)

	// Service identification
	ServiceName    string `mapstructure:"service_name"`    // e.g., "xatu-cbt-api"
	ServiceVersion string `mapstructure:"service_version"` // e.g., "1.0.0"
	Environment    string `mapstructure:"environment"`     // e.g., "mainnet", "sepolia"

	// Sampling configuration
	SampleRate         float64 `mapstructure:"sample_rate"`          // 0.0 to 1.0 (e.g., 0.1 = 10%)
	AlwaysSampleErrors bool    `mapstructure:"always_sample_errors"` // Always sample requests with status >= 400

	// Export settings
	ExportTimeout   time.Duration `mapstructure:"export_timeout"`    // OTLP export timeout
	ExportBatchSize int           `mapstructure:"export_batch_size"` // Batch size for span processor
}

// Load loads configuration from file and environment variables.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("XATU_CBT_API")
	viper.AutomaticEnv()

	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_header_timeout", 10*time.Second)
	viper.SetDefault("server.read_timeout", 30*time.Second)
	viper.SetDefault("server.write_timeout", 30*time.Second)
	viper.SetDefault("server.idle_timeout", 120*time.Second)

	// ClickHouse defaults
	viper.SetDefault("clickhouse.max_open_conns", 10)
	viper.SetDefault("clickhouse.max_idle_conns", 5)
	viper.SetDefault("clickhouse.conn_max_lifetime", 60*time.Second)
	viper.SetDefault("clickhouse.dial_timeout", 10*time.Second)
	viper.SetDefault("clickhouse.read_timeout", 30*time.Second)
	viper.SetDefault("clickhouse.write_timeout", 30*time.Second)
	viper.SetDefault("clickhouse.max_execution_time", 60)
	viper.SetDefault("clickhouse.use_final", false)
	viper.SetDefault("clickhouse.insecure_skip_verify", false)

	// Proto defaults
	viper.SetDefault("proto.output_dir", "./pkg/proto/clickhouse")
	viper.SetDefault("proto.package", "cbt.v1")
	viper.SetDefault("proto.go_package", "github.com/ethpandaops/xatu-cbt-api/pkg/proto/clickhouse")
	viper.SetDefault("proto.include_comments", true)

	// API defaults
	viper.SetDefault("api.base_path", "/api/v1")
	viper.SetDefault("api.expose_prefixes", []string{"fct"})

	// Telemetry defaults
	viper.SetDefault("telemetry.enabled", false)
	viper.SetDefault("telemetry.service_name", "xatu-cbt-api")
	viper.SetDefault("telemetry.service_version", "unknown")
	viper.SetDefault("telemetry.environment", "unknown")
	viper.SetDefault("telemetry.sample_rate", 0.1)
	viper.SetDefault("telemetry.always_sample_errors", true)
	viper.SetDefault("telemetry.export_timeout", 10*time.Second)
	viper.SetDefault("telemetry.export_batch_size", 512)
	viper.SetDefault("telemetry.insecure", false)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
