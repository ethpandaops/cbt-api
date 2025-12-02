package main

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsFilterOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Comparison operators
		{
			name:     "eq operator",
			input:    "eq",
			expected: true,
		},
		{
			name:     "ne operator",
			input:    "ne",
			expected: true,
		},
		{
			name:     "lt operator",
			input:    "lt",
			expected: true,
		},
		{
			name:     "lte operator",
			input:    "lte",
			expected: true,
		},
		{
			name:     "gt operator",
			input:    "gt",
			expected: true,
		},
		{
			name:     "gte operator",
			input:    "gte",
			expected: true,
		},
		{
			name:     "in operator",
			input:    "in",
			expected: true,
		},
		{
			name:     "not_in operator",
			input:    "not_in",
			expected: true,
		},
		{
			name:     "between operator",
			input:    "between",
			expected: true,
		},
		// String operators
		{
			name:     "contains operator",
			input:    "contains",
			expected: true,
		},
		{
			name:     "starts_with operator",
			input:    "starts_with",
			expected: true,
		},
		{
			name:     "ends_with operator",
			input:    "ends_with",
			expected: true,
		},
		{
			name:     "like operator",
			input:    "like",
			expected: true,
		},
		{
			name:     "not_like operator",
			input:    "not_like",
			expected: true,
		},
		// Null operators
		{
			name:     "is_null operator",
			input:    "is_null",
			expected: true,
		},
		{
			name:     "is_not_null operator",
			input:    "is_not_null",
			expected: true,
		},
		// Map operators
		{
			name:     "has_key operator",
			input:    "has_key",
			expected: true,
		},
		{
			name:     "not_has_key operator",
			input:    "not_has_key",
			expected: true,
		},
		{
			name:     "has_any_key operator",
			input:    "has_any_key",
			expected: true,
		},
		{
			name:     "has_all_keys operator",
			input:    "has_all_keys",
			expected: true,
		},
		// Non-operators
		{
			name:     "invalid operator",
			input:    "foo",
			expected: false,
		},
		{
			name:     "pagination param page_size",
			input:    "page_size",
			expected: false,
		},
		{
			name:     "pagination param page_token",
			input:    "page_token",
			expected: false,
		},
		{
			name:     "partial match not",
			input:    "not",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFilterOperator(tt.input)
			assert.Equal(t, tt.expected, got, "isFilterOperator(%q)", tt.input)
		})
	}
}

func TestToGoType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		format   string
		pointer  bool
		expected string
	}{
		// Integer types with pointer
		{
			name:     "uint32 pointer",
			typ:      "integer",
			format:   "uint32",
			pointer:  true,
			expected: "*uint32",
		},
		{
			name:     "uint64 pointer",
			typ:      "integer",
			format:   "uint64",
			pointer:  true,
			expected: "*uint64",
		},
		{
			name:     "int32 pointer",
			typ:      "integer",
			format:   "int32",
			pointer:  true,
			expected: "*int32",
		},
		{
			name:     "int64 pointer",
			typ:      "integer",
			format:   "int64",
			pointer:  true,
			expected: "*int64",
		},
		// Integer types without pointer
		{
			name:     "uint32 value",
			typ:      "integer",
			format:   "uint32",
			pointer:  false,
			expected: "uint32",
		},
		{
			name:     "uint64 value",
			typ:      "integer",
			format:   "uint64",
			pointer:  false,
			expected: "uint64",
		},
		// String types
		{
			name:     "string pointer",
			typ:      "string",
			format:   "",
			pointer:  true,
			expected: "*string",
		},
		{
			name:     "string value",
			typ:      "string",
			format:   "",
			pointer:  false,
			expected: "string",
		},
		// Boolean types
		{
			name:     "bool pointer",
			typ:      "boolean",
			format:   "",
			pointer:  true,
			expected: "*bool",
		},
		{
			name:     "bool value",
			typ:      "boolean",
			format:   "",
			pointer:  false,
			expected: "bool",
		},
		// Default integer (no format)
		{
			name:     "generic int pointer",
			typ:      "integer",
			format:   "",
			pointer:  true,
			expected: "*int",
		},
		{
			name:     "generic int value",
			typ:      "integer",
			format:   "",
			pointer:  false,
			expected: "int",
		},
		// Unknown types
		{
			name:     "unknown type pointer",
			typ:      "unknown",
			format:   "",
			pointer:  true,
			expected: "*interface{}",
		},
		{
			name:     "unknown type value",
			typ:      "unknown",
			format:   "",
			pointer:  false,
			expected: "interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toGoType(tt.typ, tt.format, tt.pointer)
			assert.Equal(t, tt.expected, got, "toGoType(%q, %q, %v)", tt.typ, tt.format, tt.pointer)
		})
	}
}

func TestToHandlerName(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		expected    string
	}{
		{
			name:        "simple operation ID",
			operationID: "FctBlock_List",
			expected:    "FctBlockList",
		},
		{
			name:        "service operation ID",
			operationID: "FctBlockService_List",
			expected:    "FctBlockServiceList",
		},
		{
			name:        "get operation",
			operationID: "FctBlockService_Get",
			expected:    "FctBlockServiceGet",
		},
		{
			name:        "no underscores",
			operationID: "FctBlockList",
			expected:    "FctBlockList",
		},
		{
			name:        "multiple underscores",
			operationID: "Fct_Node_Active_Last_24H_Service_List",
			expected:    "FctNodeActiveLast24HServiceList",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toHandlerName(tt.operationID)
			assert.Equal(t, tt.expected, got, "toHandlerName(%q)", tt.operationID)
		})
	}
}

func TestToParamsType(t *testing.T) {
	tests := []struct {
		name        string
		handlerName string
		expected    string
	}{
		{
			name:        "list handler",
			handlerName: "FctBlockServiceList",
			expected:    "FctBlockServiceListParams",
		},
		{
			name:        "get handler",
			handlerName: "FctBlockServiceGet",
			expected:    "FctBlockServiceGetParams",
		},
		{
			name:        "complex handler",
			handlerName: "FctNodeActiveLast24HServiceList",
			expected:    "FctNodeActiveLast24HServiceListParams",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toParamsType(tt.handlerName)
			assert.Equal(t, tt.expected, got, "toParamsType(%q)", tt.handlerName)
		})
	}
}

func TestToResponseType(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		expected  string
	}{
		{
			name:      "simple table",
			tableName: "fct_block",
			expected:  "ListFctBlockResponse",
		},
		{
			name:      "complex table",
			tableName: "fct_node_active_last_24h",
			expected:  "ListFctNodeActiveLast24HResponse",
		},
		{
			name:      "table with milliseconds",
			tableName: "fct_attestation_first_seen_chunked_50ms",
			expected:  "ListFctAttestationFirstSeenChunked50MsResponse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toResponseType(tt.tableName)
			assert.Equal(t, tt.expected, got, "toResponseType(%q)", tt.tableName)
		})
	}
}

func TestParseParam(t *testing.T) {
	tests := []struct {
		name     string
		param    *openapi3.Parameter
		expected Param
	}{
		{
			name: "simple eq filter",
			param: &openapi3.Parameter{
				Name: "slot_eq",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{"integer"},
						Format: "uint32",
					},
				},
			},
			expected: Param{
				Name:     "slot_eq",
				Field:    "slot",
				Operator: "eq",
				Type:     "integer",
				Format:   "uint32",
				GoType:   "*uint32",
			},
		},
		{
			name: "gte filter",
			param: &openapi3.Parameter{
				Name: "slot_gte",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{"integer"},
						Format: "uint32",
					},
				},
			},
			expected: Param{
				Name:     "slot_gte",
				Field:    "slot",
				Operator: "gte",
				Type:     "integer",
				Format:   "uint32",
				GoType:   "*uint32",
			},
		},
		{
			name: "string contains filter",
			param: &openapi3.Parameter{
				Name: "block_root_contains",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type: &openapi3.Types{"string"},
					},
				},
			},
			expected: Param{
				Name:     "block_root_contains",
				Field:    "block_root",
				Operator: "contains",
				Type:     "string",
				Format:   "",
				GoType:   "*string",
			},
		},
		{
			name: "two-part operator not_in",
			param: &openapi3.Parameter{
				Name: "slot_not_in",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{"string"},
						Format: "",
					},
				},
			},
			// Two-part operators are now correctly parsed before single-part operators
			// So "slot_not_in" is parsed as field="slot", operator="not_in"
			expected: Param{
				Name:     "slot_not_in",
				Field:    "slot",
				Operator: "not_in",
				Type:     "string",
				Format:   "",
				GoType:   "*string",
			},
		},
		{
			name: "pagination param page_size",
			param: &openapi3.Parameter{
				Name: "page_size",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{"integer"},
						Format: "uint32",
					},
				},
			},
			expected: Param{
				Name:     "page_size",
				Field:    "page_size",
				Operator: "",
				Type:     "integer",
				Format:   "uint32",
				GoType:   "*uint32",
			},
		},
		{
			name: "field with underscore",
			param: &openapi3.Parameter{
				Name: "slot_start_date_time_eq",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						Type:   &openapi3.Types{"string"},
						Format: "",
					},
				},
			},
			expected: Param{
				Name:     "slot_start_date_time_eq",
				Field:    "slot_start_date_time",
				Operator: "eq",
				Type:     "string",
				Format:   "",
				GoType:   "*string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseParam(tt.param)
			assert.Equal(t, tt.expected.Name, got.Name, "Name mismatch")
			assert.Equal(t, tt.expected.Field, got.Field, "Field mismatch")
			assert.Equal(t, tt.expected.Operator, got.Operator, "Operator mismatch")
			assert.Equal(t, tt.expected.Type, got.Type, "Type mismatch")
			assert.Equal(t, tt.expected.Format, got.Format, "Format mismatch")
			assert.Equal(t, tt.expected.GoType, got.GoType, "GoType mismatch")
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		op       *openapi3.Operation
		expected Endpoint
	}{
		{
			name:   "list endpoint",
			path:   "/api/v1/fct_block",
			method: "GET",
			op: &openapi3.Operation{
				OperationID: "FctBlockService_List",
				Parameters: []*openapi3.ParameterRef{
					{
						Value: &openapi3.Parameter{
							Name: "slot_eq",
							In:   "query",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"integer"},
									Format: "uint32",
								},
							},
						},
					},
				},
			},
			expected: Endpoint{
				Path:          "/api/v1/fct_block",
				Method:        "GET",
				OperationID:   "FctBlockService_List",
				HandlerName:   "FctBlockServiceList",
				Operation:     "List",
				TableName:     "fct_block",
				ParamsType:    "FctBlockServiceListParams",
				ResponseType:  "ListFctBlockResponse",
				PathParameter: nil,
			},
		},
		{
			name:   "get endpoint with path parameter",
			path:   "/api/v1/fct_block/{slot}",
			method: "GET",
			op: &openapi3.Operation{
				OperationID: "FctBlockService_Get",
				Parameters: []*openapi3.ParameterRef{
					{
						Value: &openapi3.Parameter{
							Name: "slot",
							In:   "path",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type:   &openapi3.Types{"integer"},
									Format: "uint32",
								},
							},
						},
					},
				},
			},
			expected: Endpoint{
				Path:         "/api/v1/fct_block/{slot}",
				Method:       "GET",
				OperationID:  "FctBlockService_Get",
				HandlerName:  "FctBlockServiceGet",
				Operation:    "Get",
				TableName:    "fct_block",
				ParamsType:   "FctBlockServiceGetParams",
				ResponseType: "ListFctBlockResponse",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEndpoint("/api/v1", tt.path, tt.method, tt.op)

			assert.Equal(t, tt.expected.Path, got.Path, "Path mismatch")
			assert.Equal(t, tt.expected.Method, got.Method, "Method mismatch")
			assert.Equal(t, tt.expected.OperationID, got.OperationID, "OperationID mismatch")
			assert.Equal(t, tt.expected.HandlerName, got.HandlerName, "HandlerName mismatch")
			assert.Equal(t, tt.expected.Operation, got.Operation, "Operation mismatch")
			assert.Equal(t, tt.expected.TableName, got.TableName, "TableName mismatch")
			assert.Equal(t, tt.expected.ParamsType, got.ParamsType, "ParamsType mismatch")
			assert.Equal(t, tt.expected.ResponseType, got.ResponseType, "ResponseType mismatch")

			if tt.expected.PathParameter != nil {
				require.NotNil(t, got.PathParameter, "PathParameter should not be nil")
				assert.Equal(t, tt.expected.PathParameter.Name, got.PathParameter.Name, "PathParameter name mismatch")
			}
		})
	}
}
