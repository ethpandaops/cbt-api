package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzip(t *testing.T) {
	tests := []struct {
		name              string
		acceptEncoding    string
		responseBody      string
		expectCompressed  bool
		expectVaryHeader  bool
		expectContentType string
	}{
		{
			name:             "compresses when client accepts gzip",
			acceptEncoding:   "gzip",
			responseBody:     strings.Repeat("test data ", 100),
			expectCompressed: true,
			expectVaryHeader: true,
		},
		{
			name:             "compresses with multiple encodings",
			acceptEncoding:   "deflate, gzip, br",
			responseBody:     strings.Repeat("test data ", 100),
			expectCompressed: true,
			expectVaryHeader: true,
		},
		{
			name:             "does not compress when gzip not accepted",
			acceptEncoding:   "deflate",
			responseBody:     "test data",
			expectCompressed: false,
			expectVaryHeader: false,
		},
		{
			name:             "does not compress when no accept-encoding",
			acceptEncoding:   "",
			responseBody:     "test data",
			expectCompressed: false,
			expectVaryHeader: false,
		},
		{
			name:             "compresses small response",
			acceptEncoding:   "gzip",
			responseBody:     "small",
			expectCompressed: true,
			expectVaryHeader: true,
		},
		{
			name:             "compresses large response",
			acceptEncoding:   "gzip",
			responseBody:     strings.Repeat("large data ", 1000),
			expectCompressed: true,
			expectVaryHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := Gzip()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(tt.responseBody))
				require.NoError(t, err)
			}))

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			// Record response
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Check status code
			assert.Equal(t, http.StatusOK, rec.Code)

			// Check Vary header
			if tt.expectVaryHeader {
				assert.Equal(t, "Accept-Encoding", rec.Header().Get("Vary"))
			}

			// Check compression
			if tt.expectCompressed {
				assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
				assert.Empty(t, rec.Header().Get("Content-Length"))

				// Decompress and verify content
				gr, err := gzip.NewReader(rec.Body)
				require.NoError(t, err)

				defer gr.Close()

				decompressed, err := io.ReadAll(gr)
				require.NoError(t, err)
				assert.Equal(t, tt.responseBody, string(decompressed))
			} else {
				assert.Empty(t, rec.Header().Get("Content-Encoding"))
				assert.Equal(t, tt.responseBody, rec.Body.String())
			}
		})
	}
}

func TestGzip_WriteHeader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Gzip()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
			assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
		})
	}
}

func TestGzip_HeadersSetOnFirstWrite(t *testing.T) {
	handler := Gzip()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write multiple times
		_, err := w.Write([]byte("first"))
		require.NoError(t, err)
		_, err = w.Write([]byte("second"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	assert.Empty(t, rec.Header().Get("Content-Length"))

	// Verify decompressed content
	gr, err := gzip.NewReader(rec.Body)
	require.NoError(t, err)

	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	require.NoError(t, err)
	assert.Equal(t, "firstsecond", string(decompressed))
}

func TestGzip_PoolReuse(t *testing.T) {
	handler := Gzip()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test"))
		require.NoError(t, err)
	}))

	// Make multiple requests to test pool reuse
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Accept-Encoding", "gzip")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
	}
}

func TestGzip_PreexistingContentEncoding(t *testing.T) {
	handler := Gzip()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set content encoding before write
		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("test"))
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should respect pre-existing Content-Encoding
	assert.Equal(t, "br", rec.Header().Get("Content-Encoding"))
}
