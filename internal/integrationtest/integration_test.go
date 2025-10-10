package integrationtest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu-cbt-api/internal/config"
	"github.com/ethpandaops/xatu-cbt-api/internal/server"
)

var (
	testContainer *ClickHouseContainer
	testServerURL string
	testLogger    *logrus.Logger
)

// TestMain sets up shared test infrastructure for integration tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Setup logger
	testLogger = logrus.New()
	testLogger.SetOutput(os.Stdout)
	testLogger.SetLevel(logrus.InfoLevel)

	testLogger.Info("Setting up integration test infrastructure...")

	// 1. Setup ClickHouse container
	testLogger.Info("Starting ClickHouse testcontainer...")

	container, err := SetupClickHouseContainer(ctx, "test_network")
	if err != nil {
		testLogger.Fatalf("Failed to setup container: %v", err)
	}

	testContainer = container

	defer func() {
		testLogger.Info("Cleaning up ClickHouse container...")

		if cleanupErr := testContainer.Cleanup(ctx); cleanupErr != nil {
			testLogger.Errorf("Failed to cleanup container: %v", cleanupErr)
		}
	}()

	testLogger.WithFields(logrus.Fields{
		"dsn":      container.DSN,
		"database": container.Database,
	}).Info("ClickHouse container started")

	// 2. Connect to ClickHouse
	testLogger.Info("Connecting to ClickHouse...")

	conn, err := connectToClickHouse(ctx, container.DSN, container.Database)
	if err != nil {
		testLogger.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	defer conn.Close()

	// 3. Run migrations
	testLogger.Info("Running xatu-cbt migrations...")

	migrationsPath := filepath.Join("..", "..", ".xatu-cbt", "migrations")
	migrationConfig := MigrationConfig{
		MigrationsPath: migrationsPath,
		Database:       container.Database,
	}

	if err := RunMigrations(ctx, conn, migrationConfig); err != nil {
		testLogger.Fatalf("Failed to run migrations: %v", err)
	}

	testLogger.Info("Migrations completed successfully")

	// 4. Seed test data
	testLogger.Info("Seeding test data from JSON files...")

	testdataDir := filepath.Join("testdata")
	if seedErr := SeedTestData(ctx, conn, container.Database, testdataDir); seedErr != nil {
		testLogger.Fatalf("Failed to seed test data: %v", seedErr)
	}

	testLogger.Info("Test data seeded successfully")

	// 5. Start xatu-cbt-api server
	testLogger.Info("Starting xatu-cbt-api server...")

	serverPort := 18080 // Different from default 8080 to avoid conflicts
	testServerURL = fmt.Sprintf("http://localhost:%d", serverPort)

	cfg := &config.Config{
		ClickHouse: config.ClickHouseConfig{
			DSN:              container.DSN,
			Database:         container.Database,
			UseFinal:         true,
			MaxOpenConns:     5,
			MaxIdleConns:     2,
			ConnMaxLifetime:  60 * time.Second,
			DialTimeout:      10 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     30 * time.Second,
			MaxExecutionTime: 60,
		},
		Server: config.ServerConfig{
			Port:              serverPort,
			Host:              "localhost",
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}

	// Create and start server
	srv, srvErr := server.New(cfg, testLogger)
	if srvErr != nil {
		testLogger.Fatalf("Failed to create server: %v", srvErr)
	}

	// Start server in goroutine
	go func() {
		testLogger.WithField("address", srv.Addr).Info("Server listening")

		if err := srv.ListenAndServe(); err != nil {
			testLogger.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	testLogger.Info("Waiting for server to be ready...")

	if waitErr := waitForServer(testServerURL+"/health", 30*time.Second); waitErr != nil {
		testLogger.Fatalf("Server failed to start: %v", waitErr)
	}

	testLogger.Info("Server is ready")

	// 6. Run tests
	testLogger.Info("Running integration tests...")

	exitCode := m.Run()

	// Cleanup server
	if shutdownErr := srv.Shutdown(context.Background()); shutdownErr != nil {
		testLogger.Errorf("Failed to shutdown server: %v", shutdownErr)
	}

	os.Exit(exitCode)
}

// TestAPIEndpoints tests all API endpoints discovered from openapi.yaml.
func TestAPIEndpoints(t *testing.T) {
	ctx := context.Background()

	// 1. Parse OpenAPI spec to discover endpoints
	specPath := filepath.Join("..", "..", "openapi.yaml")
	endpoints, err := ParseOpenAPISpec(specPath)
	require.NoError(t, err, "Failed to parse OpenAPI spec")
	require.NotEmpty(t, endpoints, "No endpoints found in OpenAPI spec")

	t.Logf("Discovered %d endpoints to test", len(endpoints))

	// 2. Test all endpoints
	tester := NewEndpointTester(testServerURL, SeedData)
	results := tester.TestAllEndpoints(ctx, endpoints)

	// 3. Report results
	reporter := NewReporter(os.Stdout)
	reporter.Report(results)

	// 4. Assert all tests passed
	var failures []TestResult

	for _, result := range results {
		if !result.Success {
			failures = append(failures, result)
		}
	}

	require.Empty(
		t,
		failures,
		"Some endpoints failed (see detailed output above)",
	)
}

// connectToClickHouse establishes a native ClickHouse connection.
func connectToClickHouse(
	ctx context.Context,
	dsn string,
	database string,
) (clickhouse.Conn, error) {
	// Parse DSN to extract host:port
	// DSN format: clickhouse://user:pass@host:port or just host:port
	addr := dsn

	// Strip protocol prefix if present
	if len(addr) > 12 && addr[:12] == "clickhouse://" {
		addr = addr[12:]
	}

	// Strip userinfo if present (user:pass@)
	if atIndex := findLastAt(addr); atIndex != -1 {
		addr = addr[atIndex+1:]
	}

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: "default",
			Password: "password", // Must match container setup
		},
		DialTimeout:      10 * time.Second,
		MaxOpenConns:     5,
		MaxIdleConns:     2,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Test connection
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()

		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return conn, nil
}

// findLastAt finds the last '@' in a string (for parsing user:pass@host).
func findLastAt(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			return i
		}
	}

	return -1
}

// waitForServer polls the server health endpoint until it responds or times out.
func waitForServer(healthURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server to start")

		case <-ticker.C:
			// Try to dial the server's port
			serverURL, err := parseHealthURLToAddress(healthURL)
			if err != nil {
				continue
			}

			conn, err := net.DialTimeout("tcp", serverURL, 500*time.Millisecond)
			if err == nil {
				_ = conn.Close()

				return nil
			}
		}
	}
}

// parseHealthURLToAddress extracts host:port from health URL.
func parseHealthURLToAddress(healthURL string) (string, error) {
	// healthURL is like "http://localhost:18080/health"
	// We need "localhost:18080"
	u, err := url.Parse(healthURL)
	if err != nil {
		return "", fmt.Errorf("invalid health URL: %w", err)
	}

	return u.Host, nil
}
