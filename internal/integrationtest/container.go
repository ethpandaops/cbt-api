package integrationtest

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// ClickHouseContainer represents a ClickHouse test container with connection details.
type ClickHouseContainer struct {
	Container *clickhouse.ClickHouseContainer
	DSN       string
	Database  string
}

// SetupClickHouseContainer creates a ClickHouse container with embedded Keeper and cluster macros.
// This allows using cluster-aware migrations without requiring a multi-node cluster.
func SetupClickHouseContainer(ctx context.Context, database string) (*ClickHouseContainer, error) {
	username := "default"
	password := "password"

	// Create temporary config files
	configPath, err := createClickHouseConfigWithEmbeddedKeeper()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}
	defer os.Remove(configPath)

	// Start ClickHouse with embedded keeper and cluster config
	container, err := clickhouse.Run(
		ctx,
		"clickhouse/clickhouse-server:latest",
		clickhouse.WithDatabase(database),
		clickhouse.WithUsername(username),
		clickhouse.WithPassword(password),
		clickhouse.WithConfigFile(configPath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start ClickHouse container: %w", err)
	}

	// Get connection details
	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)

		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	nativePort, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		_ = container.Terminate(ctx)

		return nil, fmt.Errorf("failed to get native port: %w", err)
	}

	// Build DSN
	userInfo := fmt.Sprintf("%s:%s", username, password)
	dsn := fmt.Sprintf("clickhouse://%s@%s", userInfo, net.JoinHostPort(host, nativePort.Port()))

	return &ClickHouseContainer{
		Container: container,
		DSN:       dsn,
		Database:  database,
	}, nil
}

// Cleanup terminates the ClickHouse container.
func (c *ClickHouseContainer) Cleanup(ctx context.Context) error {
	if c.Container == nil {
		return nil
	}

	if err := c.Container.Terminate(ctx); err != nil {
		return fmt.Errorf("failed to terminate ClickHouse container: %w", err)
	}

	return nil
}

// createClickHouseConfigWithEmbeddedKeeper creates a ClickHouse config file with:
// - Embedded Keeper for single-node replication support
// - Cluster macros ({installation}, {cluster}, {shard}, {replica})
// - Single-node cluster definition.
func createClickHouseConfigWithEmbeddedKeeper() (string, error) {
	config := `
<clickhouse>
    <!-- Embedded Keeper configuration for single-node testing -->
    <keeper_server>
        <tcp_port>9181</tcp_port>
        <server_id>1</server_id>
        <log_storage_path>/var/lib/clickhouse/coordination/log</log_storage_path>
        <snapshot_storage_path>/var/lib/clickhouse/coordination/snapshots</snapshot_storage_path>
        <coordination_settings>
            <operation_timeout_ms>10000</operation_timeout_ms>
            <session_timeout_ms>30000</session_timeout_ms>
            <raft_logs_level>information</raft_logs_level>
        </coordination_settings>
        <raft_configuration>
            <server>
                <id>1</id>
                <hostname>localhost</hostname>
                <port>9234</port>
            </server>
        </raft_configuration>
    </keeper_server>

    <!-- Cluster macros for migration compatibility -->
    <macros>
        <installation>test</installation>
        <cluster>test_cluster</cluster>
        <shard>1</shard>
        <replica>replica1</replica>
    </macros>

    <!-- Single-node cluster definition -->
    <remote_servers>
        <test_cluster>
            <shard>
                <internal_replication>true</internal_replication>
                <replica>
                    <host>localhost</host>
                    <port>9000</port>
                </replica>
            </shard>
        </test_cluster>
    </remote_servers>

    <!-- Keeper client configuration -->
    <zookeeper>
        <node>
            <host>localhost</host>
            <port>9181</port>
        </node>
    </zookeeper>
</clickhouse>
`

	// Write config to temp file
	configPath := filepath.Join(os.TempDir(), "clickhouse_test_config.xml")

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil { //nolint:gosec // fine for test.
		return "", fmt.Errorf("failed to write config file: %w", err)
	}

	return configPath, nil
}
