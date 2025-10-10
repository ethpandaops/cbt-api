package integrationtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

		// Skip migrations that reference external tables (they don't exist in test env)
		shouldSkip, reason, err := shouldSkipMigration(ctx, conn, file)
		if err != nil {
			return fmt.Errorf(
				"failed to check if migration %s should be skipped: %w",
				filename,
				err,
			)
		}

		if shouldSkip {
			fmt.Printf(
				"  ⏭️  Skipping %s (reason: %s)\n",
				filename,
				reason,
			)

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

// shouldSkipMigration dynamically determines if a migration should be skipped by
// checking if it references external tables that don't exist in the test environment.
// It parses the SQL content looking for references to tables in other databases
// (e.g., default.table_name) and verifies if those tables exist.
// Returns: (shouldSkip bool, reason string, error).
func shouldSkipMigration(
	ctx context.Context,
	conn clickhouse.Conn,
	filePath string,
) (bool, string, error) {
	// Read the migration file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to read migration file: %w", err)
	}

	sql := string(content)

	// Extract referenced external tables (database.table pattern)
	externalTables := extractExternalTableReferences(sql)

	// If no external tables referenced, don't skip
	if len(externalTables) == 0 {
		return false, "", nil
	}

	// Check if all external tables exist
	missingTables := make([]string, 0)

	for _, table := range externalTables {
		exists, err := tableExists(ctx, conn, table.Database, table.Table)
		if err != nil {
			return false, "", fmt.Errorf(
				"failed to check if table %s.%s exists: %w",
				table.Database,
				table.Table,
				err,
			)
		}

		// Track missing tables
		if !exists {
			missingTables = append(
				missingTables,
				fmt.Sprintf("%s.%s", table.Database, table.Table),
			)
		}
	}

	// If any external tables are missing, skip this migration
	if len(missingTables) > 0 {
		reason := fmt.Sprintf(
			"references %d missing external table(s): %s",
			len(missingTables),
			strings.Join(missingTables, ", "),
		)

		return true, reason, nil
	}

	// All external tables exist, don't skip
	return false, "", nil
}

// ExternalTableRef represents a reference to a table in another database.
type ExternalTableRef struct {
	Database string
	Table    string
}

// extractExternalTableReferences parses SQL and extracts references to external tables.
// It looks for patterns like "database.table" or "`database`.`table`" in FROM/JOIN clauses.
func extractExternalTableReferences(sql string) []ExternalTableRef {
	// Pattern matches: database.table or `database`.`table`
	// Common patterns: FROM default.table, JOIN default.table, etc.
	pattern := `(?i)(?:FROM|JOIN)\s+` + // FROM or JOIN keyword
		`(?:\x60?)(\w+)(?:\x60?)\.` + // database name (with optional backticks)
		`(?:\x60?)(\w+)(?:\x60?)` // table name (with optional backticks)

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(sql, -1)

	// Use map to deduplicate
	seen := make(map[string]bool)
	refs := make([]ExternalTableRef, 0)

	for _, match := range matches {
		if len(match) >= 3 {
			database := match[1]
			table := match[2]
			key := database + "." + table

			// Skip if we've already seen this reference
			if seen[key] {
				continue
			}

			seen[key] = true

			refs = append(refs, ExternalTableRef{
				Database: database,
				Table:    table,
			})
		}
	}

	return refs
}

// tableExists checks if a table exists in ClickHouse.
func tableExists(
	ctx context.Context,
	conn clickhouse.Conn,
	database string,
	table string,
) (bool, error) {
	query := `
		SELECT count()
		FROM system.tables
		WHERE database = ? AND name = ?
	`

	var count uint64
	if err := conn.QueryRow(ctx, query, database, table).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
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
