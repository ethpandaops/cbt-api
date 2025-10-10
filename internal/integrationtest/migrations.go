package integrationtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// MigrationConfig holds configuration for running migrations against a
// ClickHouse database.
type MigrationConfig struct {
	// MigrationsPath is the directory containing migration files
	MigrationsPath string
	// Database is the target database name used to replace ${NETWORK_NAME}
	// template in migration files
	Database string
}

// RunMigrations executes all migration files in the configured path against
// the provided ClickHouse connection. Migration files are executed in order
// by their numeric prefix (001_, 002_, etc). The ${NETWORK_NAME} template
// in migration files is replaced with the configured database name.
//
// Returns an error if any migration fails to execute.
func RunMigrations(
	ctx context.Context,
	conn clickhouse.Conn,
	config MigrationConfig,
) error {
	if config.MigrationsPath == "" {
		return fmt.Errorf("migrations path cannot be empty")
	}

	if config.Database == "" {
		return fmt.Errorf("database name cannot be empty")
	}

	files, err := findMigrationFiles(config.MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf(
			"no migration files found in %s",
			config.MigrationsPath,
		)
	}

	for _, file := range files {
		filename := filepath.Base(file)

		// Skip migrations that reference external tables (default.*)
		if shouldSkipMigration(file) {
			fmt.Printf("  ⏭️  Skipping %s (references external tables)\n", filename)

			continue
		}

		if err := executeMigration(ctx, conn, file, config.Database); err != nil {
			return fmt.Errorf(
				"failed to execute migration %s: %w",
				filename,
				err,
			)
		}
	}

	return nil
}

// findMigrationFiles discovers and sorts all *.up.sql migration files in the
// specified directory. Files are sorted by their numeric prefix to ensure
// migrations execute in the correct order (001_, 002_, etc).
func findMigrationFiles(migrationsPath string) ([]string, error) {
	pattern := filepath.Join(migrationsPath, "*.up.sql")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob migration files: %w", err)
	}

	// Sort by filename which includes the numeric prefix
	sort.Strings(matches)

	return matches, nil
}

// executeMigration reads a migration file, replaces the ${NETWORK_NAME}
// template with the database name, and executes the SQL against the
// ClickHouse connection. The migration is split into individual statements
// and executed separately, as ClickHouse does not allow multi-statement execution.
func executeMigration(
	ctx context.Context,
	conn clickhouse.Conn,
	filePath string,
	database string,
) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	sql := string(content)

	// Replace all instances of ${NETWORK_NAME} with the database name
	sql = strings.ReplaceAll(sql, "${NETWORK_NAME}", database)

	// Split into individual statements and execute each one
	statements := splitSQLStatements(sql)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}

	return nil
}

// shouldSkipMigration checks if a migration references external tables (default.*).
// Migrations that reference external tables cannot run in the test environment
// because those external tables don't exist.
func shouldSkipMigration(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		// If we can't read it, don't skip (let the error surface during execution)
		return false
	}

	sql := string(content)

	// Check if the migration references tables in the default database
	// These are external tables that don't exist in the test environment
	return strings.Contains(sql, "default.")
}

// splitSQLStatements splits a SQL file into individual statements by semicolons.
// This is a simple implementation that handles basic SQL files. It does not
// handle semicolons inside string literals or complex cases, but works for
// typical migration files.
func splitSQLStatements(sql string) []string {
	// Split by semicolons
	parts := strings.Split(sql, ";")

	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			statements = append(statements, part)
		}
	}

	return statements
}
