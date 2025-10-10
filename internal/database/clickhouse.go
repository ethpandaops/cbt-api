package database

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/ethpandaops/xatu-cbt-api/internal/config"
	"github.com/sirupsen/logrus"
)

// ClickHouse protocol constants.
const (
	SchemeClickHouse        = "clickhouse://"
	SchemeHTTPS             = "https://"
	SchemeHTTP              = "http://"
	PortClickHouseNative    = ":9000"
	PortClickHouseNativeTLS = ":9440"
)

// DatabaseClient defines the interface for database operations.
// This interface allows for instrumentation wrappers (e.g., tracing) without modifying generated code.
type DatabaseClient interface {
	Query(ctx context.Context, query string, args ...any) (driver.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) driver.Row
	Select(ctx context.Context, dest any, query string, args ...any) error
	Exec(ctx context.Context, query string, args ...any) error
	Close() error
}

// Client wraps the official ClickHouse Go driver (native interface).
type Client struct {
	conn   driver.Conn
	log    logrus.FieldLogger
	config *config.ClickHouseConfig
}

// Ensure Client implements DatabaseClient interface
var _ DatabaseClient = (*Client)(nil)

// NewClient creates a new ClickHouse client using the official Go driver.
func NewClient(cfg *config.ClickHouseConfig, logger logrus.FieldLogger) (*Client, error) {
	log := logger.WithFields(logrus.Fields{
		"module": "clickhouse",
	})

	log.Debug("Initializing ClickHouse client")

	// Parse and process DSN
	parsedDSN, err := parseDSN(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Create ClickHouse options
	options := createClickHouseOptions(cfg, parsedDSN)

	// Open connection using native driver
	conn, err := clickhouse.Open(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse connection: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.WithFields(logrus.Fields{
		"database": cfg.Database,
		"host":     parsedDSN.Host,
	}).Info("ClickHouse client initialised")

	return &Client{
		conn:   conn,
		log:    log,
		config: cfg,
	}, nil
}

// Query executes a SQL query and returns rows (native driver interface).
func (c *Client) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.conn.Query(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func (c *Client) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return c.conn.QueryRow(ctx, query, args...)
}

// Select executes a query and scans results directly into a slice of structs.
func (c *Client) Select(ctx context.Context, dest any, query string, args ...any) error {
	return c.conn.Select(ctx, dest, query, args...)
}

// Exec executes a query without returning any rows.
func (c *Client) Exec(ctx context.Context, query string, args ...any) error {
	return c.conn.Exec(ctx, query, args...)
}

// Close closes the database connection.
func (c *Client) Close() error {
	c.log.Info("Closing ClickHouse connection")

	return c.conn.Close()
}

// parseDSN processes the DSN and determines connection protocol.
func parseDSN(dsn string) (*url.URL, error) {
	// Normalize scheme based on explicit scheme or port detection
	var processedDSN string

	// If DSN already has a scheme, keep it as-is
	if strings.HasPrefix(dsn, SchemeClickHouse) || strings.HasPrefix(dsn, SchemeHTTPS) || strings.HasPrefix(dsn, SchemeHTTP) {
		processedDSN = dsn
	} else {
		// Auto-detect protocol based on port
		useNative := strings.Contains(dsn, PortClickHouseNative) || strings.Contains(dsn, PortClickHouseNativeTLS)
		if useNative {
			// Use native protocol for ports 9000/9440
			processedDSN = SchemeClickHouse + dsn
		} else {
			// Default to HTTPS for other ports
			processedDSN = SchemeHTTPS + dsn
		}
	}

	// Parse URL
	parsedURL, err := url.Parse(processedDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	return parsedURL, nil
}

// createClickHouseOptions builds connection options for the driver.
func createClickHouseOptions(cfg *config.ClickHouseConfig, parsedURL *url.URL) *clickhouse.Options {
	// Extract auth
	auth := clickhouse.Auth{
		Database: cfg.Database,
		Username: parsedURL.User.Username(),
	}

	if password, ok := parsedURL.User.Password(); ok {
		auth.Password = password
	}

	// Determine protocol
	protocol := clickhouse.HTTP
	if strings.HasPrefix(parsedURL.Scheme, "clickhouse") {
		protocol = clickhouse.Native
	}

	// Build options
	options := &clickhouse.Options{
		Addr:     []string{parsedURL.Host},
		Auth:     auth,
		Protocol: protocol,
		Settings: clickhouse.Settings{
			"max_execution_time": cfg.MaxExecutionTime,
		},
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
	}

	// Add TLS if using HTTPS
	if strings.HasPrefix(parsedURL.Scheme, "https") || parsedURL.Port() == "9440" {
		options.TLS = &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // config for dev environments
		}
	}

	return options
}
