package middleware

import (
	"net/http"

	apierrors "github.com/ethpandaops/cbt-api/internal/errors"
)

// NotFoundHandler returns a middleware that converts 404 responses to Status format.
func NotFoundHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the ResponseWriter to capture the status code
			wrapped := &notFoundResponseWriter{
				ResponseWriter: w,
				path:           r.URL.Path,
			}

			next.ServeHTTP(wrapped, r)

			// If a 404 was written, convert to Status format
			if wrapped.status == http.StatusNotFound {
				status := apierrors.NotFoundf("path not found: %s", r.URL.Path)
				status.WriteJSON(w)
			}
		})
	}
}

type notFoundResponseWriter struct {
	http.ResponseWriter
	status        int
	headerWritten bool
	path          string
}

func (w *notFoundResponseWriter) WriteHeader(status int) {
	w.status = status
	w.headerWritten = true

	// Only write the header if it's not a 404
	// (we'll handle 404s after the handler completes)
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *notFoundResponseWriter) Write(b []byte) (int, error) {
	// If no explicit status was set, assume 200
	if w.status == 0 {
		w.status = http.StatusOK
	}

	// Don't write 404 body - we'll replace it with Status format
	if w.status == http.StatusNotFound {
		return len(b), nil // Pretend we wrote it
	}

	// Write header if not already written
	if !w.headerWritten {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerWritten = true
	}

	return w.ResponseWriter.Write(b)
}
