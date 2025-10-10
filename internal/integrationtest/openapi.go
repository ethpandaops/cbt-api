package integrationtest

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Endpoint represents a single API endpoint extracted from the OpenAPI spec.
type Endpoint struct {
	Path                string
	Method              string
	OperationID         string
	RequiredParams      []string
	RequiredGroupParams map[string][]string // group name -> param names
	TableName           string
}

// Parameter represents an OpenAPI parameter definition.
type Parameter struct {
	Name          string                 `yaml:"name"`
	In            string                 `yaml:"in"`
	Required      bool                   `yaml:"required"`
	Extensions    map[string]interface{} `yaml:",inline"` // Catches x- extensions
	RequiredGroup string                 // Populated from x-required-group
}

// PathItem represents an OpenAPI path item with operations.
type PathItem struct {
	Get    *Operation `yaml:"get"`
	Post   *Operation `yaml:"post"`
	Put    *Operation `yaml:"put"`
	Delete *Operation `yaml:"delete"`
	Patch  *Operation `yaml:"patch"`
}

// Operation represents an OpenAPI operation (GET, POST, etc).
type Operation struct {
	OperationID string      `yaml:"operationId"`
	Parameters  []Parameter `yaml:"parameters"`
}

// OpenAPISpec represents a simplified OpenAPI specification for parsing.
type OpenAPISpec struct {
	Paths map[string]PathItem `yaml:"paths"`
}

// ParseOpenAPISpec parses the OpenAPI specification file and extracts endpoint information.
// It reads the YAML file, unmarshals it, and iterates through all paths and methods to
// create Endpoint structs containing path, method, operation ID, required parameters,
// and the associated table name.
func ParseOpenAPISpec(specPath string) ([]Endpoint, error) {
	// Read the OpenAPI spec file
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAPI spec file: %w", err)
	}

	// Unmarshal the YAML into our simplified spec structure
	var spec OpenAPISpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAPI spec: %w", err)
	}

	// Extract endpoints from all paths and methods
	endpoints := make([]Endpoint, 0, len(spec.Paths)*5)

	for path, pathItem := range spec.Paths {
		// Extract table name from the path
		tableName := extractTableName(path)

		// Process each HTTP method
		if pathItem.Get != nil {
			endpoint := buildEndpoint(path, "GET", tableName, pathItem.Get)
			endpoints = append(endpoints, endpoint)
		}

		if pathItem.Post != nil {
			endpoint := buildEndpoint(path, "POST", tableName, pathItem.Post)
			endpoints = append(endpoints, endpoint)
		}

		if pathItem.Put != nil {
			endpoint := buildEndpoint(path, "PUT", tableName, pathItem.Put)
			endpoints = append(endpoints, endpoint)
		}

		if pathItem.Delete != nil {
			endpoint := buildEndpoint(path, "DELETE", tableName, pathItem.Delete)
			endpoints = append(endpoints, endpoint)
		}

		if pathItem.Patch != nil {
			endpoint := buildEndpoint(path, "PATCH", tableName, pathItem.Patch)
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints, nil
}

// buildEndpoint creates an Endpoint struct from an operation.
func buildEndpoint(path, method, tableName string, operation *Operation) Endpoint {
	// Extract required parameters and required groups
	requiredParams := make([]string, 0)
	requiredGroups := make(map[string][]string)

	for _, param := range operation.Parameters {
		if param.Required {
			requiredParams = append(requiredParams, param.Name)
		}

		// Check for x-required-group extension
		if param.Extensions != nil {
			if groupVal, ok := param.Extensions["x-required-group"]; ok {
				if groupName, ok := groupVal.(string); ok && groupName != "" {
					if requiredGroups[groupName] == nil {
						requiredGroups[groupName] = make([]string, 0)
					}

					requiredGroups[groupName] = append(requiredGroups[groupName], param.Name)
				}
			}
		}
	}

	return Endpoint{
		Path:                path,
		Method:              method,
		OperationID:         operation.OperationID,
		RequiredParams:      requiredParams,
		RequiredGroupParams: requiredGroups,
		TableName:           tableName,
	}
}

// extractTableName extracts the table name from an API path.
// For example, "/api/v1/fct_block" returns "fct_block".
// For paths with parameters like "/api/v1/fct_block/{id}", it still returns "fct_block".
func extractTableName(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Split by slash
	parts := strings.Split(path, "/")

	// The table name is typically the last part of the path (after /api/v1/)
	// If the path has more than 2 parts (api, v1, table_name), get the last part
	if len(parts) >= 3 {
		tablePart := parts[2]

		// Remove any path parameters (text within curly braces)
		// For example, "fct_block/{id}" becomes "fct_block"
		if idx := strings.Index(tablePart, "{"); idx != -1 {
			tablePart = tablePart[:idx]
		}

		// Trim any trailing characters
		tablePart = strings.TrimRight(tablePart, "/")

		return tablePart
	}

	// If the path doesn't match expected format, return the last non-empty part
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]

		// Skip path parameters
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}

		if part != "" {
			return part
		}
	}

	return ""
}
