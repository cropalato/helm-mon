package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cropalato/helm-monitor/internal/helm"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// NewServer creates a new metrics server instance
func NewServer(helmClient *helm.Client, logger *zap.Logger) *Server {
	return &Server{
		metrics:   newMetrics(),
		helm:      helmClient,
		logger:    logger,
		startTime: time.Now(),
	}
}

// Start initializes and starts the metrics server
func (s *Server) Start(ctx context.Context) error {
	// Create error channel for collecting errors from goroutines
	errChan := make(chan error, 1)

	// Start metrics collection
	if err := s.startMetricsCollection(ctx, errChan); err != nil {
		return fmt.Errorf("failed to start metrics collection: %w", err)
	}

	// Wait for initial metrics
	if err := s.waitForInitialMetrics(ctx); err != nil {
		return err
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", s.healthHandler)

	s.server = &http.Server{
		Addr:    ":2112",
		Handler: mux,
	}

	// Start HTTP server
	go func() {
		s.logger.Info("starting metrics server",
			zap.String("address", "http://0.0.0.0:2112/metrics"))

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Monitor for errors
	go func() {
		select {
		case err := <-errChan:
			s.logger.Error("metrics server error", zap.Error(err))
		case <-ctx.Done():
			return
		}
	}()

	return nil
}

// Shutdown gracefully stops the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down metrics server")
	return s.server.Shutdown(ctx)
}

// startMetricsCollection starts the background metrics collection
func (s *Server) startMetricsCollection(ctx context.Context, errChan chan<- error) error {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				s.logger.Info("stopping metrics collection",
					zap.String("reason", "context cancelled"))
				return
			case <-ticker.C:
				if err := s.collectMetrics(); err != nil {
					s.logger.Error("failed to collect metrics",
						zap.Error(err))
					s.metrics.recordOperationError("collect_metrics", "collection_failed")
				}
			}
		}
	}()

	return nil
}

// collectMetrics collects and updates all metrics
func (s *Server) collectMetrics() error {
	start := time.Now()
	s.logger.Debug("starting metrics collection")

	// Update uptime metric
	s.metrics.updateUptimeMetric(s.startTime)

	// Collect Helm metrics
	status, err := s.helm.GetChartStatus(&helm.ReleaseFilter{Namespace: "*"})
	if err != nil {
		return fmt.Errorf("failed to get helm status: %w", err)
	}

	// Update metrics based on status
	for _, metric := range status {
		for _, r := range metric.Releases {
			s.metrics.recordChartOverdue(
				metric.Name,
				r.Namespace,
				r.Version,
				r.Overdue,
			)
		}
	}

	// Record collection duration
	s.metrics.recordOperationDuration("collect_metrics", time.Since(start).Seconds())

	s.logger.Debug("metrics collection completed",
		zap.Int("metric_count", len(status)),
		zap.Duration("duration", time.Since(start)))

	return nil
}

// waitForInitialMetrics waits for the first metrics collection
func (s *Server) waitForInitialMetrics(ctx context.Context) error {
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// Try collecting metrics
		if err := s.collectMetrics(); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for initial metrics")
		case <-ticker.C:
			s.logger.Info("waiting for initial metrics...")
		}
	}
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Perform a metrics collection to check health
	if err := s.collectMetrics(); err != nil {
		s.logger.Error("health check failed", zap.Error(err))
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
