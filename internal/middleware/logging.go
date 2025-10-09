package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Logging returns a middleware that logs HTTP requests.
func Logging(logger logrus.FieldLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      wrapped.statusCode,
				"duration_ms": time.Since(start).Milliseconds(),
				"remote_addr": r.RemoteAddr,
			}).Info("request")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
