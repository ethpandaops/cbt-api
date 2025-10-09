package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// ============================================================================
// Type Definitions
// ============================================================================

// ProtoDescriptions maps service name -> field name -> description.
type ProtoDescriptions map[string]map[string]string

// ProtoFieldTypes maps field name -> google.protobuf wrapper type (e.g., "DoubleValue").
type ProtoFieldTypes map[string]string

// WrapperTypeMapping defines correct OpenAPI type/format for google.protobuf wrapper types.
type WrapperTypeMapping struct {
	Type   string
	Format string
}

// ============================================================================
// Constants
// ============================================================================

const (
	colorGreen = "\033[0;32m"
	colorReset = "\033[0m"
)

// Mapping of google.protobuf wrapper types to correct OpenAPI type/format.
// protoc-gen-openapi generates incorrect mappings, causing oapi-codegen to
// generate wrong Go types that break ClickHouse scanning.
var wrapperTypeMappings = map[string]WrapperTypeMapping{
	"DoubleValue": {Type: "number", Format: "double"},  // *float64
	"FloatValue":  {Type: "number", Format: "float"},   // *float32
	"Int32Value":  {Type: "integer", Format: "int32"},  // *int32
	"Int64Value":  {Type: "integer", Format: "int64"},  // *int64
	"UInt32Value": {Type: "integer", Format: "uint32"}, // *uint32
	"UInt64Value": {Type: "integer", Format: "uint64"}, // *uint64
	"BoolValue":   {Type: "boolean", Format: ""},       // *bool
	"StringValue": {Type: "string", Format: ""},        // *string
	"BytesValue":  {Type: "string", Format: "byte"},    // []byte
}

// Regex patterns for parsing proto files.
var (
	servicePattern      = regexp.MustCompile(`service\s+(\w+)`)
	messagePattern      = regexp.MustCompile(`^message\s+(List\w+Request|Get\w+Request|Fct\w+|Dim\w+)`)
	fieldCommentPattern = regexp.MustCompile(`^\s*//\s*(.+)`)
	fieldPattern        = regexp.MustCompile(`^\s*(?:\w+)\s+(\w+)\s+=\s+\d+`)
	wrapperFieldPattern = regexp.MustCompile(`^\s*google\.protobuf\.(\w+)\s+(\w+)\s+=\s+\d+`)
)

// ============================================================================
// Main
// ============================================================================

func main() {
	input := flag.String("input", "", "Input OpenAPI YAML")
	output := flag.String("output", "", "Output OpenAPI YAML")
	protoPath := flag.String("proto-path", ".xatu-cbt/pkg/proto/clickhouse", "Path to proto files")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Println("Error: --input and --output are required")
		flag.Usage()
		os.Exit(1)
	}

	// Load proto data
	descriptions, fieldTypes, err := loadProtoData(*protoPath)
	if err != nil {
		fmt.Printf("Warning: Could not load proto data: %v\n", err)

		descriptions = make(ProtoDescriptions)
		fieldTypes = make(ProtoFieldTypes)
	}

	// Load OpenAPI spec
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(*input)
	if err != nil {
		fmt.Printf("Error loading OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Apply transformations
	_ = applyTransformations(doc, descriptions, fieldTypes)

	// Write output
	if err := writeOpenAPIYAML(doc, *output); err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Wrote processed OpenAPI to %s%s\n", colorGreen, *output, colorReset)
}

// ============================================================================
// Orchestration
// ============================================================================

// TransformationStats tracks what was changed.
type TransformationStats struct {
	FiltersFlatted int
	SchemasFixed   int
	TypesFixed     int
}

// applyTransformations applies all OpenAPI transformations.
func applyTransformations(doc *openapi3.T, descriptions ProtoDescriptions, fieldTypes ProtoFieldTypes) TransformationStats {
	stats := TransformationStats{}

	// 1. Flatten filter parameters (dot notation -> underscore notation)
	stats.FiltersFlatted = flattenFilterParameters(doc, descriptions)

	// 2. Fix schema names (e.g., "50ms" -> "50Ms")
	stats.SchemasFixed = fixSchemaNames(doc)

	// 3. Fix wrapper type mappings
	stats.TypesFixed = fixWrapperTypes(doc, fieldTypes)

	return stats
}

// ============================================================================
// Proto Parsing
// ============================================================================

// loadProtoData loads descriptions and field types from proto files.
func loadProtoData(protoPath string) (ProtoDescriptions, ProtoFieldTypes, error) {
	descriptions := make(ProtoDescriptions)
	fieldTypes := make(ProtoFieldTypes)

	files, err := filepath.Glob(filepath.Join(protoPath, "*.proto"))
	if err != nil {
		return nil, nil, err
	}

	for _, file := range files {
		if err := parseProtoFile(file, descriptions, fieldTypes); err != nil {
			fmt.Printf("Warning: Error parsing %s: %v\n", file, err)
		}
	}

	return descriptions, fieldTypes, nil
}

// parseProtoFile extracts field descriptions and wrapper types from a proto file.
func parseProtoFile(filename string, descriptions ProtoDescriptions, fieldTypes ProtoFieldTypes) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Extract service name
	serviceName := extractServiceName(content)
	if serviceName == "" {
		return nil // No service in this file
	}

	serviceNameLower := strings.ToLower(serviceName)
	if _, ok := descriptions[serviceNameLower]; !ok {
		descriptions[serviceNameLower] = make(map[string]string)
	}

	// Parse file line by line
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var currentMessage, lastComment string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for message definition
		if matches := messagePattern.FindStringSubmatch(line); matches != nil {
			currentMessage = matches[1]

			continue
		}

		// Skip if not in a message
		if currentMessage == "" {
			continue
		}

		// Check for wrapper type field
		if matches := wrapperFieldPattern.FindStringSubmatch(line); matches != nil {
			wrapperType := matches[1] // e.g., "DoubleValue"
			fieldName := matches[2]   // e.g., "meta_client_geo_longitude"
			fieldTypes[fieldName] = wrapperType

			continue
		}

		// Check for comment
		if matches := fieldCommentPattern.FindStringSubmatch(line); matches != nil {
			lastComment = extractFieldDescription(matches[1])

			continue
		}

		// Check for field definition
		if lastComment != "" {
			if matches := fieldPattern.FindStringSubmatch(line); matches != nil {
				fieldName := matches[1]
				descriptions[serviceNameLower][fieldName] = lastComment
				lastComment = ""

				continue
			}
		}

		// Reset comment if not a comment/field line
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			lastComment = ""
		}
	}

	return scanner.Err()
}

// extractServiceName finds the service name in proto file content.
func extractServiceName(content []byte) string {
	if matches := servicePattern.FindSubmatch(content); matches != nil {
		return string(matches[1])
	}

	return ""
}

// extractFieldDescription cleans up field description from comment.
func extractFieldDescription(comment string) string {
	comment = strings.TrimSpace(comment)

	// Only process "Filter by" comments
	if !strings.HasPrefix(comment, "Filter by ") {
		return ""
	}

	comment = strings.TrimPrefix(comment, "Filter by ")

	// Split on " - " and take description part
	if parts := strings.SplitN(comment, " - ", 2); len(parts) == 2 {
		comment = parts[1]
	}

	// Remove trailing notes like "(PRIMARY KEY - required)"
	comment = regexp.MustCompile(`\s*\([^)]+\)\s*$`).ReplaceAllString(comment, "")

	return comment
}

// ============================================================================
// OpenAPI Transformations
// ============================================================================

// flattenFilterParameters converts dot notation to underscore notation.
// Example: slotStartDateTime.eq -> slot_start_date_time_eq.
func flattenFilterParameters(doc *openapi3.T, descriptions ProtoDescriptions) int {
	converted := 0

	for path, pathItem := range doc.Paths.Map() {
		for _, op := range []*openapi3.Operation{pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Delete} {
			if op == nil {
				continue
			}

			converted += convertOperationParameters(op, path, descriptions)
		}
	}

	return converted
}

// convertOperationParameters processes parameters in a single operation.
func convertOperationParameters(op *openapi3.Operation, path string, descriptions ProtoDescriptions) int {
	serviceName := strings.ToLower(extractServiceNameFromOperationID(op.OperationID))
	converted := 0

	for _, paramRef := range op.Parameters {
		if paramRef.Value == nil || !strings.Contains(paramRef.Value.Name, ".") {
			continue
		}

		param := paramRef.Value
		parts := strings.Split(param.Name, ".")

		// Convert to snake_case
		snakeParts := make([]string, len(parts))
		for i, part := range parts {
			snakeParts[i] = camelToSnake(part)
		}

		newName := strings.Join(snakeParts, "_")
		fieldName := camelToSnake(parts[0])
		operator := strings.Join(parts[1:], "_")

		// Set description
		if desc, ok := descriptions[serviceName]; ok {
			if fieldDesc, ok := desc[fieldName]; ok {
				param.Description = fmt.Sprintf("%s (filter: %s)", fieldDesc, operator)
			} else {
				param.Description = fmt.Sprintf("Filter %s using %s", fieldName, operator)
			}
		} else {
			param.Description = fmt.Sprintf("Filter %s using %s", fieldName, operator)
		}

		param.Name = newName

		// Convert array parameters to comma-separated strings
		if strings.HasSuffix(newName, "_in_values") || strings.HasSuffix(newName, "_not_in_values") {
			convertArrayParamToString(param)
		}

		converted++
	}

	return converted
}

// convertArrayParamToString converts array parameter to comma-separated string.
func convertArrayParamToString(param *openapi3.Parameter) {
	if param.Schema == nil || param.Schema.Value == nil {
		return
	}

	schema := param.Schema.Value
	if schema.Type == nil || len(schema.Type.Slice()) == 0 || schema.Type.Slice()[0] != "array" {
		return
	}

	// Get item type for pattern
	itemType := ""
	itemFormat := ""

	if schema.Items != nil && schema.Items.Value != nil {
		if schema.Items.Value.Type != nil && len(schema.Items.Value.Type.Slice()) > 0 {
			itemType = schema.Items.Value.Type.Slice()[0]
		}

		itemFormat = schema.Items.Value.Format
	}

	// Convert to string with pattern
	schema.Type = &openapi3.Types{"string"}
	schema.Items = nil

	if pattern := getArrayItemPattern(itemType, itemFormat); pattern != "" {
		schema.Pattern = pattern
	}

	// Update description
	param.Description = fmt.Sprintf("%s (comma-separated list)", param.Description)
}

// getArrayItemPattern returns regex pattern for comma-separated values.
func getArrayItemPattern(itemType, itemFormat string) string {
	switch itemType {
	case "integer":
		if itemFormat == "uint32" || itemFormat == "uint64" {
			return `^\d+(,\d+)*$`
		}

		return `^-?\d+(,-?\d+)*$`
	case "number":
		return `^-?\d+(\.\d+)?(,-?\d+(\.\d+)?)*$`
	case "string":
		return `^[^,]+(,[^,]+)*$`
	default:
		return ""
	}
}

// fixSchemaNames fixes capitalization in schema names (e.g., "50ms" -> "50Ms").
func fixSchemaNames(doc *openapi3.T) int {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return 0
	}

	renamedSchemas := make(map[string]string)
	newSchemas := make(openapi3.Schemas)

	// Build rename mapping and new schemas
	for name, schemaRef := range doc.Components.Schemas {
		newName := fixCapitalization(name)
		if newName != name {
			renamedSchemas[name] = newName
		}

		newSchemas[newName] = schemaRef
	}

	doc.Components.Schemas = newSchemas

	// Update all references
	if len(renamedSchemas) > 0 {
		updateAllSchemaReferences(doc, renamedSchemas)
	}

	return len(renamedSchemas)
}

// fixCapitalization capitalizes letters after digits (e.g., "50ms" -> "50Ms").
func fixCapitalization(s string) string {
	result := []rune(s)
	for i := 0; i < len(result)-1; i++ {
		if result[i] >= '0' && result[i] <= '9' {
			if i+1 < len(result) && result[i+1] >= 'a' && result[i+1] <= 'z' {
				result[i+1] = result[i+1] - 32 // Convert to uppercase
			}
		}
	}

	return string(result)
}

// fixWrapperTypes corrects type/format for fields using google.protobuf wrapper types.
func fixWrapperTypes(doc *openapi3.T, fieldTypes ProtoFieldTypes) int {
	if doc.Components == nil || doc.Components.Schemas == nil {
		return 0
	}

	fixed := 0

	for _, schemaRef := range doc.Components.Schemas {
		if schemaRef.Value == nil {
			continue
		}

		for propName, propRef := range schemaRef.Value.Properties {
			if propRef.Value == nil {
				continue
			}

			// Check if we have wrapper type info for this field
			wrapperType, ok := fieldTypes[propName]
			if !ok {
				continue
			}

			// Get correct mapping
			correctMapping, ok := wrapperTypeMappings[wrapperType]
			if !ok {
				continue
			}

			// Apply fix if needed
			if needsTypeUpdate(propRef.Value, correctMapping) {
				propRef.Value.Type = &openapi3.Types{correctMapping.Type}
				propRef.Value.Format = correctMapping.Format
				fixed++
			}
		}
	}

	return fixed
}

// needsTypeUpdate checks if property needs type/format update.
func needsTypeUpdate(schema *openapi3.Schema, correctMapping WrapperTypeMapping) bool {
	// Check type
	if schema.Type != nil && len(schema.Type.Slice()) > 0 {
		if schema.Type.Slice()[0] != correctMapping.Type {
			return true
		}
	}

	// Check format
	if schema.Format != correctMapping.Format {
		return true
	}

	return false
}

// ============================================================================
// Schema Reference Updates
// ============================================================================

// updateAllSchemaReferences updates $ref values throughout the document.
func updateAllSchemaReferences(doc *openapi3.T, renamedSchemas map[string]string) {
	// Update in paths
	for _, pathItem := range doc.Paths.Map() {
		updateOperationRefs(pathItem.Get, renamedSchemas)
		updateOperationRefs(pathItem.Post, renamedSchemas)
		updateOperationRefs(pathItem.Put, renamedSchemas)
		updateOperationRefs(pathItem.Delete, renamedSchemas)
		updateOperationRefs(pathItem.Patch, renamedSchemas)
		updateOperationRefs(pathItem.Head, renamedSchemas)
		updateOperationRefs(pathItem.Options, renamedSchemas)
	}

	// Update in components
	if doc.Components != nil {
		for _, respRef := range doc.Components.Responses {
			updateResponseRef(respRef, renamedSchemas)
		}

		for _, paramRef := range doc.Components.Parameters {
			updateParameterRef(paramRef, renamedSchemas)
		}

		for _, schemaRef := range doc.Components.Schemas {
			updateSchemaRef(schemaRef, renamedSchemas)
		}
	}
}

// updateOperationRefs updates references in an operation.
func updateOperationRefs(op *openapi3.Operation, renamedSchemas map[string]string) {
	if op == nil {
		return
	}

	for _, paramRef := range op.Parameters {
		updateParameterRef(paramRef, renamedSchemas)
	}

	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, mediaType := range op.RequestBody.Value.Content {
			if mediaType.Schema != nil {
				updateSchemaRef(mediaType.Schema, renamedSchemas)
			}
		}
	}

	for _, respRef := range op.Responses.Map() {
		updateResponseRef(respRef, renamedSchemas)
	}
}

// updateResponseRef updates references in a response.
func updateResponseRef(respRef *openapi3.ResponseRef, renamedSchemas map[string]string) {
	if respRef == nil || respRef.Value == nil {
		return
	}

	for _, mediaType := range respRef.Value.Content {
		if mediaType.Schema != nil {
			updateSchemaRef(mediaType.Schema, renamedSchemas)
		}
	}
}

// updateParameterRef updates references in a parameter.
func updateParameterRef(paramRef *openapi3.ParameterRef, renamedSchemas map[string]string) {
	if paramRef == nil || paramRef.Value == nil {
		return
	}

	if paramRef.Value.Schema != nil {
		updateSchemaRef(paramRef.Value.Schema, renamedSchemas)
	}
}

// updateSchemaRef recursively updates references in a schema.
func updateSchemaRef(schemaRef *openapi3.SchemaRef, renamedSchemas map[string]string) {
	if schemaRef == nil {
		return
	}

	// Update $ref
	if schemaRef.Ref != "" {
		for oldName, newName := range renamedSchemas {
			oldRef := "#/components/schemas/" + oldName

			newRef := "#/components/schemas/" + newName
			if schemaRef.Ref == oldRef {
				schemaRef.Ref = newRef

				break
			}
		}
	}

	// Recursively update nested schemas
	if schemaRef.Value != nil {
		for _, propRef := range schemaRef.Value.Properties {
			updateSchemaRef(propRef, renamedSchemas)
		}

		if schemaRef.Value.Items != nil {
			updateSchemaRef(schemaRef.Value.Items, renamedSchemas)
		}

		for _, s := range schemaRef.Value.AllOf {
			updateSchemaRef(s, renamedSchemas)
		}

		for _, s := range schemaRef.Value.AnyOf {
			updateSchemaRef(s, renamedSchemas)
		}

		for _, s := range schemaRef.Value.OneOf {
			updateSchemaRef(s, renamedSchemas)
		}

		if schemaRef.Value.Not != nil {
			updateSchemaRef(schemaRef.Value.Not, renamedSchemas)
		}

		if schemaRef.Value.AdditionalProperties.Schema != nil {
			updateSchemaRef(schemaRef.Value.AdditionalProperties.Schema, renamedSchemas)
		}
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

// extractServiceNameFromOperationID extracts service name from operation ID.
// Example: "FctAttestationService_List" -> "FctAttestationService".
func extractServiceNameFromOperationID(operationID string) string {
	parts := strings.Split(operationID, "_")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// camelToSnake converts camelCase to snake_case.
func camelToSnake(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prevIsLower := i > 0 && s[i-1] >= 'a' && s[i-1] <= 'z'
			nextIsLower := i < len(s)-1 && s[i+1] >= 'a' && s[i+1] <= 'z'

			if prevIsLower || nextIsLower {
				result.WriteRune('_')
			}
		}

		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// writeOpenAPIYAML writes OpenAPI document to YAML file.
func writeOpenAPIYAML(doc *openapi3.T, filename string) error {
	// Marshal to JSON first
	data, err := doc.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal to JSON: %w", err)
	}

	// Convert to generic map
	var jsonData interface{}

	if unmarshalErr := json.Unmarshal(data, &jsonData); unmarshalErr != nil {
		return fmt.Errorf("unmarshal JSON: %w", unmarshalErr)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("marshal to YAML: %w", err)
	}

	// Write file
	if err := os.WriteFile(filename, yamlData, 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
