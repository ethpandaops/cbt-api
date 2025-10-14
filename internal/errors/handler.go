package errors

import (
	"errors"
	"net/http"

	"github.com/ethpandaops/cbt-api/internal/handlers"
	"github.com/sirupsen/logrus"
)

// DefaultErrorHandler is the default HTTP error handler that converts errors
// to Status responses matching the OpenAPI specification.
func DefaultErrorHandler(logger logrus.FieldLogger) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// Check for parameter binding errors from oapi-codegen
		var paramErr *handlers.InvalidParamFormatError
		if errors.As(err, &paramErr) {
			// Return 400 Bad Request with sanitized message (don't expose internal parse errors)
			status := BadRequestf("Invalid value for parameter '%s'", paramErr.ParamName)

			logger.WithFields(logrus.Fields{
				"param":  paramErr.ParamName,
				"path":   r.URL.Path,
				"method": r.Method,
			}).Warn("invalid parameter format")

			status.WriteJSON(w)

			return
		}

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

		// For unknown errors, return generic message (don't expose internals)
		status := Internal("An internal error occurred")

		logger.WithFields(logrus.Fields{
			"error":  err, // Log actual error for debugging
			"path":   r.URL.Path,
			"method": r.Method,
		}).Error("internal error")

		status.WriteJSON(w)
	}
}
