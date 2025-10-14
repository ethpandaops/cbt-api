package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	apierrors "github.com/ethpandaops/cbt-api/internal/errors"
	"github.com/ethpandaops/cbt-api/internal/handlers"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"
)

// QueryParameterValidation returns a middleware that validates query parameters
// against the OpenAPI specification, returning 400 Bad Request for:
// - Unknown parameters
// - Invalid parameter types (e.g., non-numeric value for uint32)
// - Invalid parameter formats (e.g., pattern violations)
//
// Note: This middleware validates only query parameters, not routes/paths.
// Route validation is handled by the http.ServeMux itself (Go 1.22+).
func QueryParameterValidation(logger logrus.FieldLogger) func(http.Handler) http.Handler {
	// Load OpenAPI spec once at initialization
	swagger, err := handlers.GetSwagger()
	if err != nil {
		logger.WithError(err).Fatal("failed to load OpenAPI specification")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to find the OpenAPI operation by matching path patterns
			var operation *openapi3.Operation

			for path, pathItem := range swagger.Paths.Map() {
				if matchesPattern(r.URL.Path, path) {
					switch r.Method {
					case http.MethodGet:
						operation = pathItem.Get
					case http.MethodPost:
						operation = pathItem.Post
					case http.MethodPut:
						operation = pathItem.Put
					case http.MethodDelete:
						operation = pathItem.Delete
					case http.MethodPatch:
						operation = pathItem.Patch
					}

					if operation != nil {
						break
					}
				}
			}

			// If we can't find the operation, let it through - the handler will return 404
			if operation == nil {
				next.ServeHTTP(w, r)

				return
			}

			// Build map of parameter definitions for validation
			paramDefs := make(map[string]*openapi3.Parameter)
			validParams := make(map[string]bool)

			for _, paramRef := range operation.Parameters {
				if paramRef.Value != nil && paramRef.Value.In == openapi3.ParameterInQuery {
					paramDefs[paramRef.Value.Name] = paramRef.Value
					validParams[paramRef.Value.Name] = true
				}
			}

			// Check for unknown query parameters
			var unknownParams []string

			for paramName := range r.URL.Query() {
				if !validParams[paramName] {
					unknownParams = append(unknownParams, paramName)
				}
			}

			if len(unknownParams) > 0 {
				sort.Strings(unknownParams)

				validParamList := make([]string, 0, len(validParams))
				for param := range validParams {
					validParamList = append(validParamList, param)
				}

				sort.Strings(validParamList)

				status := apierrors.BadRequestf(
					"unknown query parameter(s): %s",
					strings.Join(unknownParams, ", "),
				).WithMetadata(map[string]string{
					"unknown_parameters": strings.Join(unknownParams, ", "),
					"valid_parameters":   strings.Join(validParamList, ", "),
				})

				status.WriteJSON(w)

				return
			}

			// Validate parameter types and formats
			for paramName, paramDef := range paramDefs {
				values := r.URL.Query()[paramName]
				if len(values) == 0 {
					continue // Parameter not provided, skip validation
				}

				// Validate each value
				for _, value := range values {
					if err := validateParameterValue(paramName, value, paramDef); err != nil {
						logger.WithFields(logrus.Fields{
							"param": paramName,
							"value": value,
							"path":  r.URL.Path,
						}).Warn("parameter validation failed")

						status := apierrors.BadRequest(err.Error())
						status.WriteJSON(w)

						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateParameterValue validates a single parameter value against its OpenAPI schema.
func validateParameterValue(paramName, value string, param *openapi3.Parameter) error {
	if param.Schema == nil || param.Schema.Value == nil {
		return nil // No schema to validate against
	}

	schema := param.Schema.Value

	// Get the parameter type
	if schema.Type == nil || len(schema.Type.Slice()) == 0 {
		return nil // No type specified
	}

	paramType := schema.Type.Slice()[0]
	format := schema.Format

	// Validate based on type
	switch paramType {
	case "integer":
		return validateIntegerParameter(paramName, value, format, schema)
	case "number":
		return validateNumberParameter(paramName, value, schema)
	case "string":
		return validateStringParameter(paramName, value, schema)
	case "boolean":
		return validateBooleanParameter(paramName, value)
	}

	return nil
}

// validateIntegerParameter validates integer parameters (uint32, uint64, int32, int64).
func validateIntegerParameter(paramName, value, format string, schema *openapi3.Schema) error {
	switch format {
	case "uint32":
		val, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("parameter '%s' must be a valid unsigned 32-bit integer", paramName)
		}

		// Check minimum if specified
		if schema.Min != nil && float64(val) < *schema.Min {
			return fmt.Errorf("parameter '%s' must be >= %v", paramName, *schema.Min)
		}

		// Check maximum if specified
		if schema.Max != nil && float64(val) > *schema.Max {
			return fmt.Errorf("parameter '%s' must be <= %v", paramName, *schema.Max)
		}

	case "uint64":
		val, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parameter '%s' must be a valid unsigned 64-bit integer", paramName)
		}

		if schema.Min != nil && float64(val) < *schema.Min {
			return fmt.Errorf("parameter '%s' must be >= %v", paramName, *schema.Min)
		}

		if schema.Max != nil && float64(val) > *schema.Max {
			return fmt.Errorf("parameter '%s' must be <= %v", paramName, *schema.Max)
		}

	case "int32":
		val, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("parameter '%s' must be a valid signed 32-bit integer", paramName)
		}

		if schema.Min != nil && float64(val) < *schema.Min {
			return fmt.Errorf("parameter '%s' must be >= %v", paramName, *schema.Min)
		}

		if schema.Max != nil && float64(val) > *schema.Max {
			return fmt.Errorf("parameter '%s' must be <= %v", paramName, *schema.Max)
		}

	case "int64":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parameter '%s' must be a valid signed 64-bit integer", paramName)
		}

		if schema.Min != nil && float64(val) < *schema.Min {
			return fmt.Errorf("parameter '%s' must be >= %v", paramName, *schema.Min)
		}

		if schema.Max != nil && float64(val) > *schema.Max {
			return fmt.Errorf("parameter '%s' must be <= %v", paramName, *schema.Max)
		}

	default:
		// Generic integer validation
		_, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parameter '%s' must be a valid integer", paramName)
		}
	}

	return nil
}

// validateNumberParameter validates number parameters (float, double).
func validateNumberParameter(paramName, value string, schema *openapi3.Schema) error {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("parameter '%s' must be a valid number", paramName)
	}

	if schema.Min != nil && val < *schema.Min {
		return fmt.Errorf("parameter '%s' must be >= %v", paramName, *schema.Min)
	}

	if schema.Max != nil && val > *schema.Max {
		return fmt.Errorf("parameter '%s' must be <= %v", paramName, *schema.Max)
	}

	return nil
}

// validateStringParameter validates string parameters with pattern, minLength, maxLength.
func validateStringParameter(paramName, value string, schema *openapi3.Schema) error {
	// Check pattern if specified
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil {
			return fmt.Errorf("parameter '%s' has invalid pattern in schema", paramName)
		}

		if !matched {
			return fmt.Errorf("parameter '%s' has invalid format", paramName)
		}
	}

	// Check minLength if specified
	if schema.MinLength > 0 && uint64(len(value)) < schema.MinLength {
		return fmt.Errorf("parameter '%s' must be at least %d characters", paramName, schema.MinLength)
	}

	// Check maxLength if specified
	if schema.MaxLength != nil && uint64(len(value)) > *schema.MaxLength {
		return fmt.Errorf("parameter '%s' must be at most %d characters", paramName, *schema.MaxLength)
	}

	return nil
}

// validateBooleanParameter validates boolean parameters.
func validateBooleanParameter(paramName, value string) error {
	_, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("parameter '%s' must be a valid boolean (true or false)", paramName)
	}

	return nil
}

// matchesPattern checks if a request path matches an OpenAPI path pattern.
// Example: "/api/v1/fct_block/123" matches "/api/v1/fct_block/{slot_start_date_time}".
func matchesPattern(requestPath, patternPath string) bool {
	// Split paths into segments
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	patternParts := strings.Split(strings.Trim(patternPath, "/"), "/")

	// Must have same number of segments
	if len(requestParts) != len(patternParts) {
		return false
	}

	// Match each segment
	for i := range requestParts {
		// Pattern segments starting with { are wildcards (path parameters)
		if strings.HasPrefix(patternParts[i], "{") && strings.HasSuffix(patternParts[i], "}") {
			continue
		}

		// Otherwise must be exact match
		if requestParts[i] != patternParts[i] {
			return false
		}
	}

	return true
}
