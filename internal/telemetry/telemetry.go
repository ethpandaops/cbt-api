package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethpandaops/xatu-cbt-api/internal/config"
	"github.com/ethpandaops/xatu-cbt-api/internal/version"
)

// Service defines the telemetry service interface.
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// service implements the Service interface.
type service struct {
	config        *config.TelemetryConfig
	log           logrus.FieldLogger
	traceProvider *sdktrace.TracerProvider
	exporter      sdktrace.SpanExporter
	networkName   string // Database name to use as network label
}

// Compile-time interface compliance check.
var _ Service = (*service)(nil)

// NewService creates a new telemetry service
// Returns the interface, not the struct (ethPandaOps pattern)
// networkName should be the database name from ClickHouse config.
func NewService(cfg *config.TelemetryConfig, networkName string, logger logrus.FieldLogger) Service {
	return &service{
		config: cfg,
		log: logger.WithFields(logrus.Fields{
			"module": "telemetry",
		}),
		networkName: networkName,
	}
}

// Start initializes OpenTelemetry SDK.
// Heavy initialization happens here, not in constructor.
func (s *service) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.log.Info("Telemetry disabled, skipping initialization")

		return nil
	}

	s.log.WithFields(logrus.Fields{
		"endpoint":     s.config.Endpoint,
		"service_name": s.config.ServiceName,
		"version":      version.Short(),
		"environment":  s.config.Environment,
		"network_name": s.networkName,
		"sample_rate":  s.config.SampleRate,
	}).Info("Initializing telemetry")

	// Create OTLP exporter
	exporter, err := s.createExporter(ctx)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	s.exporter = exporter

	// Create resource with service identification
	res, err := s.createResource()
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler
	sampler := NewCustomSampler(s.config.SampleRate, s.config.AlwaysSampleErrors)

	// Create trace provider
	s.traceProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithMaxExportBatchSize(s.config.ExportBatchSize),
			sdktrace.WithBatchTimeout(s.config.ExportTimeout),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global trace provider
	otel.SetTracerProvider(s.traceProvider)

	// Set global propagator (W3C Trace Context + Baggage)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	s.log.Info("Telemetry initialized successfully")

	return nil
}

// Stop shuts down telemetry and flushes pending spans.
func (s *service) Stop(ctx context.Context) error {
	if s.traceProvider == nil {
		return nil
	}

	s.log.Info("Shutting down telemetry")

	// Shutdown trace provider (flushes pending spans)
	if err := s.traceProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown trace provider: %w", err)
	}

	s.log.Info("Telemetry shut down successfully")

	return nil
}

// createExporter creates OTLP gRPC exporter with authentication.
func (s *service) createExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	// Strip scheme from endpoint if present (gRPC expects host:port only)
	endpoint := s.config.Endpoint
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	// Build gRPC options
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	// TLS configuration
	if s.config.Insecure {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")))
	}

	// Add authentication headers if configured
	if len(s.config.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(s.config.Headers))
	}

	// Set timeouts
	opts = append(opts, otlptracegrpc.WithTimeout(s.config.ExportTimeout))

	// Create exporter
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return exporter, nil
}

// createResource creates resource with service identification.
func (s *service) createResource() (*resource.Resource, error) {
	return resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(s.config.ServiceName),
			semconv.ServiceVersion(version.Short()),
			AttrNetworkName.String(s.networkName),
		),
	)
}
