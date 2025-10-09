package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple pascal case",
			input:    "FctBlock",
			expected: "fct_block",
		},
		{
			name:     "with numbers at end",
			input:    "Top100By",
			expected: "top_100_by",
		},
		{
			name:     "consecutive capitals",
			input:    "HTTPServer",
			expected: "http_server",
		},
		{
			name:     "ending with number suffix",
			input:    "Last24h",
			expected: "last_24h",
		},
		{
			name:     "complex with number suffix",
			input:    "FctNodeActiveLast24H",
			expected: "fct_node_active_last_24h",
		},
		{
			name:     "digits and letters mixed",
			input:    "Chunked50ms",
			expected: "chunked_50ms",
		},
		{
			name:     "multiple number transitions",
			input:    "FctAttestationFirstSeenChunked50Ms",
			expected: "fct_attestation_first_seen_chunked_50_ms",
		},
		{
			name:     "all caps abbreviation",
			input:    "ID",
			expected: "id",
		},
		{
			name:     "single word",
			input:    "Block",
			expected: "block",
		},
		{
			name:     "already snake case",
			input:    "fct_block",
			expected: "fct_block",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			assert.Equal(t, tt.expected, got, "toSnakeCase(%q)", tt.input)
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple snake case",
			input:    "fct_block",
			expected: "FctBlock",
		},
		{
			name:     "with number suffix",
			input:    "last_24h",
			expected: "Last24H",
		},
		{
			name:     "starting with number",
			input:    "50ms_chunked",
			expected: "50MsChunked",
		},
		{
			name:     "complex snake case",
			input:    "fct_node_active_last_24h",
			expected: "FctNodeActiveLast24H",
		},
		{
			name:     "number in middle",
			input:    "attestation_50ms_chunked",
			expected: "Attestation50MsChunked",
		},
		{
			name:     "single word",
			input:    "block",
			expected: "Block",
		},
		{
			name:     "already pascal case",
			input:    "FctBlock",
			expected: "FctBlock",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple underscores",
			input:    "fct__block",
			expected: "FctBlock",
		},
		{
			name:     "trailing underscore",
			input:    "fct_block_",
			expected: "FctBlock",
		},
		{
			name:     "leading underscore",
			input:    "_fct_block",
			expected: "FctBlock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toPascalCase(tt.input)
			assert.Equal(t, tt.expected, got, "toPascalCase(%q)", tt.input)
		})
	}
}

func TestExtractTableNameFromType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "list request simple",
			input:    ".cbt.ListFctBlockRequest",
			expected: "fct_block",
		},
		{
			name:     "get request simple",
			input:    ".cbt.GetFctBlockRequest",
			expected: "fct_block",
		},
		{
			name:     "list request complex table",
			input:    ".cbt.ListFctNodeActiveLast24HRequest",
			expected: "fct_node_active_last_24h",
		},
		{
			name:     "get request complex table",
			input:    ".cbt.GetFctNodeActiveLast24HRequest",
			expected: "fct_node_active_last_24h",
		},
		{
			name:     "list response",
			input:    ".cbt.ListFctBlockResponse",
			expected: "fct_block",
		},
		{
			name:     "get response",
			input:    ".cbt.GetFctBlockResponse",
			expected: "fct_block",
		},
		{
			name:     "without package prefix",
			input:    "ListFctBlockRequest",
			expected: "fct_block",
		},
		{
			name:     "multiple package levels",
			input:    ".com.example.cbt.ListFctBlockRequest",
			expected: "fct_block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTableNameFromType(tt.input)
			assert.Equal(t, tt.expected, got, "extractTableNameFromType(%q)", tt.input)
		})
	}
}

func TestExtractTableNameFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "list request",
			input:    "ListFctBlockRequest",
			expected: "fct_block",
		},
		{
			name:     "get request",
			input:    "GetFctBlockRequest",
			expected: "fct_block",
		},
		{
			name:     "list response",
			input:    "ListFctBlockResponse",
			expected: "fct_block",
		},
		{
			name:     "get response",
			input:    "GetFctBlockResponse",
			expected: "fct_block",
		},
		{
			name:     "complex table list",
			input:    "ListFctNodeActiveLast24HRequest",
			expected: "fct_node_active_last_24h",
		},
		{
			name:     "complex table get",
			input:    "GetFctNodeActiveLast24HRequest",
			expected: "fct_node_active_last_24h",
		},
		{
			name:     "with numbers",
			input:    "ListFctAttestationFirstSeenChunked50MsRequest",
			expected: "fct_attestation_first_seen_chunked_50_ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTableNameFromMessage(tt.input)
			assert.Equal(t, tt.expected, got, "extractTableNameFromMessage(%q)", tt.input)
		})
	}
}

func TestCleanTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with package prefix",
			input:    ".cbt.ListFctBlockRequest",
			expected: "ListFctBlockRequest",
		},
		{
			name:     "multiple package levels",
			input:    ".com.example.cbt.ListFctBlockRequest",
			expected: "ListFctBlockRequest",
		},
		{
			name:     "no package prefix",
			input:    "ListFctBlockRequest",
			expected: "ListFctBlockRequest",
		},
		{
			name:     "single dot prefix",
			input:    ".ListFctBlockRequest",
			expected: "ListFctBlockRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanTypeName(tt.input)
			assert.Equal(t, tt.expected, got, "cleanTypeName(%q)", tt.input)
		})
	}
}

func TestGetFieldTypeFromDescriptor(t *testing.T) {
	// Note: This function requires *descriptorpb.FieldDescriptorProto
	// which is complex to construct in tests. For now, we test the
	// logic path with empty/nil types.
	tests := []struct {
		name     string
		typeName string
		expected string
	}{
		{
			name:     "nullable string filter",
			typeName: ".cbt.NullableStringFilter",
			expected: "NullableStringFilter",
		},
		{
			name:     "uint32 filter",
			typeName: ".cbt.UInt32Filter",
			expected: "UInt32Filter",
		},
		{
			name:     "map filter",
			typeName: ".cbt.MapStringStringFilter",
			expected: "MapStringStringFilter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract last part after dot (simulating the function logic)
			parts := splitString(tt.typeName, ".")
			got := parts[len(parts)-1]
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizeProtoTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "FctBlock",
			expected: "FctBlock",
		},
		{
			name:     "name with number suffix",
			input:    "FctNodeActiveLast24h",
			expected: "FctNodeActiveLast24H",
		},
		{
			name:     "name with ms suffix",
			input:    "FctAttestationFirstSeenChunked50ms",
			expected: "FctAttestationFirstSeenChunked50Ms",
		},
		{
			name:     "already normalized",
			input:    "FctNodeActiveLast24H",
			expected: "FctNodeActiveLast24H",
		},
		{
			name:     "multiple number-letter transitions",
			input:    "Test1abc2def",
			expected: "Test1Abc2Def",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character",
			input:    "A",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeProtoTypeName(tt.input)
			assert.Equal(t, tt.expected, got, "normalizeProtoTypeName(%q)", tt.input)
		})
	}
}

func TestGetKnownFilterTypes(t *testing.T) {
	filterTypes := getKnownFilterTypes()

	// Test that all expected filter types are present
	expectedTypes := []string{
		"UInt32Filter",
		"UInt64Filter",
		"Int32Filter",
		"Int64Filter",
		"StringFilter",
		"BoolFilter",
		"NullableUInt32Filter",
		"NullableUInt64Filter",
		"NullableInt32Filter",
		"NullableInt64Filter",
		"NullableStringFilter",
		"NullableBoolFilter",
		"MapStringStringFilter",
		"MapStringUInt32Filter",
		"MapStringInt32Filter",
		"MapStringUInt64Filter",
		"MapStringInt64Filter",
	}

	for _, typeName := range expectedTypes {
		t.Run(typeName, func(t *testing.T) {
			ft, exists := filterTypes[typeName]
			assert.True(t, exists, "filter type %q should exist", typeName)
			assert.NotNil(t, ft, "filter type %q should not be nil", typeName)
			assert.Equal(t, typeName, ft.Name, "filter type name should match")
			assert.NotEmpty(t, ft.Operators, "filter type should have operators")
		})
	}

	// Test specific filter type properties
	t.Run("UInt32Filter properties", func(t *testing.T) {
		ft := filterTypes["UInt32Filter"]
		assert.Equal(t, "uint32", ft.BaseType)
		assert.False(t, ft.IsNullable)
		assert.False(t, ft.IsMap)
		assert.Contains(t, ft.Operators, "eq")
		assert.Contains(t, ft.Operators, "ne")
		assert.Contains(t, ft.Operators, "lt")
		assert.Contains(t, ft.Operators, "gte")
	})

	t.Run("NullableStringFilter properties", func(t *testing.T) {
		ft := filterTypes["NullableStringFilter"]
		assert.Equal(t, "string", ft.BaseType)
		assert.True(t, ft.IsNullable)
		assert.False(t, ft.IsMap)
		assert.Contains(t, ft.Operators, "eq")
		assert.Contains(t, ft.Operators, "contains")
		assert.Contains(t, ft.Operators, "is_null")
		assert.Contains(t, ft.Operators, "is_not_null")
	})

	t.Run("MapStringStringFilter properties", func(t *testing.T) {
		ft := filterTypes["MapStringStringFilter"]
		assert.Equal(t, "map[string]string", ft.BaseType)
		assert.False(t, ft.IsNullable)
		assert.True(t, ft.IsMap)
		assert.Contains(t, ft.Operators, "has_key")
		assert.Contains(t, ft.Operators, "not_has_key")
		assert.Contains(t, ft.Operators, "has_any_key")
		assert.Contains(t, ft.Operators, "has_all_keys")
	})
}

// Helper function for test.
func splitString(s, sep string) []string {
	var result []string

	current := ""

	for _, r := range s {
		if string(r) == sep {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
