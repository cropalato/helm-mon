package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cropalato/helm-monitor/internal/helm"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Server represents the metrics server and its dependencies
type Server struct {
	metrics   *Metrics
	helm      *helm.Client
	logger    *zap.Logger
	server    *http.Server
	startTime time.Time
}

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

// MetricsError represents errors that can occur during metrics operations
type MetricsError struct {
	Op         string
	Err        error
	MetricName string
}

func (e *MetricsError) Error() string {
	if e.MetricName != "" {
		return fmt.Sprintf("metrics operation %s failed for metric %s: %v", e.Op, e.MetricName, e.Err)
	}
	return fmt.Sprintf("metrics operation %s failed: %v", e.Op, e.Err)
}
