// metrics.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics represents all Prometheus metrics for the application
type Metrics struct {
	// Operation metrics
	operationErrors   *prometheus.CounterVec
	operationDuration *prometheus.HistogramVec

	// Helm metrics
	chartVersions *prometheus.GaugeVec
	chartOverdue  *prometheus.GaugeVec

	// Repository metrics
	repoErrors   *prometheus.CounterVec
	lastRepoSync *prometheus.GaugeVec

	// Server metrics
	serverUptime prometheus.Gauge
}

// Global metrics instance
var metrics *Metrics

// newMetrics creates and registers all metrics
func newMetrics() *Metrics {
	return &Metrics{
		operationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "helm_monitor",
				Name:      "operation_errors_total",
				Help:      "Total number of failed operations",
			},
			[]string{"operation", "error_type"},
		),

		operationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "helm_monitor",
				Name:      "operation_duration_seconds",
				Help:      "Duration of operations in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"operation"},
		),

		chartVersions: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "helm_monitor",
				Name:      "chart_versions_available",
				Help:      "Number of available versions for each chart",
			},
			[]string{"chart", "namespace"},
		),

		chartOverdue: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "helm_monitor",
				Name:      "chart_versions_overdue",
				Help:      "Number of versions a chart is behind latest",
			},
			[]string{"chart", "namespace", "current_version"},
		),

		repoErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "helm_monitor",
				Name:      "repo_errors_total",
				Help:      "Total number of repository operation errors",
			},
			[]string{"repo", "operation"},
		),

		lastRepoSync: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "helm_monitor",
				Name:      "repo_last_sync_timestamp",
				Help:      "Timestamp of last successful repository sync",
			},
			[]string{"repo"},
		),

		serverUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "helm_monitor",
				Name:      "server_uptime_seconds",
				Help:      "Time since the server started in seconds",
			},
		),
	}
}

// Initialize metrics in init()
func init() {
	metrics = newMetrics()
}

// Helper functions for recording metrics
func recordOperationError(operation, errorType string) {
	metrics.operationErrors.WithLabelValues(operation, errorType).Inc()
}

func recordOperationDuration(operation string, duration float64) {
	metrics.operationDuration.WithLabelValues(operation).Observe(duration)
}

func recordChartVersions(chart, namespace string, count float64) {
	metrics.chartVersions.WithLabelValues(chart, namespace).Set(count)
}

func recordChartOverdue(chart, namespace, currentVersion string, overdueCount float64) {
	metrics.chartOverdue.WithLabelValues(chart, namespace, currentVersion).Set(overdueCount)
}

func recordRepoError(repo, operation string) {
	metrics.repoErrors.WithLabelValues(repo, operation).Inc()
}

func recordRepoSync(repo string, timestamp float64) {
	metrics.lastRepoSync.WithLabelValues(repo).Set(timestamp)
}

// updateUptimeMetric updates the server uptime metric
func updateUptimeMetric(startTime time.Time) {
	metrics.serverUptime.Set(time.Since(startTime).Seconds())
}

// recordMetrics periodically updates dynamic metrics
func recordMetrics(ctx context.Context, logger *zap.Logger, startTime time.Time) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updateUptimeMetric(startTime)

			// Update chart metrics
			if len(helmMetrics) != 0 {
				for _, metric := range helmMetrics {
					for chartName, chartData := range metric {
						recordChartOverdue(
							chartName,
							chartData.Namespace,
							chartData.ChartVersion,
							chartData.Overdue,
						)
					}
				}
			}
		}
	}
}

// exposeMetric starts the metrics server and handles graceful shutdown
func exposeMetric(ctx context.Context, logger *zap.Logger) error {
	startTime := time.Now()

	// Wait for initial metrics
	timeout := time.After(time.Minute * 2)
	for {
		if len(helmMetrics) != 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for initial metrics")
		default:
			logger.Debug("waiting for initial metrics...")
			time.Sleep(2 * time.Second)
		}
	}

	// Start recording metrics
	go recordMetrics(ctx, logger, startTime)

	// Configure metrics server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if len(helmMetrics) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":2112",
		Handler: mux,
	}

	// Channel to capture server errors
	serverError := make(chan error, 1)

	go func() {
		logger.Info("starting metrics server",
			zap.String("address", "http://0.0.0.0:2112/metrics"))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverError <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverError:
		return err
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		logger.Info("shutting down metrics server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down metrics server: %w", err)
		}
		return ctx.Err()
	}
}
