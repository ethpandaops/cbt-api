package telemetry

import (
	"fmt"
	"net/http"
	"strconv"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware returns a middleware that traces HTTP requests
// Uses otelhttp for standard HTTP instrumentation.
func HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Use otelhttp.NewHandler with custom options
		handler := otelhttp.NewHandler(next, "xatu-cbt-api",
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
			otelhttp.WithSpanNameFormatter(spanNameFormatter),
			otelhttp.WithSpanOptions(oteltrace.WithSpanKind(oteltrace.SpanKindServer)),
		)

		// Wrap with custom attribute extraction and status setting
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract and add custom attributes
			ctx := r.Context()
			span := oteltrace.SpanFromContext(ctx)

			if span.SpanContext().IsValid() {
				addCustomHTTPAttributes(span, r)
			}

			// Wrap response writer to capture status code
			wrapped := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default to 200
			}

			// Serve request
			handler.ServeHTTP(wrapped, r)

			// Set span status based on HTTP status code
			if span.SpanContext().IsValid() {
				setSpanStatus(span, wrapped.statusCode)
			}
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// setSpanStatus sets the span status based on HTTP status code.
func setSpanStatus(span oteltrace.Span, statusCode int) {
	switch {
	case statusCode >= 200 && statusCode < 400:
		// 2xx and 3xx are successful
		span.SetStatus(codes.Ok, "")
	case statusCode >= 400 && statusCode < 500:
		// 4xx are client errors
		span.SetStatus(codes.Error, fmt.Sprintf("Client error: %d", statusCode))
	case statusCode >= 500:
		// 5xx are server errors
		span.SetStatus(codes.Error, fmt.Sprintf("Server error: %d", statusCode))
	}
}

// spanNameFormatter formats span names as "HTTP {METHOD} {ROUTE}".
func spanNameFormatter(operation string, r *http.Request) string {
	return "HTTP " + r.Method + " " + r.URL.Path
}

// addCustomHTTPAttributes adds xatu-cbt-api specific attributes.
func addCustomHTTPAttributes(span oteltrace.Span, r *http.Request) {
	// Add query parameters as attribute (truncated if too long)
	queryParams := r.URL.RawQuery
	if len(queryParams) > 0 {
		if len(queryParams) > 1024 {
			queryParams = queryParams[:1024] + "...[truncated]"
		}

		span.SetAttributes(AttrQueryParams.String(queryParams))
	}

	// Extract common pagination parameters.
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if offsetInt, err := strconv.Atoi(offset); err == nil {
			span.SetAttributes(AttrPaginationOffset.Int(offsetInt))
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if limitInt, err := strconv.Atoi(limit); err == nil {
			span.SetAttributes(AttrPaginationLimit.Int(limitInt))
		}
	}

	// Add request content length if available.
	if r.ContentLength > 0 {
		span.SetAttributes(HTTPRequestBodySizeKey.Int64(r.ContentLength))
	}
}
