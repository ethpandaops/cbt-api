package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecovery(t *testing.T) {
	tests := []struct {
		name               string
		handler            http.HandlerFunc
		expectPanic        bool
		expectStatusCode   int
		expectErrorMessage string
		expectLogFields    []string
	}{
		{
			name: "recovers from panic with string",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic("something went wrong")
			},
			expectPanic:        true,
			expectStatusCode:   http.StatusInternalServerError,
			expectErrorMessage: "Internal Server Error",
			expectLogFields:    []string{"error", "path", "stack"},
		},
		{
			name: "recovers from panic with error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(http.ErrAbortHandler)
			},
			expectPanic:        true,
			expectStatusCode:   http.StatusInternalServerError,
			expectErrorMessage: "Internal Server Error",
			expectLogFields:    []string{"error", "path", "stack"},
		},
		{
			name: "recovers from panic with custom struct",
			handler: func(w http.ResponseWriter, r *http.Request) {
				panic(struct{ msg string }{"custom panic"})
			},
			expectPanic:        true,
			expectStatusCode:   http.StatusInternalServerError,
			expectErrorMessage: "Internal Server Error",
			expectLogFields:    []string{"error", "path", "stack"},
		},
		{
			name: "does not interfere with normal requests",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("success"))
			},
			expectPanic:      false,
			expectStatusCode: http.StatusOK,
		},
		{
			name: "handles panic after partial response write",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("partial"))
				panic("panic after write")
			},
			expectPanic:        true,
			expectStatusCode:   http.StatusOK, // Status already written
			expectErrorMessage: "",            // Can't change response after write
			expectLogFields:    []string{"error", "path", "stack"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logger with buffer to capture logs
			var logBuf bytes.Buffer

			logger := logrus.New()
			logger.SetOutput(&logBuf)
			logger.SetFormatter(&logrus.JSONFormatter{})

			// Create middleware
			middleware := Recovery(logger)
			handler := middleware(tt.handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
			rec := httptest.NewRecorder()

			// Execute handler
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, tt.expectStatusCode, rec.Code)

			if tt.expectPanic {
				// Verify log entry was created
				logOutput := logBuf.String()
				assert.NotEmpty(t, logOutput)

				// Parse log entry
				var logEntry map[string]interface{}

				err := json.Unmarshal([]byte(logOutput), &logEntry)
				require.NoError(t, err)

				// Verify log fields
				for _, field := range tt.expectLogFields {
					assert.Contains(t, logEntry, field, "Expected log field %s", field)
				}

				// Verify path is logged correctly
				assert.Equal(t, "/test/path", logEntry["path"])

				// Verify log level is error
				assert.Equal(t, "error", logEntry["level"])

				// Verify log message
				assert.Equal(t, "panic recovered", logEntry["msg"])

				// Check response body contains error message if status was 500
				if tt.expectStatusCode == http.StatusInternalServerError {
					body := rec.Body.String()
					if tt.expectErrorMessage != "" {
						assert.Contains(t, body, tt.expectErrorMessage)
					}
					// Verify Content-Type header
					assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
				}
			} else {
				// No panic should mean no error logs
				logOutput := logBuf.String()
				assert.Empty(t, logOutput)
			}
		})
	}
}

func TestRecovery_StackTraceIncluded(t *testing.T) {
	var logBuf bytes.Buffer

	logger := logrus.New()
	logger.SetOutput(&logBuf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	middleware := Recovery(logger)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Parse log entry
	var logEntry map[string]interface{}

	err := json.Unmarshal(logBuf.Bytes(), &logEntry)
	require.NoError(t, err)

	// Verify stack trace is present and contains useful information
	stack, ok := logEntry["stack"].(string)
	require.True(t, ok, "Stack trace should be a string")
	assert.NotEmpty(t, stack)
	assert.Contains(t, stack, "panic", "Stack trace should contain panic info")
	assert.Contains(t, stack, "goroutine", "Stack trace should contain goroutine info")
}

func TestRecovery_DifferentPaths(t *testing.T) {
	paths := []string{
		"/",
		"/api/v1/test",
		"/users/123/profile",
		"/api/v2/blocks?limit=10",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			var logBuf bytes.Buffer

			logger := logrus.New()
			logger.SetOutput(&logBuf)
			logger.SetFormatter(&logrus.JSONFormatter{})

			middleware := Recovery(logger)
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("test panic")
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Parse log entry
			var logEntry map[string]interface{}

			err := json.Unmarshal(logBuf.Bytes(), &logEntry)
			require.NoError(t, err)

			// Verify correct path is logged
			loggedPath, ok := logEntry["path"].(string)
			require.True(t, ok)
			assert.True(t, strings.HasPrefix(path, loggedPath), "Path %s should start with %s", path, loggedPath)
		})
	}
}

func TestRecovery_NilPanic(t *testing.T) {
	var logBuf bytes.Buffer

	logger := logrus.New()
	logger.SetOutput(&logBuf)
	logger.SetFormatter(&logrus.JSONFormatter{})

	middleware := Recovery(logger)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(nil)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should handle nil panic gracefully
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify log was created
	logOutput := logBuf.String()
	assert.NotEmpty(t, logOutput)
}

func TestRecovery_ResponseFormat(t *testing.T) {
	var logBuf bytes.Buffer

	logger := logrus.New()
	logger.SetOutput(&logBuf)

	middleware := Recovery(logger)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test error")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify Content-Type
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	// Parse JSON response
	var response map[string]interface{}

	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Equal(t, "Internal Server Error", response["error"])
	assert.Equal(t, "An unexpected error occurred", response["message"])
	assert.Equal(t, float64(500), response["code"])
}
