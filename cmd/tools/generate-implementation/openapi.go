package main

import (
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPISpec represents the parsed OpenAPI specification.
type OpenAPISpec struct {
	Endpoints []Endpoint
	Types     map[string]*Type
}

// Endpoint represents a REST API endpoint.
type Endpoint struct {
	Path          string // "/api/v1/fct_block"
	Method        string // "GET"
	OperationID   string // "FctBlockService_List"
	HandlerName   string // "FctBlockServiceList" (same as oapi-codegen generates)
	Operation     string // "List" or "Get"
	ParamsType    string // "FctBlockServiceListParams"
	ResponseType  string // Item type for responses (e.g., "FctBlock")
	TableName     string // "fct_block"
	Parameters    []Param
	PathParameter *Param // For Get operations: the primary key path parameter
}

// Param represents a parameter.
type Param struct {
	Name     string // "slot_gte"
	Field    string // "slot"
	Operator string // "gte"
	Type     string // "integer"
	Format   string // "uint32"
	GoType   string // "*uint32"
}

// Type represents a schema type.
type Type struct {
	Name   string
	Fields []Field
}

// Field represents a field in a type.
type Field struct {
	Name     string
	Type     string
	JSONTag  string
	Nullable bool
}

// loadOpenAPI loads and parses an OpenAPI specification file.
func loadOpenAPI(path string) (*OpenAPISpec, error) {
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	spec := &OpenAPISpec{
		Types: make(map[string]*Type),
	}

	// Parse endpoints - sort paths for deterministic output
	pathsMap := doc.Paths.Map()
	paths := make([]string, 0, len(pathsMap))

	for path := range pathsMap {
		paths = append(paths, path)
	}

	sort.Strings(paths)

	for _, path := range paths {
		pathItem := doc.Paths.Map()[path]
		if pathItem.Get != nil {
			endpoint := parseEndpoint(path, "GET", pathItem.Get)
			spec.Endpoints = append(spec.Endpoints, endpoint)
		}
	}

	// Parse types/schemas
	for name, schemaRef := range doc.Components.Schemas {
		if schemaRef.Value != nil {
			spec.Types[name] = parseType(name, schemaRef.Value)
		}
	}

	return spec, nil
}

// parseEndpoint parses an OpenAPI operation into an Endpoint struct.
func parseEndpoint(path, method string, op *openapi3.Operation) Endpoint {
	endpoint := Endpoint{
		Path:        path,
		Method:      method,
		OperationID: op.OperationID,
		HandlerName: toHandlerName(op.OperationID),
	}

	// Extract table name from path: "/api/v1/fct_block" → "fct_block"
	// or "/api/v1/fct_block/{slotStartDateTime}" → "fct_block"
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	endpoint.TableName = parts[0]

	// Determine operation type from OperationID: "FctBlockService_List" → "List"
	if strings.HasSuffix(op.OperationID, "_List") {
		endpoint.Operation = "List"
	} else if strings.HasSuffix(op.OperationID, "_Get") {
		endpoint.Operation = "Get"
	}

	// Parse parameters
	for _, paramRef := range op.Parameters {
		if paramRef.Value != nil {
			param := parseParam(paramRef.Value)

			// Check if this is a path parameter (for Get operations)
			if paramRef.Value.In == "path" {
				endpoint.PathParameter = &param
			} else {
				// Query parameters (for List operations)
				endpoint.Parameters = append(endpoint.Parameters, param)
			}
		}
	}

	// Determine types from operation ID
	endpoint.ParamsType = toParamsType(endpoint.HandlerName)
	endpoint.ResponseType = toResponseType(endpoint.TableName)

	return endpoint
}

// parseParam parses an OpenAPI parameter into a Param struct, extracting field and operator from underscore notation.
func parseParam(p *openapi3.Parameter) Param {
	param := Param{
		Name: p.Name,
	}

	// Extract field and operator from underscore notation
	// "slot_gte" → field: "slot", operator: "gte"
	// "slot_not_in_values" → field: "slot", operator: "not_in_values"
	parts := strings.Split(p.Name, "_")
	if len(parts) >= 2 {
		// Check for three-part operators first (e.g., "not_in_values")
		if len(parts) >= 4 {
			lastThreeParts := parts[len(parts)-3] + "_" + parts[len(parts)-2] + "_" + parts[len(parts)-1]
			if isFilterOperator(lastThreeParts) {
				param.Operator = lastThreeParts
				param.Field = strings.Join(parts[:len(parts)-3], "_")

				return param
			}
		}

		// Check if last part is an operator
		lastPart := parts[len(parts)-1]
		if isFilterOperator(lastPart) {
			param.Operator = lastPart
			param.Field = strings.Join(parts[:len(parts)-1], "_")
		} else if len(parts) >= 3 {
			// Check for two-part operators like "not_in", "is_null"
			lastTwoParts := parts[len(parts)-2] + "_" + parts[len(parts)-1]
			if isFilterOperator(lastTwoParts) {
				param.Operator = lastTwoParts
				param.Field = strings.Join(parts[:len(parts)-2], "_")
			} else {
				// Not a filter parameter (e.g., page_size, page_token)
				param.Field = p.Name
			}
		} else {
			// Not a filter parameter
			param.Field = p.Name
		}
	} else {
		// Single word parameter
		param.Field = p.Name
	}

	// Get type info from schema
	if p.Schema != nil && p.Schema.Value != nil {
		if p.Schema.Value.Type != nil && len(p.Schema.Value.Type.Slice()) > 0 {
			param.Type = p.Schema.Value.Type.Slice()[0]
		}

		param.Format = p.Schema.Value.Format
		param.GoType = toGoType(param.Type, param.Format, true) // pointer type
	}

	return param
}

// isFilterOperator checks if a string is a known filter operator.
func isFilterOperator(s string) bool {
	operators := map[string]bool{
		// Comparison operators
		"eq":      true,
		"ne":      true,
		"lt":      true,
		"lte":     true,
		"gt":      true,
		"gte":     true,
		"in":      true,
		"not_in":  true,
		"between": true,

		// Array operators (for numeric filters)
		"in_values":     true,
		"not_in_values": true,

		// String operators
		"contains":    true,
		"starts_with": true,
		"ends_with":   true,
		"like":        true,
		"not_like":    true,

		// Null operators
		"is_null":     true,
		"is_not_null": true,

		// Map operators
		"has_key":      true,
		"not_has_key":  true,
		"has_any_key":  true,
		"has_all_keys": true,
	}

	return operators[s]
}

// toGoType converts OpenAPI type and format to Go type.
func toGoType(typ, format string, pointer bool) string {
	var goType string

	switch typ {
	case "integer":
		switch format {
		case "uint32":
			goType = "uint32"
		case "uint64":
			goType = "uint64"
		case "int32":
			goType = "int32"
		case "int64":
			goType = "int64"
		default:
			goType = "int"
		}
	case "string":
		goType = "string"
	case "boolean":
		goType = "bool"
	default:
		goType = "interface{}"
	}

	if pointer {
		return "*" + goType
	}

	return goType
}

// toHandlerName converts an operation ID to a handler name by removing underscores.
func toHandlerName(operationID string) string {
	return strings.ReplaceAll(operationID, "_", "")
}

// toParamsType converts a handler name to the params type name.
func toParamsType(handlerName string) string {
	// "FctBlockServiceList" → "FctBlockServiceListParams"
	// Note: Get operations don't have Params structs, they use path parameters directly
	return handlerName + "Params"
}

// toResponseType converts a table name to the response type name.
func toResponseType(tableName string) string {
	return "List" + toPascalCase(tableName) + "Response"
}

// parseType parses an OpenAPI schema into a Type struct.
func parseType(name string, schema *openapi3.Schema) *Type {
	t := &Type{Name: name}

	if schema.Properties != nil {
		for propName, propRef := range schema.Properties {
			if propRef.Value != nil {
				var propType string

				if propRef.Value.Type != nil && len(propRef.Value.Type.Slice()) > 0 {
					baseType := propRef.Value.Type.Slice()[0]

					// Handle array types specially
					if baseType == "array" {
						propType = "array"
					} else {
						propType = baseType
					}
				}

				field := Field{
					Name:     propName,
					Type:     propType,
					JSONTag:  propName,
					Nullable: propRef.Value.Nullable,
				}
				t.Fields = append(t.Fields, field)
			}
		}
	}

	return t
}
