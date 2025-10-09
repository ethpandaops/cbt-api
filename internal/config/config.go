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
	DSN      string `mapstructure:"dsn"`
	Database string `mapstructure:"database"`
	UseFinal bool   `mapstructure:"use_final"`

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
