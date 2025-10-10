package middleware

import (
	"net/http"
	"sort"
	"strings"

	apierrors "github.com/ethpandaops/xatu-cbt-api/internal/errors"
	"github.com/ethpandaops/xatu-cbt-api/internal/handlers"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"
)

// QueryParameterValidation returns a middleware that validates query parameters
// against the OpenAPI specification, returning 400 Bad Request for unknown parameters.
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
			// This is a simple best-effort match without pulling in a full router
			var operation *openapi3.Operation

			for path, pathItem := range swagger.Paths.Map() {
				// Simple pattern matching - this won't handle all cases but covers our use
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

			// Build set of valid query parameter names
			validParams := make(map[string]bool)

			for _, param := range operation.Parameters {
				if param.Value != nil && param.Value.In == openapi3.ParameterInQuery {
					validParams[param.Value.Name] = true
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
				// Sort for consistent error messages
				sort.Strings(unknownParams)

				// Build list of valid parameters for error details
				validParamList := make([]string, 0, len(validParams))
				for param := range validParams {
					validParamList = append(validParamList, param)
				}

				sort.Strings(validParamList)

				// Create Status error with metadata
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

			next.ServeHTTP(w, r)
		})
	}
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
