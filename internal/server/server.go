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
	"github.com/ethpandaops/cbt-api/internal/middleware/headers"
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

	// Health endpoint
	mux.HandleFunc("GET /health", handlers.Health)

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

	// Initialize headers manager from config
	var headersManager *headers.Manager
	if len(cfg.Headers.Policies) > 0 {
		var err error
		headersManager, err = headers.NewManager(cfg.Headers.Policies)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize headers manager: %w", err)
		}

		logger.WithField("count", len(cfg.Headers.Policies)).Info("initialized headers manager with policies")
	}

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

	handler = middleware.Gzip()(handler)

	// Apply headers middleware if configured
	if headersManager != nil {
		handler = headersManager.Middleware(logger.WithField("component", "headers"))(handler)
	}

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
    <style>
        @keyframes border-glimmer {
            0% { background-position: 0% 0%; }
            100% { background-position: 200% 0%; }
        }

        body {
            margin: 0;
            padding: 0;
            background: #0f172a;
        }

        .custom-header {
            background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
            padding: 32px 20px;
            border-bottom: 1px solid rgba(99, 102, 241, 0.2);
        }

        .header-content {
            max-width: 1400px;
            margin: 0 auto;
            display: flex;
            align-items: center;
            gap: 24px;
        }

        .logo-container {
            position: relative;
            flex-shrink: 0;
        }

        .logo-container .absolute {
            position: absolute;
        }

        .logo-container .inset-0 {
            inset: 0;
        }

        .logo-container .inset-negative {
            inset: -2px;
        }

        .backdrop {
            border-radius: 1rem;
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(12px);
        }

        .gradient-layer {
            border-radius: 1rem;
            background: linear-gradient(135deg, rgba(255, 255, 255, 0.1) 0%, transparent 50%, rgba(255, 255, 255, 0.05) 100%);
        }

        .shadow-layer {
            border-radius: 1rem;
            box-shadow:
                rgba(255, 255, 255, 0.1) 0px 1px 2px 0px inset,
                rgba(0, 0, 0, 0.1) 0px -1px 1px 0px inset;
        }

        .glimmer-wrapper {
            border-radius: 1rem;
            opacity: 0.75;
            transition: opacity 0.3s;
        }

        .logo-container:hover .glimmer-wrapper {
            opacity: 1;
        }

        .glimmer-border {
            border-radius: 1rem;
            padding: 2px;
        }

        .glimmer-border-inner {
            width: 100%;
            height: 100%;
            border-radius: 1rem;
            background: rgba(15, 23, 42, 0.95);
        }

        .glimmer-effect {
            border-radius: 1rem;
            background: linear-gradient(105deg, transparent 40%, rgba(34, 211, 238, 0.3) 50%, transparent 60%) 0% 0% / 200% 200%;
            animation: border-glimmer 3s linear infinite;
            filter: blur(4px);
        }

        .logo-img {
            position: relative;
            width: 80px;
            height: 80px;
            border-radius: 1rem;
            object-fit: contain;
            transition: all 0.5s;
        }

        .logo-container:hover .logo-img {
            transform: scale(1.05);
        }

        .shimmer-overlay {
            pointer-events: none;
            border-radius: 1rem;
            opacity: 0;
            transition: opacity 0.5s;
            background: linear-gradient(105deg, transparent 40%, rgba(255, 255, 255, 0.1) 50%, transparent 60%) 0% 0% / 200% 200%;
            animation: border-glimmer 2s linear infinite;
        }

        .logo-container:hover .shimmer-overlay {
            opacity: 1;
        }

        .header-text {
            text-align: left;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
        }

        .header-title {
            font-size: 32px;
            font-weight: 700;
            color: #f1f5f9;
            margin: 0 0 8px 0;
            letter-spacing: -0.5px;
            font-family: inherit;
        }

        .header-subtitle {
            font-size: 16px;
            color: #94a3b8;
            margin: 0;
            font-weight: 400;
            font-family: inherit;
        }

        #api-reference {
            height: calc(100vh - 164px);
        }

        @media (max-width: 768px) {
            .header-content {
                flex-direction: column;
                gap: 16px;
            }
            .header-title {
                font-size: 24px;
            }
            .header-subtitle {
                font-size: 14px;
            }
            .logo-img {
                width: 64px;
                height: 64px;
            }
            #api-reference {
                height: calc(100vh - 180px);
            }
        }
    </style>
</head>
<body>
    <div class="custom-header">
        <div class="header-content">
            <div class="logo-container">
                <div class="absolute inset-0 backdrop"></div>
                <div class="absolute inset-0 gradient-layer"></div>
                <div class="absolute inset-0 shadow-layer"></div>
                <div class="absolute inset-negative glimmer-wrapper">
                    <div class="glimmer-border">
                        <div class="glimmer-border-inner"></div>
                    </div>
                    <div class="absolute inset-0 glimmer-effect"></div>
                </div>
                <img class="logo-img" alt="CBT Logo" src="https://cbt.mainnet.ethpandaops.io/logo.png">
                <div class="absolute inset-0 shimmer-overlay"></div>
            </div>
            <div class="header-text">
                <h1 class="header-title">CBT API Documentation</h1>
                <p class="header-subtitle">ClickHouse-based REST APIs via CBT (ClickHouse Build Tool)</p>
            </div>
        </div>
    </div>
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

// NewMetrics creates a dedicated metrics HTTP server.
func NewMetrics(cfg *config.Config) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())

	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.MetricsPort),
		Handler:           mux,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}
}
