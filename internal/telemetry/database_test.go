package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTableName(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "simple select",
			query:    "SELECT * FROM users",
			expected: "users",
		},
		{
			name:     "select with database prefix",
			query:    "SELECT * FROM mydb.users",
			expected: "users",
		},
		{
			name:     "select with where clause",
			query:    "SELECT id, name FROM orders WHERE status = 'active'",
			expected: "orders",
		},
		{
			name:     "insert into",
			query:    "INSERT INTO events (id, data) VALUES (?, ?)",
			expected: "events",
		},
		{
			name:     "insert into with database",
			query:    "INSERT INTO analytics.events (id) VALUES (?)",
			expected: "events",
		},
		{
			name:     "lowercase from",
			query:    "select * from fct_blocks where slot > 100",
			expected: "fct_blocks",
		},
		{
			name:     "mixed case",
			query:    "SELECT * From FctAttestations WHERE epoch = 1",
			expected: "FctAttestations",
		},
		{
			name:     "with backticks",
			query:    "SELECT * FROM `my_table`",
			expected: "my_table",
		},
		{
			name:     "with double quotes",
			query:    `SELECT * FROM "my_table"`,
			expected: "my_table",
		},
		{
			name:     "complex query with joins",
			query:    "SELECT a.id FROM users a JOIN orders b ON a.id = b.user_id",
			expected: "users",
		},
		{
			name:     "subquery - extracts outer table",
			query:    "SELECT * FROM blocks WHERE id IN (SELECT block_id FROM transactions)",
			expected: "blocks",
		},
		{
			name:     "no from clause",
			query:    "SELECT 1",
			expected: "unknown",
		},
		{
			name:     "empty query",
			query:    "",
			expected: "unknown",
		},
		{
			name:     "create table",
			query:    "CREATE TABLE test (id Int32)",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTableName(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractSQLOperation(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "select",
			query:    "SELECT * FROM users",
			expected: "SELECT",
		},
		{
			name:     "insert",
			query:    "INSERT INTO users VALUES (1)",
			expected: "INSERT",
		},
		{
			name:     "update",
			query:    "UPDATE users SET name = 'test'",
			expected: "UPDATE",
		},
		{
			name:     "delete",
			query:    "DELETE FROM users WHERE id = 1",
			expected: "DELETE",
		},
		{
			name:     "lowercase",
			query:    "select * from users",
			expected: "SELECT",
		},
		{
			name:     "with leading whitespace",
			query:    "   SELECT * FROM users",
			expected: "SELECT",
		},
		{
			name:     "empty query",
			query:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSQLOperation(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}
