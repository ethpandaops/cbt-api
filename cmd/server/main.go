package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu-cbt-api/internal/config"
	"github.com/ethpandaops/xatu-cbt-api/internal/server"
)

func main() {
	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config")
	}

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create server")
	}

	// Start server
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Starting server")
		logger.WithField("url", "/health").Info("Health endpoint")
		logger.WithField("url", "/metrics").Info("Metrics endpoint")
		logger.WithField("url", "/docs/").Info("Docs endpoint")
		logger.WithField("url", "/openapi.yaml").Info("OpenAPI spec endpoint")
		logger.Info("Server ready, accepting connections")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Server error")
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server")

	if err := srv.Shutdown(context.Background()); err != nil {
		logger.WithError(err).Error("Server shutdown error")
	}

	logger.Info("Server stopped")
}
