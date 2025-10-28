package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/cbt-api/internal/config"
	"github.com/ethpandaops/cbt-api/internal/server"
	"github.com/ethpandaops/cbt-api/internal/telemetry"
	"github.com/ethpandaops/cbt-api/internal/version"
)

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Log version information
	logger.WithFields(logrus.Fields{
		"version":  version.Short(),
		"platform": version.FullWithPlatform(),
	}).Info("Starting cbt-api")

	// Load config
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}

	// Initialize telemetry (use database name as network label)
	telemetryService := telemetry.NewService(&cfg.Telemetry, cfg.ClickHouse.Database, logger)

	ctx := context.Background()
	if serr := telemetryService.Start(ctx); serr != nil {
		logger.WithError(serr).Fatal("Failed to start telemetry")
	}

	// Ensure telemetry shutdown on exit
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if serr := telemetryService.Stop(shutdownCtx); serr != nil {
			logger.WithError(serr).Error("Failed to stop telemetry")
		}
	}()

	// Create API server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create server")
	}

	// Create metrics server
	metricsSrv := server.NewMetrics(cfg)

	// Start API server
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Starting API server")
		logger.WithField("url", "/health").Info("Health endpoint")
		logger.WithField("url", "/docs/").Info("Docs endpoint")
		logger.WithField("url", "/openapi.yaml").Info("OpenAPI spec endpoint")
		logger.Info("API server ready, accepting connections")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("API server error")
		}
	}()

	// Start metrics server
	go func() {
		logger.WithField("port", cfg.Server.MetricsPort).Info("Starting metrics server")
		logger.WithField("url", "/metrics").Info("Metrics endpoint")

		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Metrics server error")
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down servers")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown API server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("API server shutdown error")
	}

	// Shutdown metrics server
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Metrics server shutdown error")
	}

	logger.Info("Servers stopped")
}
