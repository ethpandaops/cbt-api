package errors

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
)

// Code is an alias for gRPC status codes.
// These codes match the google.rpc.Code enum values.
type Code = codes.Code

// HTTPStatus returns the HTTP status code for a gRPC code.
// Based on https://github.com/grpc-ecosystem/grpc-gateway/blob/master/runtime/errors.go
func HTTPStatus(c Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// Detail represents an error detail with a type and arbitrary fields.
type Detail map[string]any

// Status represents a logical error model compatible with google.rpc.Status.
// This matches the Status schema in the OpenAPI specification.
//
//nolint:errname // Intentionally named "Status" to match google.rpc.Status convention
type Status struct {
	// Code is the status code, which should be an enum value of google.rpc.Code.
	Code Code `json:"code"`
	// Message is a developer-facing error message in English.
	Message string `json:"message"`
	// Details is a list of messages that carry error details.
	Details []Detail `json:"details,omitempty"`
}

// Error implements the error interface.
func (s *Status) Error() string {
	return s.Message
}

// WithDetail adds a detail to the status and returns the status for chaining.
func (s *Status) WithDetail(detail Detail) *Status {
	if s.Details == nil {
		s.Details = make([]Detail, 0, 1)
	}

	s.Details = append(s.Details, detail)

	return s
}

// WithMetadata adds a detail with ErrorInfo type and the given metadata.
func (s *Status) WithMetadata(metadata map[string]string) *Status {
	detail := Detail{
		"@type":    "type.googleapis.com/ErrorInfo",
		"metadata": metadata,
	}

	return s.WithDetail(detail)
}

// WriteJSON writes the status as JSON to the http.ResponseWriter.
func (s *Status) WriteJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(HTTPStatus(s.Code))

	if err := json.NewEncoder(w).Encode(s); err != nil {
		// Fallback to plain text if JSON encoding fails
		http.Error(w, s.Message, HTTPStatus(s.Code))
	}
}

// New creates a new Status with the given code and message.
func New(code Code, message string) *Status {
	return &Status{
		Code:    code,
		Message: message,
	}
}

// Newf creates a new Status with a formatted message.
func Newf(code Code, format string, args ...any) *Status {
	return &Status{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// BadRequest creates a Status for invalid arguments (400).
func BadRequest(message string) *Status {
	return New(codes.InvalidArgument, message)
}

// BadRequestf creates a Status for invalid arguments with formatted message.
func BadRequestf(format string, args ...any) *Status {
	return Newf(codes.InvalidArgument, format, args...)
}

// NotFound creates a Status for not found errors (404).
func NotFound(message string) *Status {
	return New(codes.NotFound, message)
}

// NotFoundf creates a Status for not found errors with formatted message.
func NotFoundf(format string, args ...any) *Status {
	return Newf(codes.NotFound, format, args...)
}

// Internal creates a Status for internal errors (500).
func Internal(message string) *Status {
	return New(codes.Internal, message)
}

// Internalf creates a Status for internal errors with formatted message.
func Internalf(format string, args ...any) *Status {
	return Newf(codes.Internal, format, args...)
}

// Unavailable creates a Status for service unavailable errors (503).
func Unavailable(message string) *Status {
	return New(codes.Unavailable, message)
}

// Unavailablef creates a Status for service unavailable errors with formatted message.
func Unavailablef(format string, args ...any) *Status {
	return Newf(codes.Unavailable, format, args...)
}

// Unauthenticated creates a Status for authentication errors (401).
func Unauthenticated(message string) *Status {
	return New(codes.Unauthenticated, message)
}

// Unauthenticatedf creates a Status for authentication errors with formatted message.
func Unauthenticatedf(format string, args ...any) *Status {
	return Newf(codes.Unauthenticated, format, args...)
}

// PermissionDenied creates a Status for permission errors (403).
func PermissionDenied(message string) *Status {
	return New(codes.PermissionDenied, message)
}

// PermissionDeniedf creates a Status for permission errors with formatted message.
func PermissionDeniedf(format string, args ...any) *Status {
	return Newf(codes.PermissionDenied, format, args...)
}
