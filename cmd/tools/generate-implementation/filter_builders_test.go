package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToProtoTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "uint32",
			input:    "uint32",
			expected: "UInt32",
		},
		{
			name:     "uint64",
			input:    "uint64",
			expected: "UInt64",
		},
		{
			name:     "int32",
			input:    "int32",
			expected: "Int32",
		},
		{
			name:     "int64",
			input:    "int64",
			expected: "Int64",
		},
		{
			name:     "string",
			input:    "string",
			expected: "String",
		},
		{
			name:     "bool",
			input:    "bool",
			expected: "Bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toProtoTypeName(tt.input)
			assert.Equal(t, tt.expected, got, "toProtoTypeName(%q)", tt.input)
		})
	}
}

func TestGenerateFilterParams(t *testing.T) {
	tests := []struct {
		name     string
		filter   *FilterType
		expected string
	}{
		{
			name: "numeric non-nullable",
			filter: &FilterType{
				Name:       "UInt32Filter",
				BaseType:   "uint32",
				IsNullable: false,
				IsMap:      false,
			},
			expected: "eq, ne, lt, lte, gt, gte *uint32, in, notIn *string",
		},
		{
			name: "numeric nullable",
			filter: &FilterType{
				Name:       "NullableUInt32Filter",
				BaseType:   "uint32",
				IsNullable: true,
				IsMap:      false,
			},
			expected: "eq, ne, lt, lte, gt, gte *uint32, in, notIn *string, isNull, isNotNull *bool",
		},
		{
			name: "string non-nullable",
			filter: &FilterType{
				Name:       "StringFilter",
				BaseType:   "string",
				IsNullable: false,
				IsMap:      false,
			},
			expected: "eq, ne, contains, startsWith, endsWith, like, notLike *string, in, notIn *string",
		},
		{
			name: "string nullable",
			filter: &FilterType{
				Name:       "NullableStringFilter",
				BaseType:   "string",
				IsNullable: true,
				IsMap:      false,
			},
			expected: "eq, ne, contains, startsWith, endsWith, like, notLike *string, in, notIn *string, isNull, isNotNull *bool",
		},
		{
			name: "bool non-nullable",
			filter: &FilterType{
				Name:       "BoolFilter",
				BaseType:   "bool",
				IsNullable: false,
				IsMap:      false,
			},
			expected: "eq, ne *bool",
		},
		{
			name: "bool nullable",
			filter: &FilterType{
				Name:       "NullableBoolFilter",
				BaseType:   "bool",
				IsNullable: true,
				IsMap:      false,
			},
			expected: "eq, ne *bool, isNull, isNotNull *bool",
		},
		{
			name: "int64 non-nullable",
			filter: &FilterType{
				Name:       "Int64Filter",
				BaseType:   "int64",
				IsNullable: false,
				IsMap:      false,
			},
			expected: "eq, ne, lt, lte, gt, gte *int64, in, notIn *string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilterParams(tt.filter)
			assert.Equal(t, tt.expected, got, "generateFilterParams(%+v)", tt.filter)
		})
	}
}

func TestGenerateNilCheck(t *testing.T) {
	tests := []struct {
		name     string
		filter   *FilterType
		expected string
	}{
		{
			name: "string filter",
			filter: &FilterType{
				BaseType: "string",
			},
			expected: "eq == nil && ne == nil && contains == nil && startsWith == nil && endsWith == nil && like == nil && notLike == nil && in == nil && notIn == nil",
		},
		{
			name: "bool filter",
			filter: &FilterType{
				BaseType: "bool",
			},
			expected: "eq == nil && ne == nil",
		},
		{
			name: "numeric filter (uint32)",
			filter: &FilterType{
				BaseType: "uint32",
			},
			expected: "eq == nil && ne == nil && lt == nil && lte == nil && gt == nil && gte == nil && in == nil && notIn == nil",
		},
		{
			name: "numeric filter (int64)",
			filter: &FilterType{
				BaseType: "int64",
			},
			expected: "eq == nil && ne == nil && lt == nil && lte == nil && gt == nil && gte == nil && in == nil && notIn == nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNilCheck(tt.filter)
			assert.Equal(t, tt.expected, got, "generateNilCheck(%+v)", tt.filter)
		})
	}
}

func TestGenerateScalarFilterBuilder(t *testing.T) {
	tests := []struct {
		name           string
		filter         *FilterType
		expectedInCode []string // Strings that should appear in generated code
		notInCode      []string // Strings that should NOT appear
	}{
		{
			name: "bool filter",
			filter: &FilterType{
				Name:     "BoolFilter",
				BaseType: "bool",
			},
			expectedInCode: []string{
				"func buildBoolFilter(eq, ne *bool) *clickhouse.BoolFilter",
				"if eq != nil {",
				"return &clickhouse.BoolFilter{",
				"Filter: &clickhouse.BoolFilter_Eq{Eq: *eq}",
				"if ne != nil {",
				"Filter: &clickhouse.BoolFilter_Ne{Ne: *ne}",
			},
			notInCode: []string{
				"contains",
				"In",
				"Between",
			},
		},
		{
			name: "string filter",
			filter: &FilterType{
				Name:     "StringFilter",
				BaseType: "string",
			},
			expectedInCode: []string{
				"func buildStringFilter(eq, ne, contains, startsWith, endsWith, like, notLike *string, in, notIn *string) *clickhouse.StringFilter",
				"if eq != nil {",
				"if contains != nil {",
				"if startsWith != nil {",
				"if endsWith != nil {",
				"if like != nil {",
				"if notLike != nil {",
				"if in != nil {",
				"parseStringList(*in)",
				"if notIn != nil {",
				"parseStringList(*notIn)",
			},
			notInCode: []string{
				"Between",
				"isNull",
			},
		},
		{
			name: "numeric filter (uint32)",
			filter: &FilterType{
				Name:     "UInt32Filter",
				BaseType: "uint32",
			},
			expectedInCode: []string{
				"func buildUInt32Filter(eq, ne, lt, lte, gt, gte *uint32, in, notIn *string) *clickhouse.UInt32Filter",
				"if eq != nil {",
				"if gte != nil && lte != nil {",
				"Between: &clickhouse.UInt32Range{",
				"&wrapperspb.UInt32Value{Value: *lte}",
				"if lte != nil {",
				"if gte != nil {",
				"if lt != nil {",
				"if gt != nil {",
				"if in != nil {",
				"parseUInt32List(*in)",
				"if notIn != nil {",
			},
			notInCode: []string{
				"contains",
				"isNull",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateScalarFilterBuilder(tt.filter)
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

func TestGenerateNullableFilterBuilder(t *testing.T) {
	tests := []struct {
		name           string
		filter         *FilterType
		expectedInCode []string
		notInCode      []string
	}{
		{
			name: "nullable bool filter",
			filter: &FilterType{
				Name:       "NullableBoolFilter",
				BaseType:   "bool",
				IsNullable: true,
			},
			expectedInCode: []string{
				"func buildNullableBoolFilter(eq, ne *bool, isNull, isNotNull *bool) *clickhouse.NullableBoolFilter",
				"if isNull != nil && *isNull {",
				"Filter: &clickhouse.NullableBoolFilter_IsNull{IsNull: &emptypb.Empty{}}",
				"if isNotNull != nil && *isNotNull {",
				"Filter: &clickhouse.NullableBoolFilter_IsNotNull{IsNotNull: &emptypb.Empty{}}",
				"if eq != nil {",
				"if ne != nil {",
			},
			notInCode: []string{
				"contains",
				"Between",
			},
		},
		{
			name: "nullable string filter",
			filter: &FilterType{
				Name:       "NullableStringFilter",
				BaseType:   "string",
				IsNullable: true,
			},
			expectedInCode: []string{
				"func buildNullableStringFilter(eq, ne, contains, startsWith, endsWith, like, notLike *string, in, notIn *string, isNull, isNotNull *bool) *clickhouse.NullableStringFilter",
				"if isNull != nil && *isNull {",
				"if isNotNull != nil && *isNotNull {",
				"if contains != nil {",
				"if startsWith != nil {",
				"if like != nil {",
			},
			notInCode: []string{
				"Between",
			},
		},
		{
			name: "nullable uint32 filter",
			filter: &FilterType{
				Name:       "NullableUInt32Filter",
				BaseType:   "uint32",
				IsNullable: true,
			},
			expectedInCode: []string{
				"func buildNullableUInt32Filter(eq, ne, lt, lte, gt, gte *uint32, in, notIn *string, isNull, isNotNull *bool) *clickhouse.NullableUInt32Filter",
				"if isNull != nil && *isNull {",
				"if isNotNull != nil && *isNotNull {",
				"if gte != nil && lte != nil {",
				"Between: &clickhouse.UInt32Range{",
			},
			notInCode: []string{
				"contains",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNullableFilterBuilder(tt.filter)
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

func TestGenerateMapFilterBuilder(t *testing.T) {
	tests := []struct {
		name           string
		filter         *FilterType
		expectedInCode []string
	}{
		{
			name: "map string string filter",
			filter: &FilterType{
				Name:     "MapStringStringFilter",
				BaseType: "map[string]string",
				IsMap:    true,
			},
			expectedInCode: []string{
				"func buildMapStringStringFilter(hasKey, notHasKey, hasAnyKey, hasAllKeys *string) *clickhouse.MapStringStringFilter",
				"if hasKey == nil && notHasKey == nil && hasAnyKey == nil && hasAllKeys == nil {",
				"return nil",
				"if hasKey != nil {",
				"Filter: &clickhouse.MapStringStringFilter_HasKey{HasKey: *hasKey}",
				"if notHasKey != nil {",
				"Filter: &clickhouse.MapStringStringFilter_NotHasKey{NotHasKey: *notHasKey}",
				"if hasAnyKey != nil {",
				"keys := strings.Split(*hasAnyKey, \",\")",
				"Filter: &clickhouse.MapStringStringFilter_HasAnyKey{",
				"if hasAllKeys != nil {",
				"Filter: &clickhouse.MapStringStringFilter_HasAllKeys{",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateMapFilterBuilder(tt.filter)
			require.NotEmpty(t, got, "generated code should not be empty")

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}
		})
	}
}

func TestGenerateFilterBuilders(t *testing.T) {
	tests := []struct {
		name           string
		protoInfo      *ProtoInfo
		expectedInCode []string
		minLines       int
	}{
		{
			name: "multiple filter types",
			protoInfo: &ProtoInfo{
				FilterTypes: map[string]*FilterType{
					"UInt32Filter": {
						Name:       "UInt32Filter",
						BaseType:   "uint32",
						IsNullable: false,
						IsMap:      false,
					},
					"StringFilter": {
						Name:       "StringFilter",
						BaseType:   "string",
						IsNullable: false,
						IsMap:      false,
					},
					"MapStringStringFilter": {
						Name:     "MapStringStringFilter",
						BaseType: "map[string]string",
						IsMap:    true,
					},
				},
			},
			expectedInCode: []string{
				"// Auto-generated filter builder functions",
				"// DO NOT EDIT - Generated by generate-implementation",
				"func buildMapStringStringFilter(",
				"func buildStringFilter(",
				"func buildUInt32Filter(",
			},
			minLines: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilterBuilders(tt.protoInfo)
			require.NotEmpty(t, got, "generated code should not be empty")

			lines := strings.Split(got, "\n")
			assert.GreaterOrEqual(t, len(lines), tt.minLines, "should generate at least %d lines", tt.minLines)

			for _, expected := range tt.expectedInCode {
				assert.Contains(t, got, expected, "generated code should contain: %q", expected)
			}
		})
	}
}
