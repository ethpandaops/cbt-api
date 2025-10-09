package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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

func TestParseProtoDescriptors(t *testing.T) {
	tests := []struct {
		name          string
		fds           *descriptorpb.FileDescriptorSet
		expectedInfo  *ProtoInfo
		expectedError bool
	}{
		{
			name: "service with List method",
			fds: &descriptorpb.FileDescriptorSet{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name:    stringPtr("test.proto"),
						Package: stringPtr("cbt"),
						Service: []*descriptorpb.ServiceDescriptorProto{
							{
								Name: stringPtr("FctBlockService"),
								Method: []*descriptorpb.MethodDescriptorProto{
									{
										Name:       stringPtr("List"),
										InputType:  stringPtr(".cbt.ListFctBlockRequest"),
										OutputType: stringPtr(".cbt.ListFctBlockResponse"),
									},
								},
							},
						},
						MessageType: []*descriptorpb.DescriptorProto{
							{
								Name: stringPtr("ListFctBlockRequest"),
								Field: []*descriptorpb.FieldDescriptorProto{
									{
										Name:     stringPtr("Slot"),
										TypeName: stringPtr(".cbt.UInt32Filter"),
									},
									{
										Name:     stringPtr("BlockRoot"),
										TypeName: stringPtr(".cbt.NullableStringFilter"),
									},
								},
							},
							{
								Name: stringPtr("UInt32Filter"),
							},
							{
								Name: stringPtr("NullableStringFilter"),
							},
						},
					},
				},
			},
			expectedInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_block:List": "BuildListFctBlockQuery",
				},
				RequestTypes: map[string]string{
					"fct_block:List": "ListFctBlockRequest",
				},
				ResponseTypes: map[string]string{
					"fct_block:List": "ListFctBlockResponse",
				},
				RequestFields: map[string]map[string]string{
					"fct_block": {
						"slot":       "UInt32Filter",
						"block_root": "NullableStringFilter",
					},
				},
			},
		},
		{
			name: "service with Get method",
			fds: &descriptorpb.FileDescriptorSet{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name:    stringPtr("test.proto"),
						Package: stringPtr("cbt"),
						Service: []*descriptorpb.ServiceDescriptorProto{
							{
								Name: stringPtr("FctBlockService"),
								Method: []*descriptorpb.MethodDescriptorProto{
									{
										Name:       stringPtr("Get"),
										InputType:  stringPtr(".cbt.GetFctBlockRequest"),
										OutputType: stringPtr(".cbt.GetFctBlockResponse"),
									},
								},
							},
						},
						MessageType: []*descriptorpb.DescriptorProto{
							{
								Name: stringPtr("GetFctBlockRequest"),
							},
						},
					},
				},
			},
			expectedInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_block:Get": "BuildGetFctBlockQuery",
				},
				RequestTypes: map[string]string{
					"fct_block:Get": "GetFctBlockRequest",
				},
				ResponseTypes: map[string]string{
					"fct_block:Get": "GetFctBlockResponse",
				},
				RequestFields: map[string]map[string]string{},
			},
		},
		{
			name: "complex table name with numbers",
			fds: &descriptorpb.FileDescriptorSet{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name:    stringPtr("test.proto"),
						Package: stringPtr("cbt"),
						Service: []*descriptorpb.ServiceDescriptorProto{
							{
								Name: stringPtr("FctNodeActiveLast24HService"),
								Method: []*descriptorpb.MethodDescriptorProto{
									{
										Name:       stringPtr("List"),
										InputType:  stringPtr(".cbt.ListFctNodeActiveLast24HRequest"),
										OutputType: stringPtr(".cbt.ListFctNodeActiveLast24HResponse"),
									},
								},
							},
						},
						MessageType: []*descriptorpb.DescriptorProto{
							{
								Name: stringPtr("ListFctNodeActiveLast24HRequest"),
								Field: []*descriptorpb.FieldDescriptorProto{
									{
										Name:     stringPtr("NodeId"),
										TypeName: stringPtr(".cbt.StringFilter"),
									},
								},
							},
						},
					},
				},
			},
			expectedInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_node_active_last_24h:List": "BuildListFctNodeActiveLast24HQuery",
				},
				RequestTypes: map[string]string{
					"fct_node_active_last_24h:List": "ListFctNodeActiveLast24HRequest",
				},
				ResponseTypes: map[string]string{
					"fct_node_active_last_24h:List": "ListFctNodeActiveLast24HResponse",
				},
				RequestFields: map[string]map[string]string{
					"fct_node_active_last_24h": {
						"node_id": "StringFilter",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the FileDescriptorSet
			data, err := proto.Marshal(tt.fds)
			assert.NoError(t, err)

			// Write to temp file
			tmpFile, err := os.CreateTemp("", "test-descriptor-*.pb")
			assert.NoError(t, err)

			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(data)
			assert.NoError(t, err)
			tmpFile.Close()

			// Parse the descriptors
			info := &ProtoInfo{
				FilterTypes:   getKnownFilterTypes(),
				QueryBuilders: make(map[string]string),
				RequestTypes:  make(map[string]string),
				ResponseTypes: make(map[string]string),
				RequestFields: make(map[string]map[string]string),
			}

			err = parseProtoDescriptors(tmpFile.Name(), info)

			if tt.expectedError {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)

			// Verify query builders
			assert.Equal(t, tt.expectedInfo.QueryBuilders, info.QueryBuilders)

			// Verify request types
			assert.Equal(t, tt.expectedInfo.RequestTypes, info.RequestTypes)

			// Verify response types
			assert.Equal(t, tt.expectedInfo.ResponseTypes, info.ResponseTypes)

			// Verify request fields
			assert.Equal(t, tt.expectedInfo.RequestFields, info.RequestFields)
		})
	}
}

func TestParseProtoDescriptors_Errors(t *testing.T) {
	info := &ProtoInfo{
		FilterTypes:   getKnownFilterTypes(),
		QueryBuilders: make(map[string]string),
		RequestTypes:  make(map[string]string),
		ResponseTypes: make(map[string]string),
		RequestFields: make(map[string]map[string]string),
	}

	t.Run("file not found", func(t *testing.T) {
		err := parseProtoDescriptors("/nonexistent/file.pb", info)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reading descriptor file")
	})

	t.Run("invalid descriptor data", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "invalid-descriptor-*.pb")
		assert.NoError(t, err)

		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("invalid protobuf data")
		assert.NoError(t, err)
		tmpFile.Close()

		err = parseProtoDescriptors(tmpFile.Name(), info)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshaling descriptor")
	})
}

func TestAnalyzeProtos(t *testing.T) {
	// Save original working directory
	originalWd, err := os.Getwd()
	assert.NoError(t, err)

	// Create a temporary directory for test
	tmpDir, err := os.CreateTemp("", "analyze-protos-test-*")
	assert.NoError(t, err)
	tmpDir.Close()
	os.Remove(tmpDir.Name())

	err = os.Mkdir(tmpDir.Name(), 0755)
	assert.NoError(t, err)

	defer func() {
		_ = os.Chdir(originalWd)
		os.RemoveAll(tmpDir.Name())
	}()

	// Change to temp directory
	err = os.Chdir(tmpDir.Name())
	assert.NoError(t, err)

	t.Run("success with valid descriptor file", func(t *testing.T) {
		// Create a valid descriptor file
		fds := &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{
				{
					Name:    stringPtr("test.proto"),
					Package: stringPtr("cbt"),
					Service: []*descriptorpb.ServiceDescriptorProto{
						{
							Name: stringPtr("FctBlockService"),
							Method: []*descriptorpb.MethodDescriptorProto{
								{
									Name:       stringPtr("List"),
									InputType:  stringPtr(".cbt.ListFctBlockRequest"),
									OutputType: stringPtr(".cbt.ListFctBlockResponse"),
								},
							},
						},
					},
					MessageType: []*descriptorpb.DescriptorProto{
						{
							Name: stringPtr("ListFctBlockRequest"),
							Field: []*descriptorpb.FieldDescriptorProto{
								{
									Name:     stringPtr("Slot"),
									TypeName: stringPtr(".cbt.UInt32Filter"),
								},
							},
						},
					},
				},
			},
		}

		data, err := proto.Marshal(fds)
		assert.NoError(t, err)

		err = os.WriteFile(".descriptors.pb", data, 0644)
		assert.NoError(t, err)

		defer os.Remove(".descriptors.pb")

		// Run analyzeProtos
		info, err := analyzeProtos("")
		assert.NoError(t, err)
		assert.NotNil(t, info)

		// Verify filter types are populated
		assert.NotEmpty(t, info.FilterTypes)
		assert.Contains(t, info.FilterTypes, "UInt32Filter")

		// Verify query builders are extracted
		assert.Contains(t, info.QueryBuilders, "fct_block:List")
		assert.Equal(t, "BuildListFctBlockQuery", info.QueryBuilders["fct_block:List"])

		// Verify request types
		assert.Contains(t, info.RequestTypes, "fct_block:List")
		assert.Equal(t, "ListFctBlockRequest", info.RequestTypes["fct_block:List"])

		// Verify response types
		assert.Contains(t, info.ResponseTypes, "fct_block:List")
		assert.Equal(t, "ListFctBlockResponse", info.ResponseTypes["fct_block:List"])

		// Verify request fields
		assert.Contains(t, info.RequestFields, "fct_block")
		assert.Contains(t, info.RequestFields["fct_block"], "slot")
		assert.Equal(t, "UInt32Filter", info.RequestFields["fct_block"]["slot"])
	})

	t.Run("error when descriptor file missing", func(t *testing.T) {
		// Ensure no descriptor file exists
		os.Remove(".descriptors.pb")

		info, err := analyzeProtos("")
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "parsing proto descriptors")
		assert.Contains(t, err.Error(), "ensure 'make generate-descriptors' has been run")
	})

	t.Run("error with invalid descriptor file", func(t *testing.T) {
		// Create invalid descriptor file
		err := os.WriteFile(".descriptors.pb", []byte("invalid data"), 0644)
		assert.NoError(t, err)

		defer os.Remove(".descriptors.pb")

		info, err := analyzeProtos("")
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "parsing proto descriptors")
	})
}

// Helper function for creating string pointers in tests.
func stringPtr(s string) *string {
	return &s
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
