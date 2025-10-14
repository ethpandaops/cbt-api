package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apierrors "github.com/ethpandaops/cbt-api/internal/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestQueryParameterValidation(t *testing.T) {
	// Create a test logger
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Create a simple test handler that always returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	tests := []struct {
		name                  string
		path                  string
		method                string
		queryParams           string
		expectedStatus        int
		expectedCode          codes.Code
		expectedMessagePrefix string
		checkMetadata         bool
	}{
		{
			name:           "valid route with valid parameter",
			path:           "/api/v1/fct_data_types_integers",
			method:         http.MethodGet,
			queryParams:    "id_gte=100",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unknown path passes through (handled by mux)",
			path:           "/api/v1/does_not_exist",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK, // Passes through, mux will handle 404
		},
		{
			name:                  "invalid query parameter",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "id_gtw=100",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "unknown query parameter(s):",
			checkMetadata:         true,
		},
		{
			name:                  "multiple invalid parameters",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "id_gtw=100&invalid_param=test",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "unknown query parameter(s):",
			checkMetadata:         true,
		},
		{
			name:                  "mixed valid and invalid parameters",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "id_gte=100&id_gtw=99999",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "unknown query parameter(s): id_gtw",
			checkMetadata:         true,
		},
		{
			name:           "valid parameter timestamp_eq",
			path:           "/api/v1/fct_data_types_integers",
			method:         http.MethodGet,
			queryParams:    "timestamp_eq=1234",
			expectedStatus: http.StatusOK,
		},
		{
			name:                  "invalid uint32 parameter value (non-numeric)",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "timestamp_eq=xasdfsdfafds",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "parameter 'timestamp_eq' must be a valid unsigned 32-bit integer",
		},
		{
			name:                  "invalid uint32 parameter value (negative)",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "timestamp_eq=-123",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "parameter 'timestamp_eq' must be a valid unsigned 32-bit integer",
		},
		{
			name:                  "invalid uint32 parameter value (too large)",
			path:                  "/api/v1/fct_data_types_integers",
			method:                http.MethodGet,
			queryParams:           "timestamp_eq=99999999999999999999",
			expectedStatus:        http.StatusBadRequest,
			expectedCode:          codes.InvalidArgument,
			expectedMessagePrefix: "parameter 'timestamp_eq' must be a valid unsigned 32-bit integer",
		},
		{
			name:           "valid uint32 parameter value",
			path:           "/api/v1/fct_data_types_integers",
			method:         http.MethodGet,
			queryParams:    "timestamp_eq=12345",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid uint32 parameter value (zero)",
			path:           "/api/v1/fct_data_types_integers",
			method:         http.MethodGet,
			queryParams:    "timestamp_eq=0",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid uint32 parameter with multiple filters",
			path:           "/api/v1/fct_data_types_integers",
			method:         http.MethodGet,
			queryParams:    "timestamp_gte=100&timestamp_lte=200",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			url := tt.path
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(tt.method, url, nil)
			rec := httptest.NewRecorder()

			// Apply middleware
			middleware := QueryParameterValidation(logger)
			handler := middleware(testHandler)

			// Execute request
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code, "unexpected status code")

			// For error responses, verify Status format
			if tt.expectedStatus != http.StatusOK {
				var status apierrors.Status

				err := json.Unmarshal(rec.Body.Bytes(), &status)
				require.NoError(t, err, "should be valid JSON Status")

				assert.Equal(t, tt.expectedCode, status.Code, "unexpected error code")
				assert.Contains(t, status.Message, tt.expectedMessagePrefix, "unexpected message")

				if tt.checkMetadata {
					require.NotEmpty(t, status.Details, "should have error details")
					require.Contains(t, status.Details[0], "@type", "should have @type field")
					assert.Equal(
						t,
						"type.googleapis.com/ErrorInfo",
						status.Details[0]["@type"],
						"should have ErrorInfo type",
					)
					require.Contains(t, status.Details[0], "metadata", "should have metadata field")
				}
			}
		})
	}
}
