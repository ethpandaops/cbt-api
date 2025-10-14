package database

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/cbt-api/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

var (
	sharedContainer     *clickhouse.ClickHouseContainer
	sharedContainerHost string
	sharedNativePort    string
	sharedHTTPPort      string
)

// TestMain sets up a shared ClickHouse container for all tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start ClickHouse container with default settings
	container, err := clickhouse.Run(ctx, "clickhouse/clickhouse-server:24.3")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start ClickHouse container: %v\n", err)
		os.Exit(1)
	}

	sharedContainer = container

	sharedContainerHost, err = container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get container host: %v\n", err)
		os.Exit(1)
	}

	// Get native port (9000)
	nativePort, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get native port: %v\n", err)
		os.Exit(1)
	}

	sharedNativePort = nativePort.Port()

	// Get HTTP port (8123)
	httpPort, err := container.MappedPort(ctx, "8123/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get HTTP port: %v\n", err)
		os.Exit(1)
	}

	sharedHTTPPort = httpPort.Port()

	// Run tests
	exitCode := m.Run()

	// Cleanup
	if err := container.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}

	os.Exit(exitCode)
}

func TestNewClient(t *testing.T) {
	// Build DSN with credentials
	userInfo := fmt.Sprintf("%s:%s", sharedContainer.User, sharedContainer.Password)
	nativeDSN := fmt.Sprintf("clickhouse://%s@%s", userInfo, net.JoinHostPort(sharedContainerHost, sharedNativePort))
	httpDSN := fmt.Sprintf("http://%s@%s", userInfo, net.JoinHostPort(sharedContainerHost, sharedHTTPPort))

	tests := []struct {
		name        string
		dsn         string
		database    string
		expectError bool
	}{
		{
			name:        "native protocol with clickhouse:// scheme",
			dsn:         nativeDSN,
			database:    "default",
			expectError: false,
		},
		{
			name:        "http protocol with http:// scheme",
			dsn:         httpDSN,
			database:    "default",
			expectError: false,
		},
		{
			name:        "invalid DSN",
			dsn:         "://invalid-dsn",
			database:    "default",
			expectError: true,
		},
		{
			name:        "unreachable host",
			dsn:         "clickhouse://unreachable-host-that-does-not-exist:9000",
			database:    "default",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ClickHouseConfig{
				DSN:              tt.dsn,
				Database:         tt.database,
				MaxExecutionTime: 60,
				DialTimeout:      5 * time.Second,
				ReadTimeout:      10 * time.Second,
			}

			log := logrus.New()
			log.SetLevel(logrus.ErrorLevel) // Quiet during tests

			client, err := NewClient(cfg, log)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, client)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)

			defer func() {
				assert.NoError(t, client.Close())
			}()

			// Verify connection is alive
			ctx := context.Background()
			err = client.Exec(ctx, "SELECT 1")
			assert.NoError(t, err)
		})
	}
}

func TestClient_Query(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create test table
	err := client.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_query (
			id UInt32,
			name String
		) ENGINE = Memory
	`)
	require.NoError(t, err)

	// Insert test data
	err = client.Exec(ctx, "INSERT INTO test_query VALUES (1, 'Alice'), (2, 'Bob')")
	require.NoError(t, err)

	// Query data
	rows, err := client.Query(ctx, "SELECT id, name FROM test_query ORDER BY id")
	require.NoError(t, err)

	defer rows.Close()

	// Verify results
	var count int

	for rows.Next() {
		var id uint32

		var name string

		scanErr := rows.Scan(&id, &name)
		require.NoError(t, scanErr)

		count++

		switch count {
		case 1:
			assert.Equal(t, uint32(1), id)
			assert.Equal(t, "Alice", name)
		case 2:
			assert.Equal(t, uint32(2), id)
			assert.Equal(t, "Bob", name)
		}
	}

	assert.Equal(t, 2, count)
	assert.NoError(t, rows.Err())

	// Cleanup
	err = client.Exec(ctx, "DROP TABLE IF EXISTS test_query")
	assert.NoError(t, err)
}

func TestClient_QueryRow(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Test single value query
	var result uint8

	row := client.QueryRow(ctx, "SELECT 42")

	err := row.Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, uint8(42), result)
}

func TestClient_Select(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Create test table
	err := client.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS test_select (
			id UInt32,
			name String,
			age UInt32
		) ENGINE = Memory
	`)
	require.NoError(t, err)

	// Insert test data
	err = client.Exec(ctx, `
		INSERT INTO test_select VALUES
		(1, 'Alice', 30),
		(2, 'Bob', 25),
		(3, 'Charlie', 35)
	`)
	require.NoError(t, err)

	// Define struct for results
	type Person struct {
		ID   uint32 `ch:"id"`
		Name string `ch:"name"`
		Age  uint32 `ch:"age"`
	}

	// Query and scan into struct slice
	var people []Person

	err = client.Select(ctx, &people, "SELECT id, name, age FROM test_select ORDER BY id")
	require.NoError(t, err)

	// Verify results
	require.Len(t, people, 3)
	assert.Equal(t, uint32(1), people[0].ID)
	assert.Equal(t, "Alice", people[0].Name)
	assert.Equal(t, uint32(30), people[0].Age)

	// Cleanup
	err = client.Exec(ctx, "DROP TABLE IF EXISTS test_select")
	assert.NoError(t, err)
}

func TestClient_Exec(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		expectError bool
	}{
		{
			name:        "valid DDL",
			query:       "CREATE TABLE IF NOT EXISTS test_exec (id UInt32) ENGINE = Memory",
			expectError: false,
		},
		{
			name:        "valid DML",
			query:       "INSERT INTO test_exec VALUES (1), (2), (3)",
			expectError: false,
		},
		{
			name:        "invalid SQL syntax",
			query:       "SELECT FROM WHERE",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Exec(ctx, tt.query)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	// Cleanup
	err := client.Exec(ctx, "DROP TABLE IF EXISTS test_exec")
	assert.NoError(t, err)
}

func TestClient_ContextCancellation(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Test query with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var result []struct{}

	err := client.Select(ctx, &result, "SELECT sleepEachRow(1) FROM numbers(100)")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestClient_QueryTimeout(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	// Test query with short timeout using a long-running query
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var result []uint64

	// Query that will take a long time (summing a billion numbers)
	err := client.Select(ctx, &result, "SELECT sum(number) FROM numbers(1000000000)")
	require.Error(t, err)

	// Accept either "deadline exceeded" or "i/o timeout" as both indicate timeout
	errMsg := err.Error()
	assert.True(t,
		strings.Contains(errMsg, "deadline exceeded") || strings.Contains(errMsg, "i/o timeout"),
		"expected timeout error, got: %s", errMsg,
	)
}

func TestClient_ConcurrentQueries(t *testing.T) {
	client := createTestClient(t)
	defer client.Close()

	ctx := context.Background()

	// Test concurrent read queries
	const numWorkers = 10

	errChan := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			var result uint8

			row := client.QueryRow(ctx, "SELECT ?", workerID)

			err := row.Scan(&result)
			if err != nil {
				errChan <- err

				return
			}

			if result != uint8(workerID) {
				errChan <- fmt.Errorf("expected %d, got %d", workerID, result)

				return
			}

			errChan <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numWorkers; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

func TestParseDSN(t *testing.T) {
	tests := []struct {
		name           string
		dsn            string
		expectScheme   string
		expectError    bool
		expectContains string
	}{
		{
			name:         "native with port 9000 (auto-detected)",
			dsn:          "localhost:9000",
			expectScheme: "clickhouse",
			expectError:  false,
		},
		{
			name:         "native with port 9440 (auto-detected)",
			dsn:          "localhost:9440",
			expectScheme: "clickhouse",
			expectError:  false,
		},
		{
			name:         "explicit clickhouse:// scheme",
			dsn:          "clickhouse://localhost:9000",
			expectScheme: "clickhouse",
			expectError:  false,
		},
		{
			name:         "explicit https:// scheme",
			dsn:          "https://localhost:8123",
			expectScheme: "https",
			expectError:  false,
		},
		{
			name:         "explicit http:// scheme",
			dsn:          "http://localhost:8123",
			expectScheme: "http",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseDSN(tt.dsn)

			if tt.expectError {
				require.Error(t, err)

				if tt.expectContains != "" {
					assert.Contains(t, err.Error(), tt.expectContains)
				}

				return
			}

			require.NoError(t, err)
			require.NotNil(t, parsed)
			assert.Equal(t, tt.expectScheme, parsed.Scheme)
		})
	}
}

func TestCreateClickHouseOptions(t *testing.T) {
	tests := []struct {
		name               string
		dsn                string
		database           string
		expectNativeProto  bool
		expectTLS          bool
		maxExecutionTime   int
		dialTimeout        time.Duration
		insecureSkipVerify bool
	}{
		{
			name:              "native protocol",
			dsn:               "clickhouse://localhost:9000",
			database:          "test_db",
			expectNativeProto: true,
			expectTLS:         false,
			maxExecutionTime:  60,
			dialTimeout:       10 * time.Second,
		},
		{
			name:              "native with TLS (port 9440)",
			dsn:               "clickhouse://localhost:9440",
			database:          "test_db",
			expectNativeProto: true,
			expectTLS:         true,
			maxExecutionTime:  30,
			dialTimeout:       5 * time.Second,
		},
		{
			name:              "http protocol",
			dsn:               "https://localhost:8123",
			database:          "test_db",
			expectNativeProto: false,
			expectTLS:         true,
			maxExecutionTime:  120,
			dialTimeout:       15 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, err := parseDSN(tt.dsn)
			require.NoError(t, err)

			cfg := &config.ClickHouseConfig{
				Database:           tt.database,
				MaxExecutionTime:   tt.maxExecutionTime,
				DialTimeout:        tt.dialTimeout,
				InsecureSkipVerify: tt.insecureSkipVerify,
			}

			opts := createClickHouseOptions(cfg, parsedURL)

			require.NotNil(t, opts)
			assert.Equal(t, tt.database, opts.Auth.Database)
			assert.Equal(t, tt.dialTimeout, opts.DialTimeout)

			// Check protocol
			if tt.expectNativeProto {
				assert.Equal(t, 0, int(opts.Protocol)) // Native = 0
			} else {
				assert.Equal(t, 1, int(opts.Protocol)) // HTTP = 1
			}

			// Check TLS
			if tt.expectTLS {
				assert.NotNil(t, opts.TLS)
				assert.Equal(t, tt.insecureSkipVerify, opts.TLS.InsecureSkipVerify)
			} else {
				assert.Nil(t, opts.TLS)
			}
		})
	}
}

// createTestClient creates a ClickHouse client connected to the shared test container.
func createTestClient(t *testing.T) *Client {
	t.Helper()

	userInfo := fmt.Sprintf("%s:%s", sharedContainer.User, sharedContainer.Password)
	dsn := fmt.Sprintf("clickhouse://%s@%s", userInfo, net.JoinHostPort(sharedContainerHost, sharedNativePort))

	cfg := &config.ClickHouseConfig{
		DSN:              dsn,
		Database:         "default",
		MaxExecutionTime: 60,
		DialTimeout:      5 * time.Second,
		ReadTimeout:      10 * time.Second,
	}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel) // Quiet during tests

	client, err := NewClient(cfg, log)
	require.NoError(t, err)
	require.NotNil(t, client)

	return client
}
