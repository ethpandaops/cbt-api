package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// Recovery returns a middleware that recovers from panics
func Recovery(logger logrus.FieldLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"error": err,
						"path":  r.URL.Path,
						"stack": string(debug.Stack()),
					}).Error("panic recovered")

					// Return 500 Internal Server Error
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					// Write error response
					errorMsg := `{"error":"Internal Server Error","message":"An unexpected error occurred","code":500}`
					if _, err := w.Write([]byte(errorMsg)); err != nil {
						logger.WithError(err).Error("Failed to write panic recovery response")
					}

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
