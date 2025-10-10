package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ethpandaops/xatu-cbt-api/internal/handlers"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/sirupsen/logrus"
)

// QueryParameterValidation returns a middleware that validates query parameters
// against the OpenAPI specification, returning 400 Bad Request for unknown parameters.
func QueryParameterValidation(logger logrus.FieldLogger) func(http.Handler) http.Handler {
	// Load OpenAPI spec once at initialization
	swagger, err := handlers.GetSwagger()
	if err != nil {
		logger.WithError(err).Fatal("failed to load OpenAPI specification")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Find the operation for this path and method
			pathItem := swagger.Paths.Find(r.URL.Path)
			if pathItem == nil {
				// Path not found in spec, let it pass through
				// (will be handled by 404 handler)
				next.ServeHTTP(w, r)

				return
			}

			var operation *openapi3.Operation

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

			if operation == nil {
				// Method not found in spec, let it pass through
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

				// Build list of valid parameters for error message
				validParamList := make([]string, 0, len(validParams))
				for param := range validParams {
					validParamList = append(validParamList, param)
				}

				sort.Strings(validParamList)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				errorMsg := fmt.Sprintf(
					`{"error":"unknown query parameter(s): %s","valid_parameters":["%s"]}`,
					strings.Join(unknownParams, ", "),
					strings.Join(validParamList, `","`),
				)

				_, _ = w.Write([]byte(errorMsg))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
