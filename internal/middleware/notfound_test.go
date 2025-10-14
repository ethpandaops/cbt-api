package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apierrors "github.com/ethpandaops/cbt-api/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestNotFoundHandler(t *testing.T) {
	tests := []struct {
		name           string
		handlerStatus  int
		handlerBody    string
		expectedStatus int
		expectedBody   string
		checkJSON      bool
	}{
		{
			name:           "404 is converted to Status format",
			handlerStatus:  http.StatusNotFound,
			handlerBody:    "404 page not found\n",
			expectedStatus: http.StatusNotFound,
			checkJSON:      true,
		},
		{
			name:           "200 passes through unchanged",
			handlerStatus:  http.StatusOK,
			handlerBody:    `{"success":true}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success":true}`,
			checkJSON:      false,
		},
		{
			name:           "500 passes through unchanged",
			handlerStatus:  http.StatusInternalServerError,
			handlerBody:    `{"error":"internal error"}`,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"internal error"}`,
			checkJSON:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that returns the specified status and body
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				_, _ = w.Write([]byte(tt.handlerBody))
			})

			// Apply middleware
			middleware := NotFoundHandler()
			handler := middleware(testHandler)

			// Make request
			req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code, "unexpected status code")

			if tt.checkJSON {
				// Verify it's proper Status JSON format
				var status apierrors.Status

				err := json.Unmarshal(rec.Body.Bytes(), &status)
				require.NoError(t, err, "should be valid JSON Status")
				assert.Equal(t, codes.NotFound, status.Code, "should have NotFound code")
				assert.Contains(t, status.Message, "path not found", "should have proper message")
			} else {
				// Check body matches expected
				assert.Equal(t, tt.expectedBody, rec.Body.String(), "unexpected body")
			}
		})
	}
}
