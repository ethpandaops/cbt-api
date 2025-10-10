package errors

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

// DefaultErrorHandler is the default HTTP error handler that converts errors
// to Status responses matching the OpenAPI specification.
func DefaultErrorHandler(logger logrus.FieldLogger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// If the error is already a Status, write it directly
		if status, ok := err.(*Status); ok {
			logger.WithFields(logrus.Fields{
				"code":    status.Code,
				"message": status.Message,
				"path":    r.URL.Path,
				"method":  r.Method,
			}).Warn("request error")

			status.WriteJSON(w)

			return
		}

		// For unknown errors, wrap in Internal status
		status := Internal(err.Error())
		logger.WithFields(logrus.Fields{
			"error":  err,
			"path":   r.URL.Path,
			"method": r.Method,
		}).Error("internal error")

		status.WriteJSON(w)
	}
}
