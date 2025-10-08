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

// ProtoDescriptions maps service name -> request field name -> description.
type ProtoDescriptions map[string]map[string]string

func main() {
	input := flag.String("input", "", "Input OpenAPI YAML")
	output := flag.String("output", "", "Output OpenAPI YAML")
	protoPath := flag.String("proto-path", ".xatu-cbt/pkg/proto/clickhouse", "Path to proto files")
	flag.Parse()

	// Load proto descriptions
	descriptions, err := loadProtoDescriptions(*protoPath)
	if err != nil {
		fmt.Printf("Warning: Could not load proto descriptions: %v\n", err)

		descriptions = make(ProtoDescriptions)
	}

	fmt.Printf("Loaded field descriptions from proto files (%d services)\n", len(descriptions))

	// Load OpenAPI spec
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(*input)
	if err != nil {
		fmt.Printf("Error loading spec: %v\n", err)
		os.Exit(1)
	}

	// Process each path
	totalConverted := 0

	for path, pathItem := range doc.Paths.Map() {
		for _, op := range []*openapi3.Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Delete,
		} {
			if op == nil {
				continue
			}

			totalConverted += convertDotToUnderscore(op, path, descriptions)
		}
	}

	fmt.Printf("Converted %d parameters from dot notation to underscore notation\n", totalConverted)

	// Write output as YAML (not JSON)
	data, err := doc.MarshalJSON()
	if err != nil {
		fmt.Printf("Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}

	// Convert JSON to YAML for proper .yaml output
	var jsonData interface{}

	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err)
		os.Exit(1)
	}

	yamlData, err := yaml.Marshal(jsonData)
	if err != nil {
		fmt.Printf("Error marshaling to YAML: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(*output, yamlData, 0600)
	if err != nil {
		fmt.Printf("Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Wrote flattened OpenAPI to %s (YAML format)\n", *output)
}

// convertDotToUnderscore converts dot-notation parameters to underscore notation.
// e.g., slotStartDateTime.eq -> slot_start_date_time_eq.
func convertDotToUnderscore(op *openapi3.Operation, path string, descriptions ProtoDescriptions) int {
	// Extract service name from operation ID (e.g., "FctAttestationService_List")
	// Use lowercase for case-insensitive lookup (handles protoc-gen-openapi casing variations)
	serviceName := strings.ToLower(extractServiceName(op.OperationID))
	converted := 0

	for _, paramRef := range op.Parameters {
		if paramRef.Value == nil {
			continue
		}

		param := paramRef.Value

		// Check if parameter uses dot notation
		if strings.Contains(param.Name, ".") {
			parts := strings.Split(param.Name, ".")

			// Convert camelCase to snake_case
			snakeParts := make([]string, len(parts))
			for i, part := range parts {
				snakeParts[i] = camelToSnake(part)
			}

			// Join with underscores
			newName := strings.Join(snakeParts, "_")

			// Get field description from proto
			fieldName := parts[0] // e.g., "slotStartDateTime"
			operator := strings.Join(parts[1:], "_")

			// Convert field name to snake_case to match proto field names
			fieldNameSnake := camelToSnake(fieldName)

			if desc, ok := descriptions[serviceName]; ok {
				if fieldDesc, ok := desc[fieldNameSnake]; ok {
					// Build description with field context and operator
					param.Description = fmt.Sprintf("%s (filter: %s)", fieldDesc, operator)
				} else {
					param.Description = fmt.Sprintf("Filter %s using %s", fieldNameSnake, operator)
				}
			} else {
				param.Description = fmt.Sprintf("Filter %s using %s", fieldNameSnake, operator)
			}

			param.Name = newName

			// Convert array parameters to comma-separated strings for better compatibility
			if strings.HasSuffix(newName, "_in_values") || strings.HasSuffix(newName, "_not_in_values") {
				if param.Schema != nil && param.Schema.Value != nil {
					// Check if this is an array type
					if param.Schema.Value.Type != nil && len(param.Schema.Value.Type.Slice()) > 0 && param.Schema.Value.Type.Slice()[0] == "array" {
						// Get the item type to determine the pattern
						itemType := ""
						itemFormat := ""

						if param.Schema.Value.Items != nil && param.Schema.Value.Items.Value != nil {
							if param.Schema.Value.Items.Value.Type != nil && len(param.Schema.Value.Items.Value.Type.Slice()) > 0 {
								itemType = param.Schema.Value.Items.Value.Type.Slice()[0]
							}

							itemFormat = param.Schema.Value.Items.Value.Format
						}

						// Convert to string with appropriate pattern
						param.Schema.Value.Type = &openapi3.Types{"string"}
						param.Schema.Value.Items = nil

						// Set pattern based on item type
						pattern := getPatternForArrayType(itemType, itemFormat)
						if pattern != "" {
							param.Schema.Value.Pattern = pattern
						}

						// Update description
						originalDesc := param.Description
						param.Description = fmt.Sprintf("%s (comma-separated list)", originalDesc)
					}
				}
			}

			converted++
		}
	}

	return converted
}

// getPatternForArrayType returns the regex pattern for comma-separated values based on type.
func getPatternForArrayType(itemType, itemFormat string) string {
	switch itemType {
	case "integer":
		// For integer types, match comma-separated digits (optionally negative)
		if itemFormat == "uint32" || itemFormat == "uint64" {
			return `^\d+(,\d+)*$` // Only positive integers
		}

		return `^-?\d+(,-?\d+)*$` // Allow negative integers
	case "number":
		return `^-?\d+(\.\d+)?(,-?\d+(\.\d+)?)*$` // Floating point numbers
	case "string":
		// Match comma-separated non-empty strings (escaped commas allowed within quotes if needed)
		return `^[^,]+(,[^,]+)*$`
	default:
		return "" // No pattern for other types
	}
}

// extractServiceName extracts service name from operation ID.
// e.g., "FctAttestationService_List" -> "FctAttestationService".
func extractServiceName(operationID string) string {
	parts := strings.Split(operationID, "_")
	if len(parts) > 0 {
		return parts[0]
	}

	return ""
}

// camelToSnake converts camelCase to snake_case.
func camelToSnake(s string) string {
	// Handle empty string
	if s == "" {
		return ""
	}

	// Insert underscores before uppercase letters (except at start)
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous char is lowercase or next char is lowercase
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

// loadProtoDescriptions loads field descriptions from proto files.
func loadProtoDescriptions(protoPath string) (ProtoDescriptions, error) {
	descriptions := make(ProtoDescriptions)

	// Find all .proto files
	files, err := filepath.Glob(filepath.Join(protoPath, "*.proto"))
	if err != nil {
		return nil, err
	}

	// Parse each proto file
	for _, file := range files {
		if err := parseProtoFile(file, descriptions); err != nil {
			fmt.Printf("Warning: Error parsing %s: %v\n", file, err)
		}
	}

	return descriptions, nil
}

// parseProtoFile extracts field descriptions from a proto file.
func parseProtoFile(filename string, descriptions ProtoDescriptions) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Regex patterns
	servicePattern := regexp.MustCompile(`service\s+(\w+)`)
	messagePattern := regexp.MustCompile(`^message\s+(List\w+Request|Get\w+Request)`)
	fieldCommentPattern := regexp.MustCompile(`^\s*//\s*(.+)`)
	fieldPattern := regexp.MustCompile(`^\s*(?:\w+)\s+(\w+)\s+=\s+\d+`)

	var (
		foundService   string
		currentMessage string
		lastComment    string
	)

	// First pass: find the service name
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if matches := servicePattern.FindSubmatch(content); matches != nil {
		foundService = string(matches[1])
	}

	if foundService == "" {
		// No service found in this file
		return nil
	}

	// Initialize the service in descriptions (lowercase for case-insensitive lookup)
	serviceNameLower := strings.ToLower(foundService)
	if _, ok := descriptions[serviceNameLower]; !ok {
		descriptions[serviceNameLower] = make(map[string]string)
	}

	// Second pass: parse messages and fields
	for scanner.Scan() {
		line := scanner.Text()

		// Check for request message definition
		if matches := messagePattern.FindStringSubmatch(line); matches != nil {
			currentMessage = matches[1]

			continue
		}

		// Check for comment line
		if matches := fieldCommentPattern.FindStringSubmatch(line); matches != nil {
			comment := strings.TrimSpace(matches[1])
			// Only process comments that start with "Filter by"
			if strings.HasPrefix(comment, "Filter by ") {
				comment = strings.TrimPrefix(comment, "Filter by ")
				// Split on " - " and take the description part
				if parts := strings.SplitN(comment, " - ", 2); len(parts) == 2 {
					comment = parts[1]
					// Remove trailing notes like "(PRIMARY KEY - required)"
					comment = regexp.MustCompile(`\s*\([^)]+\)\s*$`).ReplaceAllString(comment, "")
				}

				lastComment = comment
			}

			continue
		}

		// Check for field definition
		if currentMessage != "" && lastComment != "" {
			if matches := fieldPattern.FindStringSubmatch(line); matches != nil {
				fieldName := matches[1]
				descriptions[serviceNameLower][fieldName] = lastComment
				lastComment = ""

				continue
			}
		}

		// Reset comment if line is not a continuation
		if !strings.HasPrefix(strings.TrimSpace(line), "//") && !strings.HasPrefix(strings.TrimSpace(line), "*") {
			lastComment = ""
		}
	}

	return scanner.Err()
}
