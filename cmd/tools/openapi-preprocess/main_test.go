package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Utility Function Tests
// ============================================================================

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
			name:     "already snake_case",
			input:    "slot_start_date_time",
			expected: "slot_start_date_time",
		},
		{
			name:     "consecutive capitals",
			input:    "HTTPServer",
			expected: "http_server",
		},
		{
			name:     "capitals at start",
			input:    "StartDateTime",
			expected: "start_date_time",
		},
		{
			name:     "single capital letter",
			input:    "A",
			expected: "a",
		},
		{
			name:     "mixed case with acronym",
			input:    "metaClientGeoLongitude",
			expected: "meta_client_geo_longitude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := camelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFixCapitalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase letter after digit",
			input:    "50ms",
			expected: "50Ms",
		},
		{
			name:     "multiple digits with letters",
			input:    "100ms200us",
			expected: "100Ms200Us",
		},
		{
			name:     "no changes needed",
			input:    "50Ms",
			expected: "50Ms",
		},
		{
			name:     "only digits",
			input:    "123",
			expected: "123",
		},
		{
			name:     "only letters",
			input:    "abc",
			expected: "abc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "digit at end",
			input:    "abc123",
			expected: "abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixCapitalization(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractServiceNameFromOperationID(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		expected    string
	}{
		{
			name:        "standard operation ID",
			operationID: "FctAttestationService_List",
			expected:    "FctAttestationService",
		},
		{
			name:        "single part",
			operationID: "FctAttestationService",
			expected:    "FctAttestationService",
		},
		{
			name:        "multiple underscores",
			operationID: "Fct_Attestation_Service_List",
			expected:    "Fct",
		},
		{
			name:        "empty string",
			operationID: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceNameFromOperationID(tt.operationID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFieldDescription(t *testing.T) {
	tests := []struct {
		name     string
		comment  string
		expected string
	}{
		{
			name:     "basic filter comment",
			comment:  "Filter by slot - The slot number",
			expected: "The slot number",
		},
		{
			name:     "filter comment with primary key note",
			comment:  "Filter by slot - The slot number (PRIMARY KEY - required)",
			expected: "The slot number",
		},
		{
			name:     "non-filter comment",
			comment:  "This is a regular comment",
			expected: "",
		},
		{
			name:     "filter comment without description",
			comment:  "Filter by slot",
			expected: "slot",
		},
		{
			name:     "empty comment",
			comment:  "",
			expected: "",
		},
		{
			name:     "filter comment with multiple dashes",
			comment:  "Filter by slot - The slot - number",
			expected: "The slot - number",
		},
		{
			name:     "filter comment with parentheses in middle",
			comment:  "Filter by slot - The slot (special) number",
			expected: "The slot (special) number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFieldDescription(tt.comment)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "service found",
			content:  "service FctAttestationService {\n  rpc List(ListRequest) returns (ListResponse);\n}",
			expected: "FctAttestationService",
		},
		{
			name:     "no service",
			content:  "message Request {\n  string id = 1;\n}",
			expected: "",
		},
		{
			name:     "multiple services - returns first",
			content:  "service ServiceOne {}\nservice ServiceTwo {}",
			expected: "ServiceOne",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName([]byte(tt.content))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetArrayItemPattern(t *testing.T) {
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
			name:       "signed integer",
			itemType:   "integer",
			itemFormat: "int32",
			expected:   `^-?\d+(,-?\d+)*$`,
		},
		{
			name:       "number",
			itemType:   "number",
			itemFormat: "double",
			expected:   `^-?\d+(\.\d+)?(,-?\d+(\.\d+)?)*$`,
		},
		{
			name:       "string",
			itemType:   "string",
			itemFormat: "",
			expected:   `^[^,]+(,[^,]+)*$`,
		},
		{
			name:       "unknown type",
			itemType:   "unknown",
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
			result := getArrayItemPattern(tt.itemType, tt.itemFormat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNeedsTypeUpdate(t *testing.T) {
	tests := []struct {
		name           string
		schema         *openapi3.Schema
		correctMapping WrapperTypeMapping
		expected       bool
	}{
		{
			name: "needs type update",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"string"},
				Format: "",
			},
			correctMapping: WrapperTypeMapping{Type: "number", Format: "double"},
			expected:       true,
		},
		{
			name: "needs format update",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"number"},
				Format: "float",
			},
			correctMapping: WrapperTypeMapping{Type: "number", Format: "double"},
			expected:       true,
		},
		{
			name: "no update needed",
			schema: &openapi3.Schema{
				Type:   &openapi3.Types{"number"},
				Format: "double",
			},
			correctMapping: WrapperTypeMapping{Type: "number", Format: "double"},
			expected:       false,
		},
		{
			name: "nil type needs update",
			schema: &openapi3.Schema{
				Type:   nil,
				Format: "",
			},
			correctMapping: WrapperTypeMapping{Type: "number", Format: "double"},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsTypeUpdate(tt.schema, tt.correctMapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// Proto Parsing Tests
// ============================================================================

func TestParseProtoFile(t *testing.T) {
	tests := []struct {
		name               string
		protoContent       string
		expectedDesc       map[string]string
		expectedFieldTypes map[string]string
		expectError        bool
	}{
		{
			name: "parse service with fields and wrapper types",
			protoContent: `syntax = "proto3";

service FctAttestationService {
  rpc List(ListFctAttestationRequest) returns (ListFctAttestationResponse);
}

message ListFctAttestationRequest {
  // Filter by slot - The slot number
  uint64 slot = 1;
  // Filter by block_root - The block root hash
  string block_root = 2;
  google.protobuf.DoubleValue meta_client_geo_longitude = 3;
}`,
			expectedDesc: map[string]string{
				"slot":       "The slot number",
				"block_root": "The block root hash",
			},
			expectedFieldTypes: map[string]string{
				"meta_client_geo_longitude": "DoubleValue",
			},
			expectError: false,
		},
		{
			name: "no service in file",
			protoContent: `syntax = "proto3";

message Request {
  string id = 1;
}`,
			expectedDesc:       map[string]string{},
			expectedFieldTypes: map[string]string{},
			expectError:        false,
		},
		{
			name: "multiple wrapper types",
			protoContent: `syntax = "proto3";

service FctTestService {
  rpc Get(GetFctTestRequest) returns (Response);
}

message GetFctTestRequest {
  google.protobuf.Int32Value count = 1;
  google.protobuf.StringValue name = 2;
  google.protobuf.BoolValue enabled = 3;
}`,
			expectedDesc: map[string]string{},
			expectedFieldTypes: map[string]string{
				"count":   "Int32Value",
				"name":    "StringValue",
				"enabled": "BoolValue",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary proto file
			tmpDir := t.TempDir()
			protoFile := filepath.Join(tmpDir, "test.proto")
			err := os.WriteFile(protoFile, []byte(tt.protoContent), 0600)
			require.NoError(t, err)

			// Parse the file
			descriptions := make(ProtoDescriptions)
			fieldTypes := make(ProtoFieldTypes)

			err = parseProtoFile(protoFile, descriptions, fieldTypes)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check field types
				for field, expectedType := range tt.expectedFieldTypes {
					assert.Equal(t, expectedType, fieldTypes[field], "field type mismatch for %s", field)
				}

				// Check descriptions (if service exists)
				if len(tt.expectedDesc) > 0 {
					for field, expectedDesc := range tt.expectedDesc {
						found := false

						for _, serviceDesc := range descriptions {
							if desc, ok := serviceDesc[field]; ok {
								assert.Equal(t, expectedDesc, desc, "description mismatch for %s", field)

								found = true

								break
							}
						}

						assert.True(t, found, "expected to find description for field %s", field)
					}
				}
			}
		})
	}
}

func TestLoadProtoData(t *testing.T) {
	tests := []struct {
		name        string
		protoFiles  map[string]string
		expectError bool
	}{
		{
			name: "load multiple proto files",
			protoFiles: map[string]string{
				"service1.proto": `syntax = "proto3";
service Service1 {
  rpc List(ListRequest) returns (Response);
}
message ListRequest {
  // Filter by id - The identifier
  string id = 1;
}`,
				"service2.proto": `syntax = "proto3";
service Service2 {
  rpc Get(GetRequest) returns (Response);
}
message GetRequest {
  // Filter by name - The name
  string name = 1;
  google.protobuf.Int64Value count = 2;
}`,
			},
			expectError: false,
		},
		{
			name:        "empty directory",
			protoFiles:  map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory with proto files
			tmpDir := t.TempDir()

			for filename, content := range tt.protoFiles {
				err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0600)
				require.NoError(t, err)
			}

			// Load proto data
			descriptions, fieldTypes, err := loadProtoData(tmpDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, descriptions)
				assert.NotNil(t, fieldTypes)
			}
		})
	}
}

// ============================================================================
// OpenAPI Transformation Tests
// ============================================================================

func TestConvertArrayParamToString(t *testing.T) {
	tests := []struct {
		name           string
		param          *openapi3.Parameter
		expectedType   string
		expectedFormat string
		expectedDesc   string
	}{
		{
			name: "convert integer array",
			param: &openapi3.Parameter{
				Name:        "slot_in_values",
				Description: "Filter slot using in",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type:   &openapi3.Types{"integer"},
								Format: "uint64",
							},
						},
					},
				},
			},
			expectedType:   "string",
			expectedFormat: "",
			expectedDesc:   "Filter slot using in (comma-separated list)",
		},
		{
			name: "convert string array",
			param: &openapi3.Parameter{
				Name:        "name_in_values",
				Description: "Filter name using in",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"array"},
						Items: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"string"},
							},
						},
					},
				},
			},
			expectedType:   "string",
			expectedFormat: "",
			expectedDesc:   "Filter name using in (comma-separated list)",
		},
		{
			name: "nil schema",
			param: &openapi3.Parameter{
				Name:        "test",
				Description: "Test param",
				Schema:      nil,
			},
			expectedType:   "",
			expectedFormat: "",
			expectedDesc:   "Test param",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			convertArrayParamToString(tt.param)

			assert.Equal(t, tt.expectedDesc, tt.param.Description)

			if tt.param.Schema != nil && tt.param.Schema.Value != nil {
				if tt.expectedType != "" {
					assert.Equal(t, tt.expectedType, tt.param.Schema.Value.Type.Slice()[0])
				}

				assert.Equal(t, tt.expectedFormat, tt.param.Schema.Value.Format)
			}
		})
	}
}

func TestFlattenFilterParameters(t *testing.T) {
	tests := []struct {
		name            string
		doc             *openapi3.T
		descriptions    ProtoDescriptions
		expectedChanges int
	}{
		{
			name: "flatten dot notation parameters",
			doc: func() *openapi3.T {
				doc := &openapi3.T{
					Paths: openapi3.NewPaths(),
				}
				doc.Paths.Set("/api/v1/attestations", &openapi3.PathItem{
					Get: &openapi3.Operation{
						OperationID: "FctAttestationService_List",
						Parameters: []*openapi3.ParameterRef{
							{
								Value: &openapi3.Parameter{
									Name: "slotStartDateTime.gte",
									Schema: &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type: &openapi3.Types{"string"},
										},
									},
								},
							},
							{
								Value: &openapi3.Parameter{
									Name: "slotStartDateTime.lt",
									Schema: &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type: &openapi3.Types{"string"},
										},
									},
								},
							},
						},
					},
				})

				return doc
			}(),
			descriptions: ProtoDescriptions{
				"fctattestationservice": {
					"slot_start_date_time": "The slot start date time",
				},
			},
			expectedChanges: 2,
		},
		{
			name: "no changes needed",
			doc: func() *openapi3.T {
				doc := &openapi3.T{
					Paths: openapi3.NewPaths(),
				}
				doc.Paths.Set("/api/v1/attestations", &openapi3.PathItem{
					Get: &openapi3.Operation{
						OperationID: "FctAttestationService_List",
						Parameters: []*openapi3.ParameterRef{
							{
								Value: &openapi3.Parameter{
									Name: "slot",
									Schema: &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type: &openapi3.Types{"integer"},
										},
									},
								},
							},
						},
					},
				})

				return doc
			}(),
			descriptions:    ProtoDescriptions{},
			expectedChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := flattenFilterParameters(tt.doc, tt.descriptions)
			assert.Equal(t, tt.expectedChanges, changes)
		})
	}
}

func TestFixSchemaNames(t *testing.T) {
	tests := []struct {
		name            string
		doc             *openapi3.T
		expectedChanges int
		expectedSchemas []string
	}{
		{
			name: "fix schema capitalization",
			doc: &openapi3.T{
				Components: &openapi3.Components{
					Schemas: openapi3.Schemas{
						"Fct50ms": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
							},
						},
						"Dim100us": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
							},
						},
					},
				},
			},
			expectedChanges: 2,
			expectedSchemas: []string{"Fct50Ms", "Dim100Us"},
		},
		{
			name: "no changes needed",
			doc: &openapi3.T{
				Components: &openapi3.Components{
					Schemas: openapi3.Schemas{
						"FctAttestation": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
							},
						},
					},
				},
			},
			expectedChanges: 0,
			expectedSchemas: []string{"FctAttestation"},
		},
		{
			name:            "nil components",
			doc:             &openapi3.T{},
			expectedChanges: 0,
			expectedSchemas: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := fixSchemaNames(tt.doc)
			assert.Equal(t, tt.expectedChanges, changes)

			if tt.doc.Components != nil && tt.doc.Components.Schemas != nil {
				for _, schemaName := range tt.expectedSchemas {
					_, exists := tt.doc.Components.Schemas[schemaName]
					assert.True(t, exists, "expected schema %s to exist", schemaName)
				}
			}
		})
	}
}

func TestFixWrapperTypes(t *testing.T) {
	tests := []struct {
		name            string
		doc             *openapi3.T
		fieldTypes      ProtoFieldTypes
		expectedChanges int
	}{
		{
			name: "fix wrapper types",
			doc: &openapi3.T{
				Components: &openapi3.Components{
					Schemas: openapi3.Schemas{
						"Request": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
								Properties: openapi3.Schemas{
									"meta_client_geo_longitude": &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type:   &openapi3.Types{"string"},
											Format: "",
										},
									},
									"meta_client_geo_latitude": &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type:   &openapi3.Types{"string"},
											Format: "",
										},
									},
								},
							},
						},
					},
				},
			},
			fieldTypes: ProtoFieldTypes{
				"meta_client_geo_longitude": "DoubleValue",
				"meta_client_geo_latitude":  "DoubleValue",
			},
			expectedChanges: 2,
		},
		{
			name: "no changes needed - correct types",
			doc: &openapi3.T{
				Components: &openapi3.Components{
					Schemas: openapi3.Schemas{
						"Request": &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"object"},
								Properties: openapi3.Schemas{
									"meta_client_geo_longitude": &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Type:   &openapi3.Types{"number"},
											Format: "double",
										},
									},
								},
							},
						},
					},
				},
			},
			fieldTypes: ProtoFieldTypes{
				"meta_client_geo_longitude": "DoubleValue",
			},
			expectedChanges: 0,
		},
		{
			name:            "nil components",
			doc:             &openapi3.T{},
			fieldTypes:      ProtoFieldTypes{},
			expectedChanges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := fixWrapperTypes(tt.doc, tt.fieldTypes)
			assert.Equal(t, tt.expectedChanges, changes)
		})
	}
}

func TestApplyTransformations(t *testing.T) {
	doc := &openapi3.T{
		Paths: openapi3.NewPaths(),
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"Test50ms": &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"object"},
						Properties: openapi3.Schemas{
							"count": &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"string"},
									Format: "",
								},
							},
						},
					},
				},
			},
		},
	}
	doc.Paths.Set("/api/v1/test", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "TestService_Get",
			Parameters: []*openapi3.ParameterRef{
				{
					Value: &openapi3.Parameter{
						Name: "slot.gte",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Type: &openapi3.Types{"integer"},
							},
						},
					},
				},
			},
		},
	})

	descriptions := ProtoDescriptions{
		"testservice": {
			"slot": "The slot number",
		},
	}

	fieldTypes := ProtoFieldTypes{
		"count": "Int64Value",
	}

	annotations := ProtoFieldAnnotations{}

	excludePatterns := []string{} // No exclusions for this test

	stats := applyTransformations(doc, descriptions, fieldTypes, annotations, excludePatterns)

	// Verify stats
	assert.Equal(t, 1, stats.FiltersFlatted, "expected 1 parameter to be flattened")
	assert.Equal(t, 1, stats.SchemasFixed, "expected 1 schema to be fixed")
	assert.Equal(t, 1, stats.TypesFixed, "expected 1 type to be fixed")
	assert.Equal(t, 0, stats.PathsExcluded, "expected 0 paths to be excluded")

	// Verify parameter was renamed
	param := doc.Paths.Map()["/api/v1/test"].Get.Parameters[0].Value
	assert.Equal(t, "slot_gte", param.Name)
	assert.Contains(t, param.Description, "The slot number")

	// Verify schema was renamed
	_, exists := doc.Components.Schemas["Test50Ms"]
	assert.True(t, exists)

	// Verify type was fixed
	countProp := doc.Components.Schemas["Test50Ms"].Value.Properties["count"].Value
	assert.Equal(t, "integer", countProp.Type.Slice()[0])
	assert.Equal(t, "int64", countProp.Format)
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestWriteOpenAPIYAML(t *testing.T) {
	doc := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: &openapi3.Paths{},
	}

	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.yaml")

	err := writeOpenAPIYAML(doc, outputFile)
	require.NoError(t, err)

	// Verify file exists and is readable
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "Test API")
}
