package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		name           string
		path           string
		queryParams    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid parameter slot_gte",
			path:           "/api/v1/fct_block",
			queryParams:    "slot_gte=12771120",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "invalid parameter slot_gtw",
			path:           "/api/v1/fct_block",
			queryParams:    "slot_gtw=12771120",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown query parameter(s): slot_gtw",
		},
		{
			name:           "multiple invalid parameters",
			path:           "/api/v1/fct_block",
			queryParams:    "slot_gtw=12771120&invalid_param=test",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown query parameter(s):",
		},
		{
			name:           "mixed valid and invalid parameters",
			path:           "/api/v1/fct_block",
			queryParams:    "slot_gte=12771120&slot_gtw=99999",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "unknown query parameter(s): slot_gtw",
		},
		{
			name:           "valid parameter block_root_eq",
			path:           "/api/v1/fct_block",
			queryParams:    "block_root_eq=0x1234",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with query parameters
			url := tt.path
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rec := httptest.NewRecorder()

			// Apply middleware
			middleware := QueryParameterValidation(logger)
			handler := middleware(testHandler)

			// Execute request
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code, "unexpected status code")

			// Check response body contains expected text
			body := rec.Body.String()
			assert.Contains(t, body, tt.expectedBody, "unexpected response body")

			// For error responses, verify JSON structure
			if tt.expectedStatus == http.StatusBadRequest {
				require.Contains(t, body, "error", "error response should contain 'error' field")
				require.Contains(t, body, "valid_parameters", "error response should contain 'valid_parameters' field")
			}
		})
	}
}
