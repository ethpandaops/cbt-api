package telemetry

import (
	"go.opentelemetry.io/otel/attribute"
)

// Standard semantic convention attribute keys.
var (
	// HTTP attribute keys (following OpenTelemetry semantic conventions).
	HTTPMethodKey           = attribute.Key("http.request.method")
	HTTPRouteKey            = attribute.Key("http.route")
	HTTPStatusCodeKey       = attribute.Key("http.response.status_code")
	HTTPUserAgentKey        = attribute.Key("user_agent.original")
	HTTPRequestBodySizeKey  = attribute.Key("http.request.body.size")
	HTTPResponseBodySizeKey = attribute.Key("http.response.body.size")

	// Database attribute keys (following OpenTelemetry semantic conventions).
	DBSystemKey    = attribute.Key("db.system")
	DBNameKey      = attribute.Key("db.name")
	DBStatementKey = attribute.Key("db.statement")
	DBOperationKey = attribute.Key("db.operation.name")
)

// Custom attribute keys specific to xatu-cbt-api.
const (
	// Query parameter attributes.
	AttrQueryParams      = attribute.Key("http.query_params")
	AttrPaginationOffset = attribute.Key("pagination.offset")
	AttrPaginationLimit  = attribute.Key("pagination.limit")

	// Database query attributes.
	AttrDBQueryParams  = attribute.Key("db.query_params")
	AttrDBRowsAffected = attribute.Key("db.rows_affected")
	AttrDBRowsReturned = attribute.Key("db.rows_returned")
	AttrDBUseFinal     = attribute.Key("db.use_final")

	// Service attributes.
	AttrNetworkName = attribute.Key("network.name")
)

// AddHTTPAttributes adds standard HTTP attributes to a span.
func AddHTTPAttributes(attrs []attribute.KeyValue, method, route string, statusCode int) []attribute.KeyValue {
	return append(attrs,
		HTTPMethodKey.String(method),
		HTTPRouteKey.String(route),
		HTTPStatusCodeKey.Int(statusCode),
	)
}

// AddDatabaseAttributes adds standard database attributes to a span.
func AddDatabaseAttributes(attrs []attribute.KeyValue, dbName, operation, statement string) []attribute.KeyValue {
	return append(attrs,
		DBSystemKey.String("clickhouse"),
		DBNameKey.String(dbName),
		DBOperationKey.String(operation),
		DBStatementKey.String(statement),
	)
}
