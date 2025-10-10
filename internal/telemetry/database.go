package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu-cbt-api/internal/database"
)

// TracedClient wraps database.Client with OpenTelemetry instrumentation.
type TracedClient struct {
	client *database.Client
	tracer oteltrace.Tracer
	dbName string
	log    logrus.FieldLogger
}

// Ensure TracedClient implements database.DatabaseClient interface.
var _ database.DatabaseClient = (*TracedClient)(nil)

// NewTracedClient wraps a database client with tracing.
func NewTracedClient(client *database.Client, dbName string, logger logrus.FieldLogger) *TracedClient {
	return &TracedClient{
		client: client,
		tracer: otel.Tracer("github.com/ethpandaops/xatu-cbt-api/database"),
		dbName: dbName,
		log:    logger,
	}
}

// Query executes a query with tracing.
func (c *TracedClient) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	ctx, span := c.startSpan(ctx, "Query", query, args...)
	defer span.End()

	rows, err := c.client.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, err
	}

	return &tracedRows{Rows: rows, span: span}, nil
}

// QueryRow executes a query that returns a single row with tracing.
func (c *TracedClient) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	ctx, span := c.startSpan(ctx, "QueryRow", query, args...)
	defer span.End()

	row := c.client.QueryRow(ctx, query, args...)

	// Note: driver.Row doesn't expose errors until Scan() is called
	// We record the operation but can't capture row-level errors here.
	span.SetStatus(codes.Ok, "")

	return row
}

// Select executes a query and scans results into dest with tracing.
func (c *TracedClient) Select(ctx context.Context, dest any, query string, args ...any) error {
	ctx, span := c.startSpan(ctx, "Select", query, args...)
	defer span.End()

	err := c.client.Select(ctx, dest, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	span.SetStatus(codes.Ok, "")

	return nil
}

// Exec executes a query without returning rows with tracing.
func (c *TracedClient) Exec(ctx context.Context, query string, args ...any) error {
	ctx, span := c.startSpan(ctx, "Exec", query, args...)
	defer span.End()

	err := c.client.Exec(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	span.SetStatus(codes.Ok, "")

	return nil
}

// Close closes the database connection.
func (c *TracedClient) Close() error {
	return c.client.Close()
}

// startSpan creates a new span for a database operation.
func (c *TracedClient) startSpan(ctx context.Context, operation, query string, args ...any) (context.Context, oteltrace.Span) {
	// Extract SQL operation (SELECT, INSERT, UPDATE, etc.)
	sqlOp := extractSQLOperation(query)

	// Create span
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("clickhouse.%s", operation),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
	)

	// Add standard database attributes
	attrs := AddDatabaseAttributes([]attribute.KeyValue{},
		c.dbName,
		sqlOp,
		truncateString(query, 2048),
	)

	// Add query parameters as attributes (if not too many)
	if len(args) > 0 && len(args) <= 10 {
		paramsStr := fmt.Sprintf("%v", args)
		paramsStr = truncateString(paramsStr, 512)
		attrs = append(attrs, AttrDBQueryParams.String(paramsStr))
	}

	span.SetAttributes(attrs...)

	return ctx, span
}

// extractSQLOperation extracts the SQL operation from a query.
func extractSQLOperation(query string) string {
	query = strings.TrimSpace(query)

	parts := strings.SplitN(query, " ", 2)

	if len(parts) > 0 {
		return strings.ToUpper(parts[0])
	}

	return "UNKNOWN"
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "...[truncated]"
	}

	return s
}

// tracedRows wraps driver.Rows to record row count on close.
type tracedRows struct {
	driver.Rows
	span     oteltrace.Span
	rowCount int64
}

// Next wraps the original Next and counts rows.
func (r *tracedRows) Next() bool {
	hasNext := r.Rows.Next()
	if hasNext {
		r.rowCount++
	} else {
		// When iteration completes, record row count
		r.span.SetAttributes(AttrDBRowsReturned.Int64(r.rowCount))
		r.span.SetStatus(codes.Ok, "")
	}

	return hasNext
}

// Close wraps the original Close.
func (r *tracedRows) Close() error {
	// Record final row count if not already done
	if r.rowCount > 0 {
		r.span.SetAttributes(AttrDBRowsReturned.Int64(r.rowCount))
	}

	return r.Rows.Close()
}
