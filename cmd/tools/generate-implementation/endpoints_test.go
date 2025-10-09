package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetItemType(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		expected  string
	}{
		{
			name:      "simple table",
			tableName: "fct_block",
			expected:  "FctBlock",
		},
		{
			name:      "table with number",
			tableName: "last_24h",
			expected:  "Last24H",
		},
		{
			name:      "complex table name",
			tableName: "fct_node_active_last_24h",
			expected:  "FctNodeActiveLast24H",
		},
		{
			name:      "table with milliseconds",
			tableName: "fct_attestation_first_seen_chunked_50ms",
			expected:  "FctAttestationFirstSeenChunked50Ms",
		},
		{
			name:      "single word",
			tableName: "block",
			expected:  "Block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getItemType(tt.tableName)
			assert.Equal(t, tt.expected, got, "getItemType(%q)", tt.tableName)
		})
	}
}

func TestGetItemFieldName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		expected  string
	}{
		{
			name:      "simple table",
			tableName: "fct_block",
			expected:  "FctBlock",
		},
		{
			name:      "table with 24h suffix",
			tableName: "fct_node_active_last_24h",
			expected:  "FctNodeActiveLast24h",
		},
		{
			name:      "table with 50ms suffix",
			tableName: "fct_attestation_first_seen_chunked_50ms",
			expected:  "FctAttestationFirstSeenChunked50ms",
		},
		{
			name:      "table with multiple underscores",
			tableName: "fct_beacon_block_execution_transaction",
			expected:  "FctBeaconBlockExecutionTransaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getItemFieldName(tt.tableName)
			assert.Equal(t, tt.expected, got, "getItemFieldName(%q)", tt.tableName)
		})
	}
}

func TestGenerateBuilderArgs(t *testing.T) {
	tests := []struct {
		name       string
		params     []Param
		filterType string
		expected   string
	}{
		{
			name: "numeric filter with eq only",
			params: []Param{
				{Name: "slot_eq", Operator: "eq"},
			},
			filterType: "UInt32Filter",
			expected:   "params.SlotEq, nil, nil, nil, nil, nil, nil, nil",
		},
		{
			name: "numeric filter with eq and gte",
			params: []Param{
				{Name: "slot_eq", Operator: "eq"},
				{Name: "slot_gte", Operator: "gte"},
			},
			filterType: "UInt32Filter",
			expected:   "params.SlotEq, nil, nil, nil, nil, params.SlotGte, nil, nil",
		},
		{
			name: "numeric filter with range",
			params: []Param{
				{Name: "slot_gte", Operator: "gte"},
				{Name: "slot_lte", Operator: "lte"},
			},
			filterType: "UInt32Filter",
			expected:   "nil, nil, nil, params.SlotLte, nil, params.SlotGte, nil, nil",
		},
		{
			name: "string filter with contains",
			params: []Param{
				{Name: "block_root_contains", Operator: "contains"},
			},
			filterType: "StringFilter",
			expected:   "nil, nil, params.BlockRootContains, nil, nil, nil, nil, nil, nil",
		},
		{
			name: "string filter with eq and contains",
			params: []Param{
				{Name: "block_root_eq", Operator: "eq"},
				{Name: "block_root_contains", Operator: "contains"},
			},
			filterType: "StringFilter",
			expected:   "params.BlockRootEq, nil, params.BlockRootContains, nil, nil, nil, nil, nil, nil",
		},
		{
			name: "string filter with in list",
			params: []Param{
				{Name: "network_in", Operator: "in"},
			},
			filterType: "StringFilter",
			expected:   "nil, nil, nil, nil, nil, nil, nil, params.NetworkIn, nil",
		},
		{
			name: "bool filter",
			params: []Param{
				{Name: "is_valid_eq", Operator: "eq"},
			},
			filterType: "BoolFilter",
			expected:   "params.IsValidEq, nil",
		},
		{
			name: "map filter with has_key",
			params: []Param{
				{Name: "labels_has_key", Operator: "has_key"},
			},
			filterType: "MapStringStringFilter",
			expected:   "params.LabelsHasKey, nil, nil, nil",
		},
		{
			name: "nullable string filter with is_null",
			params: []Param{
				{Name: "network_is_null", Operator: "is_null"},
			},
			filterType: "NullableStringFilter",
			expected:   "nil, nil, nil, nil, nil, nil, nil, nil, nil, params.NetworkIsNull, nil",
		},
		{
			name: "nullable numeric filter with is_not_null",
			params: []Param{
				{Name: "slot_is_not_null", Operator: "is_not_null"},
			},
			filterType: "NullableUInt32Filter",
			expected:   "nil, nil, nil, nil, nil, nil, nil, nil, nil, params.SlotIsNotNull",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateBuilderArgs(tt.params, tt.filterType)
			assert.Equal(t, tt.expected, got, "generateBuilderArgs(%+v, %q)", tt.params, tt.filterType)
		})
	}
}

func TestGenerateFilterAssignments(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       Endpoint
		protoInfo      *ProtoInfo
		expectedInCode []string
		notInCode      []string
	}{
		{
			name: "endpoint with single filter",
			endpoint: Endpoint{
				TableName: "fct_block",
				Parameters: []Param{
					{Name: "slot_eq", Field: "slot", Operator: "eq"},
				},
			},
			protoInfo: &ProtoInfo{
				RequestFields: map[string]map[string]string{
					"fct_block": {
						"slot": "UInt32Filter",
					},
				},
			},
			expectedInCode: []string{
				"// Filter: slot (UInt32Filter)",
				"req.Slot = buildUInt32Filter(",
			},
			notInCode: []string{},
		},
		{
			name: "endpoint with multiple filters",
			endpoint: Endpoint{
				TableName: "fct_block",
				Parameters: []Param{
					{Name: "slot_eq", Field: "slot", Operator: "eq"},
					{Name: "slot_gte", Field: "slot", Operator: "gte"},
					{Name: "block_root_contains", Field: "block_root", Operator: "contains"},
				},
			},
			protoInfo: &ProtoInfo{
				RequestFields: map[string]map[string]string{
					"fct_block": {
						"slot":       "UInt32Filter",
						"block_root": "StringFilter",
					},
				},
			},
			expectedInCode: []string{
				"// Filter: slot (UInt32Filter)",
				"req.Slot = buildUInt32Filter(",
				"// Filter: block_root (StringFilter)",
				"req.BlockRoot = buildStringFilter(",
			},
			notInCode: []string{},
		},
		{
			name: "endpoint with unmapped filter",
			endpoint: Endpoint{
				TableName: "fct_block",
				Parameters: []Param{
					{Name: "slot_eq", Field: "slot", Operator: "eq"},
					{Name: "unknown_field_eq", Field: "unknown_field", Operator: "eq"},
				},
			},
			protoInfo: &ProtoInfo{
				RequestFields: map[string]map[string]string{
					"fct_block": {
						"slot": "UInt32Filter",
					},
				},
			},
			expectedInCode: []string{
				"// Filter: slot (UInt32Filter)",
				"req.Slot = buildUInt32Filter(",
			},
			notInCode: []string{
				"unknown_field",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilterAssignments(tt.endpoint, tt.protoInfo)

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}

			for _, notExpected := range tt.notInCode {
				assert.NotContains(t, got, notExpected, "generated code should NOT contain: %q", notExpected)
			}
		})
	}
}

func TestGenerateEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       Endpoint
		protoInfo      *ProtoInfo
		expectedInCode []string
		notInCode      []string
	}{
		{
			name: "list endpoint",
			endpoint: Endpoint{
				Path:         "/api/v1/fct_block",
				Method:       "GET",
				OperationID:  "FctBlockService_List",
				HandlerName:  "FctBlockServiceList",
				Operation:    "List",
				ParamsType:   "FctBlockServiceListParams",
				ResponseType: "ListFctBlockResponse",
				TableName:    "fct_block",
				Parameters: []Param{
					{Name: "slot_eq", Field: "slot", Operator: "eq"},
				},
			},
			protoInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_block:List": "BuildListFctBlockQuery",
				},
				RequestTypes: map[string]string{
					"fct_block:List": "ListFctBlockRequest",
				},
				RequestFields: map[string]map[string]string{
					"fct_block": {
						"slot": "UInt32Filter",
					},
				},
			},
			expectedInCode: []string{
				"func (s *Server) FctBlockServiceList(w http.ResponseWriter, r *http.Request, params handlers.FctBlockServiceListParams)",
				"req := &clickhouse.ListFctBlockRequest{",
				"PageSize: 100,",
				"clickhouse.BuildListFctBlockQuery(req, s.buildQueryOptions()...)",
				"s.db.Query(ctx, sqlQuery.Query, sqlQuery.Args...)",
				"var items []handlers.FctBlock",
				"protoToOpenAPIFctBlock(&item)",
				"response := handlers.ListFctBlockResponse{",
			},
			notInCode: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateListEndpoint(tt.endpoint, tt.protoInfo)
			require.NotEmpty(t, got, "generated code should not be empty")

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}

			for _, notExpected := range tt.notInCode {
				assert.NotContains(t, got, notExpected, "generated code should NOT contain: %q", notExpected)
			}
		})
	}
}

func TestGenerateGetEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		endpoint       Endpoint
		protoInfo      *ProtoInfo
		expectedInCode []string
	}{
		{
			name: "get endpoint with path parameter",
			endpoint: Endpoint{
				Path:        "/api/v1/fct_block/{slot}",
				Method:      "GET",
				OperationID: "FctBlockService_Get",
				HandlerName: "FctBlockServiceGet",
				Operation:   "Get",
				TableName:   "fct_block",
				PathParameter: &Param{
					Name:   "slot",
					Field:  "slot",
					GoType: "*uint32",
				},
			},
			protoInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_block:Get": "BuildGetFctBlockQuery",
				},
				RequestTypes: map[string]string{
					"fct_block:Get": "GetFctBlockRequest",
				},
			},
			expectedInCode: []string{
				"func (s *Server) FctBlockServiceGet(w http.ResponseWriter, r *http.Request, slot uint32)",
				"req := &clickhouse.GetFctBlockRequest{",
				"Slot: slot,",
				"clickhouse.BuildGetFctBlockQuery(req,",
				"var item clickhouse.FctBlock",
				"if rows.Next() {",
				"w.WriteHeader(http.StatusNotFound)",
				"response := protoToOpenAPIFctBlock(&item)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateGetEndpoint(tt.endpoint, tt.protoInfo)
			require.NotEmpty(t, got, "generated code should not be empty")

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}
		})
	}
}

func TestGenerateEndpoints(t *testing.T) {
	tests := []struct {
		name           string
		spec           *OpenAPISpec
		protoInfo      *ProtoInfo
		expectedInCode []string
		minLines       int
	}{
		{
			name: "multiple endpoints",
			spec: &OpenAPISpec{
				Endpoints: []Endpoint{
					{
						HandlerName:  "FctBlockServiceList",
						Operation:    "List",
						TableName:    "fct_block",
						ResponseType: "ListFctBlockResponse",
					},
					{
						HandlerName: "FctBlockServiceGet",
						Operation:   "Get",
						TableName:   "fct_block",
						PathParameter: &Param{
							Name:   "slot",
							GoType: "*uint32",
						},
					},
				},
			},
			protoInfo: &ProtoInfo{
				QueryBuilders: map[string]string{
					"fct_block:List": "BuildListFctBlockQuery",
					"fct_block:Get":  "BuildGetFctBlockQuery",
				},
				RequestTypes: map[string]string{
					"fct_block:List": "ListFctBlockRequest",
					"fct_block:Get":  "GetFctBlockRequest",
				},
				RequestFields: map[string]map[string]string{
					"fct_block": {},
				},
			},
			expectedInCode: []string{
				"// Auto-generated endpoint implementations",
				"// DO NOT EDIT - Generated by generate-implementation",
				"func (s *Server) FctBlockServiceList(",
				"func (s *Server) FctBlockServiceGet(",
			},
			minLines: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateEndpoints(tt.spec, tt.protoInfo)
			require.NotEmpty(t, got, "generated code should not be empty")

			lines := strings.Split(got, "\n")
			assert.GreaterOrEqual(t, len(lines), tt.minLines, "should generate at least %d lines", tt.minLines)

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}
		})
	}
}
