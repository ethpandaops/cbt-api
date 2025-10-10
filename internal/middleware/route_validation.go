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

// RouteValidation returns a middleware that validates routes and query parameters
// against the OpenAPI specification, returning:
// - 404 Not Found for unknown paths/methods
// - 400 Bad Request for unknown query parameters
func RouteValidation(logger logrus.FieldLogger) func(http.Handler) http.Handler {
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
				// Path not found in spec - return 404
				status := apierrors.NotFoundf("path not found: %s", r.URL.Path)
				status.WriteJSON(w)

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
				// Method not allowed for this path - return 404
				status := apierrors.NotFoundf(
					"method %s not allowed for path: %s",
					r.Method,
					r.URL.Path,
				)
				status.WriteJSON(w)

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
