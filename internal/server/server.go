package server

import (
	"embed"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/ethpandaops/xatu-cbt-api/internal/config"
	"github.com/ethpandaops/xatu-cbt-api/internal/database"
	"github.com/ethpandaops/xatu-cbt-api/internal/handlers"
	"github.com/ethpandaops/xatu-cbt-api/internal/middleware"
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

	// Create server implementation (generated in Plan 2.5)
	impl := &Server{
		db:     db,
		config: cfg,
	}

	// Setup router using Go 1.22+ http.ServeMux with method routing
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

	// Swagger UI at /docs
	mux.Handle("GET /docs/", httpSwagger.Handler(
		httpSwagger.URL("/openapi.yaml"),
	))

	// Register generated API handlers (from Plan 2)
	// oapi-codegen generates a HandlerFromMux function that works with http.ServeMux
	handlers.HandlerFromMux(impl, mux)

	// Apply middleware stack (wrap the mux)
	handler := middleware.Logging(logger)(mux)
	handler = middleware.CORS()(handler)
	handler = middleware.Recovery(logger)(handler)
	handler = middleware.Metrics()(handler)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: handler,
	}, nil
}
