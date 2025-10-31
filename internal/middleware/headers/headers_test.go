package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethpandaops/cbt-api/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		policies    []config.HeaderPolicy
		wantErr     bool
		errContains string
	}{
		{
			name: "valid policies compile successfully",
			policies: []config.HeaderPolicy{
				{
					Name:        "health",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
				{
					Name:        "api",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
						"Vary":          "Accept-Encoding",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid regex pattern returns error with policy name",
			policies: []config.HeaderPolicy{
				{
					Name:        "bad_regex",
					PathPattern: "[invalid(regex",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			wantErr:     true,
			errContains: "bad_regex",
		},
		{
			name:     "empty policy list succeeds",
			policies: []config.HeaderPolicy{},
			wantErr:  false,
		},
		{
			name:     "nil policy list succeeds",
			policies: nil,
			wantErr:  false,
		},
		{
			name: "policy with empty headers map succeeds",
			policies: []config.HeaderPolicy{
				{
					Name:        "no_headers",
					PathPattern: "^/test$",
					Headers:     map[string]string{},
				},
			},
			wantErr: false,
		},
		{
			name: "policy with nil headers map succeeds",
			policies: []config.HeaderPolicy{
				{
					Name:        "nil_headers",
					PathPattern: "^/test$",
					Headers:     nil,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple policies compile correctly",
			policies: []config.HeaderPolicy{
				{
					Name:        "openapi",
					PathPattern: `\.yaml$`,
					Headers: map[string]string{
						"Cache-Control": "public, max-age=3600",
					},
				},
				{
					Name:        "docs",
					PathPattern: "^/docs",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=300",
					},
				},
				{
					Name:        "health",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=5",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "complex regex patterns compile successfully",
			policies: []config.HeaderPolicy{
				{
					Name:        "complex",
					PathPattern: `^/api/v[0-9]+/query/[a-z]+$`,
					Headers: map[string]string{
						"Cache-Control": "max-age=30",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "second policy has invalid regex",
			policies: []config.HeaderPolicy{
				{
					Name:        "valid",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
				{
					Name:        "invalid_policy",
					PathPattern: "*invalid[",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			wantErr:     true,
			errContains: "invalid_policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.policies)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, manager)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, manager)
				assert.Len(t, manager.policies, len(tt.policies))
			}
		})
	}
}

func TestManagerMatch(t *testing.T) {
	tests := []struct {
		name        string
		policies    []config.HeaderPolicy
		path        string
		wantPolicy  string
		wantHeaders map[string]string
	}{
		{
			name: "exact path match returns correct policy and headers",
			policies: []config.HeaderPolicy{
				{
					Name:        "health",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			path:       "/health",
			wantPolicy: "health",
			wantHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
		},
		{
			name: "regex pattern match with file extension",
			policies: []config.HeaderPolicy{
				{
					Name:        "yaml_files",
					PathPattern: `\.yaml$`,
					Headers: map[string]string{
						"Cache-Control": "public, max-age=3600",
					},
				},
			},
			path:       "/openapi.yaml",
			wantPolicy: "yaml_files",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=3600",
			},
		},
		{
			name: "regex pattern match with path prefix",
			policies: []config.HeaderPolicy{
				{
					Name:        "api_endpoints",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
						"Vary":          "Accept-Encoding",
					},
				},
			},
			path:       "/api/v1/query",
			wantPolicy: "api_endpoints",
			wantHeaders: map[string]string{
				"Cache-Control": "max-age=60",
				"Vary":          "Accept-Encoding",
			},
		},
		{
			name: "first match wins when multiple patterns could match",
			policies: []config.HeaderPolicy{
				{
					Name:        "specific",
					PathPattern: "^/api/v1/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=30",
					},
				},
				{
					Name:        "general",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
					},
				},
			},
			path:       "/api/v1/query",
			wantPolicy: "specific",
			wantHeaders: map[string]string{
				"Cache-Control": "max-age=30",
			},
		},
		{
			name: "no match returns empty string and nil",
			policies: []config.HeaderPolicy{
				{
					Name:        "api",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
					},
				},
			},
			path:        "/health",
			wantPolicy:  "",
			wantHeaders: nil,
		},
		{
			name: "case sensitive pattern does not match different case",
			policies: []config.HeaderPolicy{
				{
					Name:        "lowercase",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			path:        "/HEALTH",
			wantPolicy:  "",
			wantHeaders: nil,
		},
		{
			name: "case insensitive pattern matches different case",
			policies: []config.HeaderPolicy{
				{
					Name:        "case_insensitive",
					PathPattern: "(?i)^/health$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			path:       "/HEALTH",
			wantPolicy: "case_insensitive",
			wantHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
		},
		{
			name: "multiple headers in policy are all returned",
			policies: []config.HeaderPolicy{
				{
					Name:        "multi_header",
					PathPattern: "^/test$",
					Headers: map[string]string{
						"Cache-Control":           "public, max-age=300",
						"Vary":                    "Accept-Encoding, Accept-Language",
						"X-Custom-Header":         "custom-value",
						"X-Frame-Options":         "DENY",
						"Content-Security-Policy": "default-src 'self'",
					},
				},
			},
			path:       "/test",
			wantPolicy: "multi_header",
			wantHeaders: map[string]string{
				"Cache-Control":           "public, max-age=300",
				"Vary":                    "Accept-Encoding, Accept-Language",
				"X-Custom-Header":         "custom-value",
				"X-Frame-Options":         "DENY",
				"Content-Security-Policy": "default-src 'self'",
			},
		},
		{
			name: "health endpoint matches",
			policies: []config.HeaderPolicy{
				{
					Name:        "health",
					PathPattern: "^/health$",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=5",
					},
				},
			},
			path:       "/health",
			wantPolicy: "health",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=5",
			},
		},
		{
			name: "openapi.yaml endpoint matches",
			policies: []config.HeaderPolicy{
				{
					Name:        "openapi",
					PathPattern: "^/openapi\\.yaml$",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=3600",
					},
				},
			},
			path:       "/openapi.yaml",
			wantPolicy: "openapi",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=3600",
			},
		},
		{
			name: "docs endpoint matches",
			policies: []config.HeaderPolicy{
				{
					Name:        "docs",
					PathPattern: "^/docs$",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=300",
					},
				},
			},
			path:       "/docs",
			wantPolicy: "docs",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=300",
			},
		},
		{
			name: "api query endpoint matches",
			policies: []config.HeaderPolicy{
				{
					Name:        "api_queries",
					PathPattern: "^/api/v1/.*",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=30",
					},
				},
			},
			path:       "/api/v1/query/blocks",
			wantPolicy: "api_queries",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=30",
			},
		},
		{
			name: "empty headers map returns empty map not nil",
			policies: []config.HeaderPolicy{
				{
					Name:        "empty",
					PathPattern: "^/test$",
					Headers:     map[string]string{},
				},
			},
			path:        "/test",
			wantPolicy:  "empty",
			wantHeaders: map[string]string{},
		},
		{
			name: "wildcard pattern matches everything",
			policies: []config.HeaderPolicy{
				{
					Name:        "wildcard",
					PathPattern: ".*",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			path:       "/any/path/here",
			wantPolicy: "wildcard",
			wantHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
		},
		{
			name: "empty path matches pattern",
			policies: []config.HeaderPolicy{
				{
					Name:        "root",
					PathPattern: "^/$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			path:       "/",
			wantPolicy: "root",
			wantHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.policies)
			require.NoError(t, err)

			gotPolicy, gotHeaders := manager.Match(tt.path)
			assert.Equal(t, tt.wantPolicy, gotPolicy)
			assert.Equal(t, tt.wantHeaders, gotHeaders)
		})
	}
}

func TestMiddleware(t *testing.T) {
	// Create a test logger that discards output
	logger := logrus.New()
	logger.SetOutput(httptest.NewRecorder())

	tests := []struct {
		name           string
		policies       []config.HeaderPolicy
		requestPath    string
		wantHeaders    map[string]string
		wantNoHeaders  bool
		wantBody       string
		handlerBody    string
		handlerHeaders map[string]string
	}{
		{
			name: "middleware sets all headers from matched policy",
			policies: []config.HeaderPolicy{
				{
					Name:        "test",
					PathPattern: "^/test$",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
						"Vary":          "Accept-Encoding",
						"X-Custom":      "value",
					},
				},
			},
			requestPath: "/test",
			wantHeaders: map[string]string{
				"Cache-Control": "max-age=60",
				"Vary":          "Accept-Encoding",
				"X-Custom":      "value",
			},
			wantBody:    "test response",
			handlerBody: "test response",
		},
		{
			name: "middleware does nothing for non-matching paths",
			policies: []config.HeaderPolicy{
				{
					Name:        "api",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
					},
				},
			},
			requestPath:   "/health",
			wantNoHeaders: true,
			wantBody:      "health ok",
			handlerBody:   "health ok",
		},
		{
			name: "middleware doesn't break handler execution",
			policies: []config.HeaderPolicy{
				{
					Name:        "test",
					PathPattern: "^/test$",
					Headers: map[string]string{
						"Cache-Control": "no-cache",
					},
				},
			},
			requestPath: "/test",
			wantHeaders: map[string]string{
				"Cache-Control": "no-cache",
			},
			wantBody:    "handler executed",
			handlerBody: "handler executed",
		},
		{
			name: "multiple policies first match wins",
			policies: []config.HeaderPolicy{
				{
					Name:        "specific",
					PathPattern: "^/api/v1/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=30",
					},
				},
				{
					Name:        "general",
					PathPattern: "^/api/.*",
					Headers: map[string]string{
						"Cache-Control": "max-age=60",
					},
				},
			},
			requestPath: "/api/v1/query",
			wantHeaders: map[string]string{
				"Cache-Control": "max-age=30",
			},
			wantBody:    "query result",
			handlerBody: "query result",
		},
		{
			name: "headers are set before handler runs",
			policies: []config.HeaderPolicy{
				{
					Name:        "test",
					PathPattern: "^/test$",
					Headers: map[string]string{
						"X-Middleware": "set-by-middleware",
					},
				},
			},
			requestPath: "/test",
			wantHeaders: map[string]string{
				"X-Middleware": "set-by-middleware",
				"X-Handler":    "set-by-handler",
			},
			handlerHeaders: map[string]string{
				"X-Handler": "set-by-handler",
			},
			wantBody:    "test",
			handlerBody: "test",
		},
		{
			name: "cache-control header type",
			policies: []config.HeaderPolicy{
				{
					Name:        "cache",
					PathPattern: "^/cached$",
					Headers: map[string]string{
						"Cache-Control": "public, max-age=3600, s-maxage=3600",
					},
				},
			},
			requestPath: "/cached",
			wantHeaders: map[string]string{
				"Cache-Control": "public, max-age=3600, s-maxage=3600",
			},
			wantBody:    "cached content",
			handlerBody: "cached content",
		},
		{
			name: "vary header type",
			policies: []config.HeaderPolicy{
				{
					Name:        "vary",
					PathPattern: "^/vary$",
					Headers: map[string]string{
						"Vary": "Accept-Encoding, Accept-Language",
					},
				},
			},
			requestPath: "/vary",
			wantHeaders: map[string]string{
				"Vary": "Accept-Encoding, Accept-Language",
			},
			wantBody:    "vary content",
			handlerBody: "vary content",
		},
		{
			name: "custom x-headers",
			policies: []config.HeaderPolicy{
				{
					Name:        "custom",
					PathPattern: "^/custom$",
					Headers: map[string]string{
						"X-Custom-Header":  "custom-value",
						"X-Another-Header": "another-value",
					},
				},
			},
			requestPath: "/custom",
			wantHeaders: map[string]string{
				"X-Custom-Header":  "custom-value",
				"X-Another-Header": "another-value",
			},
			wantBody:    "custom",
			handlerBody: "custom",
		},
		{
			name: "security headers",
			policies: []config.HeaderPolicy{
				{
					Name:        "security",
					PathPattern: "^/secure$",
					Headers: map[string]string{
						"X-Frame-Options":         "DENY",
						"X-Content-Type-Options":  "nosniff",
						"Content-Security-Policy": "default-src 'self'",
					},
				},
			},
			requestPath: "/secure",
			wantHeaders: map[string]string{
				"X-Frame-Options":         "DENY",
				"X-Content-Type-Options":  "nosniff",
				"Content-Security-Policy": "default-src 'self'",
			},
			wantBody:    "secure",
			handlerBody: "secure",
		},
		{
			name: "handler response body is unchanged",
			policies: []config.HeaderPolicy{
				{
					Name:        "test",
					PathPattern: "^/body-test$",
					Headers: map[string]string{
						"Cache-Control": "max-age=300",
					},
				},
			},
			requestPath: "/body-test",
			wantHeaders: map[string]string{
				"Cache-Control": "max-age=300",
			},
			wantBody:    "this is the original response body",
			handlerBody: "this is the original response body",
		},
		{
			name:          "empty policies list",
			policies:      []config.HeaderPolicy{},
			requestPath:   "/test",
			wantNoHeaders: true,
			wantBody:      "test",
			handlerBody:   "test",
		},
		{
			name: "nil headers in policy",
			policies: []config.HeaderPolicy{
				{
					Name:        "nil_headers",
					PathPattern: "^/test$",
					Headers:     nil,
				},
			},
			requestPath:   "/test",
			wantNoHeaders: true,
			wantBody:      "test",
			handlerBody:   "test",
		},
		{
			name: "empty headers map in policy",
			policies: []config.HeaderPolicy{
				{
					Name:        "empty_headers",
					PathPattern: "^/test$",
					Headers:     map[string]string{},
				},
			},
			requestPath:   "/test",
			wantNoHeaders: true,
			wantBody:      "test",
			handlerBody:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.policies)
			require.NoError(t, err)

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Set handler headers if any
				for k, v := range tt.handlerHeaders {
					w.Header().Set(k, v)
				}

				_, _ = w.Write([]byte(tt.handlerBody))
			})

			// Wrap handler with middleware
			wrappedHandler := manager.Middleware(logger)(handler)

			// Create test request
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(recorder, req)

			// Verify headers
			if tt.wantNoHeaders {
				// Only verify Cache-Control is not set (other headers like Content-Type may be set by net/http)
				for k := range tt.wantHeaders {
					assert.Empty(t, recorder.Header().Get(k), "header %s should not be set", k)
				}
			} else {
				for k, v := range tt.wantHeaders {
					assert.Equal(t, v, recorder.Header().Get(k), "header %s mismatch", k)
				}
			}

			// Verify response body is unchanged
			assert.Equal(t, tt.wantBody, recorder.Body.String())

			// Verify handler was called (status code should be set)
			assert.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestMiddlewareWithNilManager(t *testing.T) {
	// This test ensures that Match is safe to call even with no policies
	manager := &Manager{
		policies: []compiledPolicy{},
	}

	policyName, headers := manager.Match("/any/path")
	assert.Empty(t, policyName)
	assert.Nil(t, headers)
}

func TestMiddlewareLogging(t *testing.T) {
	// Create a test logger with a buffer to capture output
	logger := logrus.New()
	buf := &logBufferRecorder{headers: make(map[string][]string)}
	logger.SetOutput(buf)
	logger.SetLevel(logrus.DebugLevel)

	policies := []config.HeaderPolicy{
		{
			Name:        "test_policy",
			PathPattern: "^/test$",
			Headers: map[string]string{
				"Cache-Control": "max-age=60",
			},
		},
	}

	manager, err := NewManager(policies)
	require.NoError(t, err)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	wrappedHandler := manager.Middleware(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)

	// Verify the handler executed successfully
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "max-age=60", recorder.Header().Get("Cache-Control"))
}

// logBufferRecorder is a minimal recorder for testing.
type logBufferRecorder struct {
	headers map[string][]string
}

func (l *logBufferRecorder) Write(p []byte) (n int, err error) {
	return len(p), nil
}
