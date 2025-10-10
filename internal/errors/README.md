# Error Handling

This package provides standardized error responses that match the OpenAPI specification's `Status` schema, which is based on Google's error model used by gRPC and Google Cloud APIs.

## Status Format

All API errors follow this structure:

```json
{
  "code": 3,
  "message": "unknown query parameter(s): slot_gtw",
  "details": [
    {
      "@type": "type.googleapis.com/ErrorInfo",
      "metadata": {
        "unknown_parameters": "slot_gtw",
        "valid_parameters": "slot_gte, slot_gt, slot_eq, ..."
      }
    }
  ]
}
```

## Error Codes

The `code` field uses gRPC status codes that map to HTTP status codes:

| gRPC Code | HTTP Status | Helper Function | Description |
|-----------|-------------|-----------------|-------------|
| 3 | 400 Bad Request | `BadRequest()` | Invalid client argument |
| 5 | 404 Not Found | `NotFound()` | Resource not found |
| 7 | 403 Forbidden | `PermissionDenied()` | Permission denied |
| 13 | 500 Internal Server Error | `Internal()` | Internal server error |
| 14 | 503 Service Unavailable | `Unavailable()` | Service unavailable |
| 16 | 401 Unauthorized | `Unauthenticated()` | Missing authentication |

See [status.go](status.go) for the complete list of codes.

## Usage

### Basic Error

```go
import apierrors "github.com/ethpandaops/xatu-cbt-api/internal/errors"

// Return a bad request error
status := apierrors.BadRequest("invalid parameter")
status.WriteJSON(w)
```

### Error with Formatted Message

```go
status := apierrors.BadRequestf("value %d is out of range", value)
status.WriteJSON(w)
```

### Error with Metadata

```go
status := apierrors.BadRequest("unknown parameter: foo").
    WithMetadata(map[string]string{
        "parameter": "foo",
        "expected": "bar",
    })
status.WriteJSON(w)
```

### Error with Custom Details

```go
detail := apierrors.Detail{
    "@type": "type.googleapis.com/FieldViolation",
    "field": "email",
    "description": "Invalid email format",
}

status := apierrors.BadRequest("validation failed").
    WithDetail(detail)
status.WriteJSON(w)
```

### Using in Middleware

```go
func MyMiddleware(logger logrus.FieldLogger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if someCondition {
                status := apierrors.BadRequest("condition not met")
                status.WriteJSON(w)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Default Error Handler

The package provides a default error handler for the generated OpenAPI handlers:

```go
handlers.HandlerWithOptions(impl, handlers.StdHTTPServerOptions{
    BaseRouter:       mux,
    ErrorHandlerFunc: apierrors.DefaultErrorHandler(logger),
})
```

This handler automatically:
- Converts `*Status` errors to JSON responses
- Wraps unknown errors as Internal errors
- Logs all errors with appropriate fields

## Benefits

- **Consistent**: All errors follow the same structure
- **Machine-readable**: Structured `details` for programmatic handling
- **Standard**: Based on Google's error model (gRPC, Google Cloud APIs)
- **Spec-compliant**: Matches the OpenAPI `Status` schema
- **Developer-friendly**: Clear error messages and metadata
- **Type-safe**: Compile-time checking with Go types

## References

- [Google API Design Guide - Errors](https://cloud.google.com/apis/design/errors)
- [gRPC Status Codes](https://grpc.github.io/grpc/core/md_doc_statuscodes.html)
- [google.rpc.Status](https://github.com/googleapis/googleapis/blob/master/google/rpc/status.proto)
