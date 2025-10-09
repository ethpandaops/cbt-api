package middleware

import (
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

// gzipResponseWriter wraps http.ResponseWriter to provide gzip compression.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	headerWritten bool
}

// Write writes the data to the connection as part of an HTTP reply.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		// Set content encoding header before first write
		if w.Header().Get("Content-Encoding") == "" {
			w.Header().Set("Content-Encoding", "gzip")
		}

		w.Header().Del("Content-Length") // Remove content-length as it will change
		w.headerWritten = true
	}

	return w.Writer.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code.
func (w *gzipResponseWriter) WriteHeader(code int) {
	if !w.headerWritten {
		if w.Header().Get("Content-Encoding") == "" {
			w.Header().Set("Content-Encoding", "gzip")
		}

		w.Header().Del("Content-Length") // Remove content-length as it will change
		w.headerWritten = true
	}

	w.ResponseWriter.WriteHeader(code)
}

// Gzip returns a middleware that compresses HTTP responses using gzip.
// It only compresses if the client supports gzip encoding.
func Gzip() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip encoding
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)

				return
			}

			// Get gzip writer from pool
			gz, ok := gzipPool.Get().(*gzip.Writer)
			if !ok {
				// If type assertion fails, serve uncompressed
				next.ServeHTTP(w, r)

				return
			}

			defer gzipPool.Put(gz)

			gz.Reset(w)

			defer func() {
				_ = gz.Close()
			}()

			// Set Vary header to indicate response varies based on Accept-Encoding
			w.Header().Set("Vary", "Accept-Encoding")

			// Wrap response writer
			gzw := &gzipResponseWriter{
				Writer:         gz,
				ResponseWriter: w,
			}

			next.ServeHTTP(gzw, r)
		})
	}
}
