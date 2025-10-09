package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddChTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple json tag without omitempty",
			input:    "`json:\"field_name\"`",
			expected: "`json:\"field_name\" ch:\"field_name\"`",
		},
		{
			name:     "json tag with omitempty",
			input:    "`json:\"field_name,omitempty\"`",
			expected: "`json:\"field_name,omitempty\" ch:\"field_name\"`",
		},
		{
			name: "multiple fields in struct",
			input: `type MyStruct struct {
	ID     string ` + "`json:\"id\"`" + `
	Name   string ` + "`json:\"name,omitempty\"`" + `
	Status string ` + "`json:\"status\"`" + `
}`,
			expected: `type MyStruct struct {
	ID     string ` + "`json:\"id\" ch:\"id\"`" + `
	Name   string ` + "`json:\"name,omitempty\" ch:\"name\"`" + `
	Status string ` + "`json:\"status\" ch:\"status\"`" + `
}`,
		},
		{
			name: "nested struct",
			input: `type Parent struct {
	Field1 string ` + "`json:\"field1,omitempty\"`" + `
	Child  struct {
		Field2 string ` + "`json:\"field2\"`" + `
	} ` + "`json:\"child\"`" + `
}`,
			expected: `type Parent struct {
	Field1 string ` + "`json:\"field1,omitempty\" ch:\"field1\"`" + `
	Child  struct {
		Field2 string ` + "`json:\"field2\" ch:\"field2\"`" + `
	} ` + "`json:\"child\" ch:\"child\"`" + `
}`,
		},
		{
			name:     "field with underscore",
			input:    "`json:\"slot_start_date_time\"`",
			expected: "`json:\"slot_start_date_time\" ch:\"slot_start_date_time\"`",
		},
		{
			name:     "field with numbers",
			input:    "`json:\"block_number123\"`",
			expected: "`json:\"block_number123\" ch:\"block_number123\"`",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "no json tags",
			input:    "type MyStruct struct {\n\tField string\n}",
			expected: "type MyStruct struct {\n\tField string\n}",
		},
		{
			name: "mixed tags - some already have ch tags",
			input: `type MyStruct struct {
	Field1 string ` + "`json:\"field1\"`" + `
	Field2 string ` + "`json:\"field2\" ch:\"field2\"`" + `
}`,
			expected: `type MyStruct struct {
	Field1 string ` + "`json:\"field1\" ch:\"field1\"`" + `
	Field2 string ` + "`json:\"field2\" ch:\"field2\"`" + `
}`,
		},
		{
			name: "complex struct with multiple tag types",
			input: `type Response struct {
	Slot          uint64  ` + "`json:\"slot\"`" + `
	BlockRoot     string  ` + "`json:\"block_root,omitempty\"`" + `
	Epoch         uint64  ` + "`json:\"epoch\"`" + `
	ValidatorID   *uint64 ` + "`json:\"validator_id,omitempty\"`" + `
}`,
			expected: `type Response struct {
	Slot          uint64  ` + "`json:\"slot\" ch:\"slot\"`" + `
	BlockRoot     string  ` + "`json:\"block_root,omitempty\" ch:\"block_root\"`" + `
	Epoch         uint64  ` + "`json:\"epoch\" ch:\"epoch\"`" + `
	ValidatorID   *uint64 ` + "`json:\"validator_id,omitempty\" ch:\"validator_id\"`" + `
}`,
		},
		{
			name:     "json tag with special characters",
			input:    "`json:\"meta_client_geo_longitude\"`",
			expected: "`json:\"meta_client_geo_longitude\" ch:\"meta_client_geo_longitude\"`",
		},
		{
			name: "multiple structs in same file",
			input: `type Struct1 struct {
	Field1 string ` + "`json:\"field1\"`" + `
}

type Struct2 struct {
	Field2 string ` + "`json:\"field2,omitempty\"`" + `
}`,
			expected: `type Struct1 struct {
	Field1 string ` + "`json:\"field1\" ch:\"field1\"`" + `
}

type Struct2 struct {
	Field2 string ` + "`json:\"field2,omitempty\" ch:\"field2\"`" + `
}`,
		},
		{
			name: "struct with other tags in separate backticks",
			input: `type MyStruct struct {
	Field1 string ` + "`json:\"field1\"`" + ` ` + "`yaml:\"field1\"`" + `
}`,
			expected: `type MyStruct struct {
	Field1 string ` + "`json:\"field1\" ch:\"field1\"`" + ` ` + "`yaml:\"field1\"`" + `
}`,
		},
		{
			name:     "field name with dash",
			input:    "`json:\"field-name\"`",
			expected: "`json:\"field-name\" ch:\"field-name\"`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addChTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddChTagsIntegration(t *testing.T) {
	tests := []struct {
		name         string
		inputContent string
		expected     string
	}{
		{
			name: "full Go file with package and imports",
			inputContent: `package models

import "time"

// Response represents an API response
type Response struct {
	ID        uint64    ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name,omitempty\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at,omitempty\"`" + `
}

// Request represents an API request
type Request struct {
	Filter string ` + "`json:\"filter\"`" + `
	Limit  int    ` + "`json:\"limit,omitempty\"`" + `
}
`,
			expected: `package models

import "time"

// Response represents an API response
type Response struct {
	ID        uint64    ` + "`json:\"id\" ch:\"id\"`" + `
	Name      string    ` + "`json:\"name,omitempty\" ch:\"name\"`" + `
	CreatedAt time.Time ` + "`json:\"created_at\" ch:\"created_at\"`" + `
	UpdatedAt time.Time ` + "`json:\"updated_at,omitempty\" ch:\"updated_at\"`" + `
}

// Request represents an API request
type Request struct {
	Filter string ` + "`json:\"filter\" ch:\"filter\"`" + `
	Limit  int    ` + "`json:\"limit,omitempty\" ch:\"limit\"`" + `
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addChTags(tt.inputContent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddChTagsWithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	inputContent := `package test

type TestStruct struct {
	Field1 string ` + "`json:\"field1\"`" + `
	Field2 int    ` + "`json:\"field2,omitempty\"`" + `
}
`

	expectedContent := `package test

type TestStruct struct {
	Field1 string ` + "`json:\"field1\" ch:\"field1\"`" + `
	Field2 int    ` + "`json:\"field2,omitempty\" ch:\"field2\"`" + `
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err := os.WriteFile(inputFile, []byte(inputContent), 0600)
	require.NoError(t, err)

	// Process the file
	content, err := os.ReadFile(inputFile)
	require.NoError(t, err)

	processed := addChTags(string(content))

	outputFile := filepath.Join(tmpDir, "output.go")
	err = os.WriteFile(outputFile, []byte(processed), 0600)
	require.NoError(t, err)

	// Verify output
	result, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(result))
}

func TestAddChTagsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "json tag with empty value - no match",
			input:    "`json:\"\"`",
			expected: "`json:\"\"`",
		},
		{
			name:     "very long field name",
			input:    "`json:\"this_is_a_very_long_field_name_with_many_underscores_and_characters\"`",
			expected: "`json:\"this_is_a_very_long_field_name_with_many_underscores_and_characters\" ch:\"this_is_a_very_long_field_name_with_many_underscores_and_characters\"`",
		},
		{
			name:     "field name with numbers only",
			input:    "`json:\"12345\"`",
			expected: "`json:\"12345\" ch:\"12345\"`",
		},
		{
			name:     "consecutive json tags on same line",
			input:    `Field1 string ` + "`json:\"field1\"`" + ` ` + "`json:\"alias\"`",
			expected: `Field1 string ` + "`json:\"field1\" ch:\"field1\"`" + ` ` + "`json:\"alias\" ch:\"alias\"`",
		},
		{
			name:     "json tag inside comment (should still be processed)",
			input:    "// This is a comment with `json:\"field\"`",
			expected: "// This is a comment with `json:\"field\" ch:\"field\"`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addChTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddChTagsPreservesFormatting(t *testing.T) {
	input := `package main

import "fmt"

type User struct {
	// ID is the user identifier
	ID   string ` + "`json:\"id\"`" + `

	// Name is the user's full name
	Name string ` + "`json:\"name,omitempty\"`" + `
}

func main() {
	fmt.Println("Hello")
}
`

	expected := `package main

import "fmt"

type User struct {
	// ID is the user identifier
	ID   string ` + "`json:\"id\" ch:\"id\"`" + `

	// Name is the user's full name
	Name string ` + "`json:\"name,omitempty\" ch:\"name\"`" + `
}

func main() {
	fmt.Println("Hello")
}
`

	result := addChTags(input)
	assert.Equal(t, expected, result)
}
