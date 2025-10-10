package integrationtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const nullStr = "NULL"

// ClickHouseJSONResponse represents the structure of ClickHouse JSON FORMAT output.
type ClickHouseJSONResponse struct {
	Data []map[string]any `json:"data"`
}

// SeedData holds the loaded seed data for all tables.
// Key is table name, value is slice of row data.
var SeedData = make(map[string][]map[string]any)

// LoadSeedDataFromJSON loads all JSON seed files from the specified directory.
// It populates the global SeedData map with table_name → rows mappings.
// The filename (without .json extension) is used as the table name.
func LoadSeedDataFromJSON(testdataDir string) error {
	// Find all JSON files in the testdata directory
	pattern := filepath.Join(testdataDir, "*.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob JSON files: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no JSON files found in %s", testdataDir)
	}

	// Parse each JSON file
	for _, filePath := range matches {
		tableName := strings.TrimSuffix(filepath.Base(filePath), ".json")

		rows, err := parseJSONFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse file %s: %w", filePath, err)
		}

		SeedData[tableName] = rows
	}

	return nil
}

// parseJSONFile reads and unmarshals a JSON file in ClickHouse JSON FORMAT.
func parseJSONFile(filePath string) ([]map[string]any, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON
	var response ClickHouseJSONResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no data rows found in file")
	}

	return response.Data, nil
}

// SeedTestData loads JSON files and inserts all seed data into ClickHouse.
// It loads seed data if not already loaded, then inserts rows for all tables.
func SeedTestData(
	ctx context.Context,
	conn driver.Conn,
	database string,
	testdataDir string,
) error {
	// Load seed data if not already loaded
	if len(SeedData) == 0 {
		if err := LoadSeedDataFromJSON(testdataDir); err != nil {
			return fmt.Errorf("failed to load seed data: %w", err)
		}
	}

	// Insert data for each table
	for tableName, rows := range SeedData {
		for _, row := range rows {
			query := buildInsertQuery(database, tableName, row)

			if err := conn.Exec(ctx, query); err != nil {
				return fmt.Errorf("failed to insert into %s.%s: %w", database, tableName, err)
			}
		}
	}

	return nil
}

// buildInsertQuery constructs an INSERT query from a row of data.
// It handles proper quoting and formatting of values.
func buildInsertQuery(database, table string, row map[string]any) string {
	// Build column names and values slices
	columns := make([]string, 0, len(row))
	values := make([]string, 0, len(row))

	for column, value := range row {
		columns = append(columns, column)
		values = append(values, formatValue(value))
	}

	// Construct INSERT query
	query := fmt.Sprintf(
		"INSERT INTO %s.%s (%s) VALUES (%s)",
		database,
		table,
		strings.Join(columns, ", "),
		strings.Join(values, ", "),
	)

	return query
}

// formatValue formats a value for use in SQL INSERT statement.
func formatValue(value any) string {
	if value == nil {
		return nullStr
	}

	switch v := value.(type) {
	case string:
		// Escape single quotes by doubling them
		escaped := strings.ReplaceAll(v, "'", "''")
		// Also escape backslashes
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")

		return fmt.Sprintf("'%s'", escaped)
	case bool:
		if v {
			return "1"
		}

		return "0"
	case float64, float32:
		return fmt.Sprintf("%v", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%v", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v)
	case []any:
		// Handle arrays - format as ClickHouse array syntax
		if len(v) == 0 {
			return "[]"
		}

		elements := make([]string, len(v))
		for i, elem := range v {
			elements[i] = formatValue(elem)
		}

		return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
	case map[string]any:
		// Handle maps/objects - convert to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nullStr
		}

		// For JSON strings, escape single quotes and backslashes
		escaped := strings.ReplaceAll(string(jsonBytes), "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "'", "''")

		return fmt.Sprintf("'%s'", escaped)
	default:
		// For other complex types, try JSON marshaling
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nullStr
		}

		// If it's a JSON array/object, don't quote it, just fix the quotes
		jsonStr := string(jsonBytes)
		if strings.HasPrefix(jsonStr, "[") || strings.HasPrefix(jsonStr, "{") {
			// Convert JSON double quotes to ClickHouse single quotes for arrays
			if strings.HasPrefix(jsonStr, "[") {
				// Parse and reformat as ClickHouse array
				var arr []any
				if err := json.Unmarshal(jsonBytes, &arr); err == nil {
					return formatValue(arr)
				}
			}
		}

		// Otherwise, treat as string
		escaped := strings.ReplaceAll(jsonStr, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "'", "''")

		return fmt.Sprintf("'%s'", escaped)
	}
}
