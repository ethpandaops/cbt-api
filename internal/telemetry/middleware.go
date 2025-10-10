package telemetry

import (
	"net/http"
	"strconv"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
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

		// Wrap with custom attribute extraction
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract and add custom attributes
			ctx := r.Context()
			span := oteltrace.SpanFromContext(ctx)

			if span.SpanContext().IsValid() {
				addCustomHTTPAttributes(span, r)
			}

			handler.ServeHTTP(w, r)
		})
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
