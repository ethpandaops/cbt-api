package server

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cbt-api/internal/config"
	"github.com/ethpandaops/cbt-api/internal/database"
	apierrors "github.com/ethpandaops/cbt-api/internal/errors"
	"github.com/ethpandaops/cbt-api/internal/handlers"
	"github.com/ethpandaops/cbt-api/internal/middleware"
	"github.com/ethpandaops/cbt-api/internal/telemetry"
)

//go:embed openapi.yaml
var openapiSpec embed.FS

// New creates a new HTTP server with all routes and middleware configured.
func New(cfg *config.Config, logger logrus.FieldLogger) (*http.Server, error) {
	// Connect to ClickHouse
	db, err := database.NewClient(&cfg.ClickHouse, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClickHouse client: %w", err)
	}

	// Wrap database client with tracing
	var tracedDB database.DatabaseClient = telemetry.NewTracedClient(db, cfg.ClickHouse.Database, logger)

	// Create generated server implementation.
	impl := &Server{
		db:     tracedDB,
		config: cfg,
	}

	// Setup router using native http.ServeMux with method routing
	mux := http.NewServeMux()

	// Health & metrics endpoints
	mux.HandleFunc("GET /health", handlers.Health)
	mux.Handle("GET /metrics", promhttp.Handler())

	// OpenAPI spec endpoint
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		data, _ := openapiSpec.ReadFile("openapi.yaml")

		w.Header().Set("Content-Type", "application/x-yaml")
		_, _ = w.Write(data)
	})

	// Scalar API documentation at /docs
	mux.HandleFunc("GET /docs", serveScalarDocs)
	mux.HandleFunc("GET /docs/", serveScalarDocs)

	// Register generated API handlers with custom error handler
	handlers.HandlerWithOptions(impl, handlers.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: apierrors.DefaultErrorHandler(logger),
	})

	// Apply middleware stack (wrap the mux)
	handler := middleware.Logging(logger)(mux)
	handler = middleware.NotFoundHandler()(handler)
	handler = middleware.QueryParameterValidation(logger)(handler)
	handler = middleware.CORS()(handler)
	handler = middleware.Recovery(logger)(handler)
	handler = middleware.Metrics()(handler)

	// Add tracing middleware (only if telemetry is enabled)
	if cfg.Telemetry.Enabled {
		handler = telemetry.HTTPMiddleware()(handler)
	}

	handler = middleware.Gzip(middleware.WithExcludePaths("/metrics"))(handler)

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           handler,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}, nil
}

// serveScalarDocs serves the Scalar API documentation UI.
func serveScalarDocs(w http.ResponseWriter, _ *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>CBT API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <script
        id="api-reference"
        type="application/json"
        data-configuration='{
            "url": "/openapi.yaml",
            "theme": "kepler",
            "hideClientButton": true,
            "hideDarkModeToggle": false,
            "showSidebar": true,
            "showToolbar": "localhost",
            "operationTitleSource": "summary",
            "persistAuth": false,
            "telemetry": true,
            "layout": "modern",
            "isEditable": false,
            "isLoading": false,
            "hideModels": false,
            "documentDownloadType": "both",
            "hideTestRequestButton": false,
            "hideSearch": false,
            "showOperationId": false,
            "withDefaultFonts": true,
            "defaultOpenAllTags": false,
            "expandAllModelSections": false,
            "expandAllResponses": false,
            "orderSchemaPropertiesBy": "alpha",
            "orderRequiredPropertiesFirst": true
        }'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
