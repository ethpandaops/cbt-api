package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	// MinGzipSize is the minimum size in bytes for gzip compression to be applied.
	// Compressing very small responses isn't worth the CPU overhead.
	MinGzipSize = 860

	// DefaultGzipLevel uses the default compression level for balance between speed and size.
	DefaultGzipLevel = gzip.DefaultCompression
)

// Pool of gzip writers to reduce allocations.
var gzipPool = sync.Pool{
	New: func() interface{} {
		w, err := gzip.NewWriterLevel(nil, DefaultGzipLevel)
		if err != nil {
			// Fall back to default compression if level is invalid
			w = gzip.NewWriter(nil)
		}

		return w
	},
}

// Pool of buffers to reduce allocations.
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// bufferedResponseWriter captures response data to determine if compression is worthwhile.
type bufferedResponseWriter struct {
	http.ResponseWriter
	buffer      *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

// Write writes the data to the buffer.
func (w *bufferedResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.buffer.Write(b)
}

// WriteHeader captures the status code.
func (w *bufferedResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
	}
}

// GzipOption is a functional option for configuring the Gzip middleware.
type GzipOption func(*gzipConfig)

// gzipConfig holds the configuration for the Gzip middleware.
type gzipConfig struct {
	excludePaths map[string]bool
}

// WithExcludePaths returns a GzipOption that excludes specific paths from gzip compression.
func WithExcludePaths(paths ...string) GzipOption {
	return func(cfg *gzipConfig) {
		for _, path := range paths {
			cfg.excludePaths[path] = true
		}
	}
}

// Gzip returns a middleware that compresses HTTP responses using gzip.
// It only compresses if the client supports gzip encoding and response size >= MinGzipSize.
func Gzip(opts ...GzipOption) func(http.Handler) http.Handler {
	cfg := &gzipConfig{
		excludePaths: make(map[string]bool),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip compression for excluded paths
			if cfg.excludePaths[r.URL.Path] {
				next.ServeHTTP(w, r)

				return
			}

			// Check if client accepts gzip encoding
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)

				return
			}

			// Get buffer from pool
			buf, ok := bufferPool.Get().(*bytes.Buffer)
			if !ok {
				buf = new(bytes.Buffer)
			}

			buf.Reset()

			defer bufferPool.Put(buf)

			// Create buffered response writer
			bw := &bufferedResponseWriter{
				ResponseWriter: w,
				buffer:         buf,
				statusCode:     http.StatusOK,
			}

			// Set Vary header to indicate response varies based on Accept-Encoding
			w.Header().Set("Vary", "Accept-Encoding")

			// Let handler write to buffer
			next.ServeHTTP(bw, r)

			// Decide whether to compress based on size
			if buf.Len() >= MinGzipSize {
				// Response is large enough to benefit from compression
				// Get gzip writer from pool
				gz, ok := gzipPool.Get().(*gzip.Writer)
				if !ok {
					// If type assertion fails, send uncompressed
					w.WriteHeader(bw.statusCode)
					_, _ = io.Copy(w, buf)

					return
				}

				defer gzipPool.Put(gz)

				gz.Reset(w)

				defer func() {
					_ = gz.Close()
				}()

				// Set compression headers
				w.Header().Set("Content-Encoding", "gzip")
				w.Header().Del("Content-Length")
				w.WriteHeader(bw.statusCode)

				// Write compressed data
				_, _ = io.Copy(gz, buf)
			} else {
				// Response is too small, send uncompressed
				w.WriteHeader(bw.statusCode)
				_, _ = io.Copy(w, buf)
			}
		})
	}
}
