package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

// TestCamelToSnake tests camelCase to snake_case conversion.
func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple camelCase",
			input:    "slotStartDateTime",
			expected: "slot_start_date_time",
		},
		{
			name:     "already snake_case",
			input:    "slot_start_date_time",
			expected: "slot_start_date_time",
		},
		{
			name:     "single word",
			input:    "slot",
			expected: "slot",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "acronyms",
			input:    "HTTPResponse",
			expected: "http_response",
		},
		{
			name:     "consecutive caps",
			input:    "IOError",
			expected: "io_error",
		},
		{
			name:     "number suffix",
			input:    "last24h",
			expected: "last24h",
		},
		{
			name:     "filter operator",
			input:    "eq",
			expected: "eq",
		},
		{
			name:     "complex field name",
			input:    "metaClientName",
			expected: "meta_client_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := camelToSnake(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractServiceName tests service name extraction from operation IDs.
func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		expected    string
	}{
		{
			name:        "standard service with List",
			operationID: "FctAttestationService_List",
			expected:    "FctAttestationService",
		},
		{
			name:        "standard service with Get",
			operationID: "FctBlockService_Get",
			expected:    "FctBlockService",
		},
		{
			name:        "service with number in name",
			operationID: "FctNodeActiveLast24HService_List",
			expected:    "FctNodeActiveLast24HService",
		},
		{
			name:        "empty operation ID",
			operationID: "",
			expected:    "",
		},
		{
			name:        "no underscore",
			operationID: "ServiceName",
			expected:    "ServiceName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.operationID)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestGetPatternForArrayType tests regex pattern generation for different types.
func TestGetPatternForArrayType(t *testing.T) {
	tests := []struct {
		name       string
		itemType   string
		itemFormat string
		expected   string
	}{
		{
			name:       "unsigned integer",
			itemType:   "integer",
			itemFormat: "uint32",
			expected:   `^\d+(,\d+)*$`,
		},
		{
			name:       "unsigned 64-bit integer",
			itemType:   "integer",
			itemFormat: "uint64",
			expected:   `^\d+(,\d+)*$`,
		},
		{
			name:       "signed integer",
			itemType:   "integer",
			itemFormat: "int32",
			expected:   `^-?\d+(,-?\d+)*$`,
		},
		{
			name:       "integer without format",
			itemType:   "integer",
			itemFormat: "",
			expected:   `^-?\d+(,-?\d+)*$`,
		},
		{
			name:       "floating point number",
			itemType:   "number",
			itemFormat: "",
			expected:   `^-?\d+(\.\d+)?(,-?\d+(\.\d+)?)*$`,
		},
		{
			name:       "string type",
			itemType:   "string",
			itemFormat: "",
			expected:   `^[^,]+(,[^,]+)*$`,
		},
		{
			name:       "unknown type",
			itemType:   "boolean",
			itemFormat: "",
			expected:   "",
		},
		{
			name:       "empty type",
			itemType:   "",
			itemFormat: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPatternForArrayType(tt.itemType, tt.itemFormat)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestParseProtoFile tests proto file parsing for field descriptions.
func TestParseProtoFile(t *testing.T) {
	// Create test fixture
	testProto := `syntax = "proto3";

package cbt;

import "common.proto";

option go_package = "github.com/ethpandaops/xatu-cbt/pkg/proto/clickhouse";

// Test message
message TestModel {
  // Timestamp when the record was last updated
  uint32 updated_date_time = 1;
  // Start block number of the chunk
  uint32 chunk_start_block_number = 2;
}

// Request for listing test_model records
message ListTestModelRequest {
  // Filter by chunk_start_block_number - Start block number of the chunk (PRIMARY KEY - required)
  UInt32Filter chunk_start_block_number = 1;

  // Filter by updated_date_time - Timestamp when the record was last updated (optional)
  UInt32Filter updated_date_time = 2;

  // The maximum number of results to return
  int32 page_size = 3;
}

// Response for listing test_model records
message ListTestModelResponse {
  repeated TestModel test_model = 1;
  string next_page_token = 2;
}

// Query test_model data
service TestModelService {
  rpc List(ListTestModelRequest) returns (ListTestModelResponse);
}
`

	// Write test proto file
	tmpDir := t.TempDir()
	protoFile := filepath.Join(tmpDir, "test_model.proto")
	err := os.WriteFile(protoFile, []byte(testProto), 0644)
	require.NoError(t, err)

	// Parse the file
	descriptions := make(ProtoDescriptions)
	err = parseProtoFile(protoFile, descriptions)
	require.NoError(t, err)

	// Verify service was found (lowercase for case-insensitive lookup)
	serviceDesc, ok := descriptions["testmodelservice"]
	require.True(t, ok, "TestModelService should be found")

	// Verify field descriptions
	require.Equal(t, "Start block number of the chunk", serviceDesc["chunk_start_block_number"])
	require.Equal(t, "Timestamp when the record was last updated", serviceDesc["updated_date_time"])

	// page_size should not be included (no "Filter by" comment)
	_, ok = serviceDesc["page_size"]
	require.False(t, ok, "page_size should not be in descriptions")
}

// TestLoadProtoDescriptions tests loading multiple proto files.
func TestLoadProtoDescriptions(t *testing.T) {
	// Create test fixtures
	tmpDir := t.TempDir()

	proto1 := `syntax = "proto3";
package cbt;

message ListService1Request {
  // Filter by field_one - Description for field one
  UInt32Filter field_one = 1;
}

service Service1 {
  rpc List(ListService1Request) returns (ListService1Response);
}
`

	proto2 := `syntax = "proto3";
package cbt;

message ListService2Request {
  // Filter by field_two - Description for field two
  StringFilter field_two = 1;
}

service Service2 {
  rpc List(ListService2Request) returns (ListService2Response);
}
`

	err := os.WriteFile(filepath.Join(tmpDir, "service1.proto"), []byte(proto1), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "service2.proto"), []byte(proto2), 0644)
	require.NoError(t, err)

	// Load all proto files
	descriptions, err := loadProtoDescriptions(tmpDir)
	require.NoError(t, err)

	// Verify both services were loaded
	require.Len(t, descriptions, 2)
	require.Contains(t, descriptions, "service1")
	require.Contains(t, descriptions, "service2")

	// Verify field descriptions
	require.Equal(t, "Description for field one", descriptions["service1"]["field_one"])
	require.Equal(t, "Description for field two", descriptions["service2"]["field_two"])
}

// TestConvertDotToUnderscore tests parameter conversion and description application.
func TestConvertDotToUnderscore(t *testing.T) {
	tests := []struct {
		name              string
		operationID       string
		inputParams       []string
		descriptions      ProtoDescriptions
		expectedNames     []string
		expectedDescs     []string
		expectedConverted int
	}{
		{
			name:        "basic conversion with description",
			operationID: "FctBlockService_List",
			inputParams: []string{"slotStartDateTime.eq"},
			descriptions: ProtoDescriptions{
				"fctblockservice": {
					"slot_start_date_time": "The wall clock time when the slot started",
				},
			},
			expectedNames:     []string{"slot_start_date_time_eq"},
			expectedDescs:     []string{"The wall clock time when the slot started (filter: eq)"},
			expectedConverted: 1,
		},
		{
			name:        "conversion without description",
			operationID: "FctBlockService_List",
			inputParams: []string{"unknownField.ne"},
			descriptions: ProtoDescriptions{
				"fctblockservice": {},
			},
			expectedNames:     []string{"unknown_field_ne"},
			expectedDescs:     []string{"Filter unknown_field using ne"},
			expectedConverted: 1,
		},
		{
			name:              "no dot notation - no conversion",
			operationID:       "FctBlockService_List",
			inputParams:       []string{"page_size"},
			descriptions:      ProtoDescriptions{},
			expectedNames:     []string{"page_size"},
			expectedDescs:     []string{""},
			expectedConverted: 0,
		},
		{
			name:        "multiple operators",
			operationID: "FctBlockService_List",
			inputParams: []string{"slotStartDateTime.gt", "slotStartDateTime.lt"},
			descriptions: ProtoDescriptions{
				"fctblockservice": {
					"slot_start_date_time": "The wall clock time when the slot started",
				},
			},
			expectedNames: []string{"slot_start_date_time_gt", "slot_start_date_time_lt"},
			expectedDescs: []string{
				"The wall clock time when the slot started (filter: gt)",
				"The wall clock time when the slot started (filter: lt)",
			},
			expectedConverted: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create operation with parameters
			op := &openapi3.Operation{
				OperationID: tt.operationID,
				Parameters:  make(openapi3.Parameters, len(tt.inputParams)),
			}

			for i, paramName := range tt.inputParams {
				op.Parameters[i] = &openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name: paramName,
						In:   "query",
					},
				}
			}

			// Convert parameters
			converted := convertDotToUnderscore(op, "/test/path", tt.descriptions)

			// Verify conversion count
			require.Equal(t, tt.expectedConverted, converted)

			// Verify parameter names and descriptions
			for i, expectedName := range tt.expectedNames {
				require.Equal(t, expectedName, op.Parameters[i].Value.Name)

				if tt.expectedDescs[i] != "" {
					require.Equal(t, tt.expectedDescs[i], op.Parameters[i].Value.Description)
				}
			}
		})
	}
}

// TestArrayParameterConversion tests conversion of array parameters to comma-separated strings.
func TestArrayParameterConversion(t *testing.T) {
	// Create operation with array parameter
	op := &openapi3.Operation{
		OperationID: "FctBlockService_List",
		Parameters: openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name: "slotStartDateTime.inValues",
					In:   "query",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: &openapi3.Types{"array"},
							Items: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"integer"},
									Format: "uint32",
								},
							},
						},
					},
				},
			},
		},
	}

	descriptions := ProtoDescriptions{
		"fctblockservice": {
			"slot_start_date_time": "The wall clock time when the slot started",
		},
	}

	// Convert parameters
	converted := convertDotToUnderscore(op, "/test/path", descriptions)

	// Verify conversion
	require.Equal(t, 1, converted)

	param := op.Parameters[0].Value

	// Check name conversion
	require.Equal(t, "slot_start_date_time_in_values", param.Name)

	// Check type conversion: array -> string
	require.NotNil(t, param.Schema.Value.Type)
	require.Equal(t, "string", param.Schema.Value.Type.Slice()[0])

	// Check pattern was added
	require.Equal(t, `^\d+(,\d+)*$`, param.Schema.Value.Pattern)

	// Check description includes "comma-separated list"
	require.Contains(t, param.Description, "comma-separated list")

	// Verify items field was cleared
	require.Nil(t, param.Schema.Value.Items)
}

// TestEndToEnd tests the full pipeline with sample OpenAPI and proto files.
func TestEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create sample proto file
	protoContent := `syntax = "proto3";
package cbt;

message ListFctBlockRequest {
  // Filter by slot_start_date_time - The wall clock time when the slot started
  UInt32Filter slot_start_date_time = 1;
  // Filter by block_number - The block number
  UInt32Filter block_number = 2;
}

service FctBlockService {
  rpc List(ListFctBlockRequest) returns (ListFctBlockResponse);
}
`
	protoFile := filepath.Join(tmpDir, "fct_block.proto")
	err := os.WriteFile(protoFile, []byte(protoContent), 0644)
	require.NoError(t, err)

	// Create sample OpenAPI spec with dot notation
	inputSpec := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: openapi3.NewPaths(),
	}

	// Add path with operations
	inputSpec.Paths.Set("/api/v1/fct_block", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "FctBlockService_List",
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Name:        "slotStartDateTime.eq",
						In:          "query",
						Description: "Filter slotStartDateTime using eq",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type:   &openapi3.Types{"integer"},
								Format: "uint32",
							},
						},
					},
				},
				{
					Value: &openapi3.Parameter{
						Name: "blockNumber.inValues",
						In:   "query",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"array"},
								Items: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Type:   &openapi3.Types{"integer"},
										Format: "uint32",
									},
								},
							},
						},
					},
				},
			},
		},
	})

	// Write input OpenAPI file
	inputFile := filepath.Join(tmpDir, "input.yaml")
	inputData, err := inputSpec.MarshalJSON()
	require.NoError(t, err)
	err = os.WriteFile(inputFile, inputData, 0644)
	require.NoError(t, err)

	// Load proto descriptions
	descriptions, err := loadProtoDescriptions(tmpDir)
	require.NoError(t, err)
	require.Contains(t, descriptions, "fctblockservice")

	// Load and process OpenAPI spec
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(inputFile)
	require.NoError(t, err)

	totalConverted := 0

	for _, pathItem := range doc.Paths.Map() {
		if pathItem.Get != nil {
			totalConverted += convertDotToUnderscore(pathItem.Get, "/api/v1/fct_block", descriptions)
		}
	}

	// Verify conversions
	require.Equal(t, 2, totalConverted)

	// Check first parameter (eq operator)
	param1 := doc.Paths.Find("/api/v1/fct_block").Get.Parameters[0].Value
	require.Equal(t, "slot_start_date_time_eq", param1.Name)
	require.Equal(t, "The wall clock time when the slot started (filter: eq)", param1.Description)

	// Check second parameter (inValues -> array to string conversion)
	param2 := doc.Paths.Find("/api/v1/fct_block").Get.Parameters[1].Value
	require.Equal(t, "block_number_in_values", param2.Name)
	require.Equal(t, "string", param2.Schema.Value.Type.Slice()[0])
	require.Equal(t, `^\d+(,\d+)*$`, param2.Schema.Value.Pattern)
	require.Contains(t, param2.Description, "comma-separated list")
	require.Nil(t, param2.Schema.Value.Items)
}

// TestCaseInsensitiveServiceLookup tests that service names are matched case-insensitively.
func TestCaseInsensitiveServiceLookup(t *testing.T) {
	// Create operation with service name that has different casing
	op := &openapi3.Operation{
		OperationID: "FctNodeActiveLast24HService_List", // Capital H (from protoc)
		Parameters: openapi3.Parameters{
			{
				Value: &openapi3.Parameter{
					Name: "lastSeenDateTime.eq",
					In:   "query",
				},
			},
		},
	}

	// Descriptions use lowercase (from proto: FctNodeActiveLast24hService)
	descriptions := ProtoDescriptions{
		"fctnodeactivelast24hservice": { // All lowercase
			"last_seen_date_time": "The last time the node was seen",
		},
	}

	// Convert parameters
	converted := convertDotToUnderscore(op, "/test/path", descriptions)

	// Verify conversion succeeded
	require.Equal(t, 1, converted)

	param := op.Parameters[0].Value
	require.Equal(t, "last_seen_date_time_eq", param.Name)
	require.Equal(t, "The last time the node was seen (filter: eq)", param.Description)
}
