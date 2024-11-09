package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"
)

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

// Helper functions for recording metrics
func (m *Metrics) recordOperationError(operation, errorType string) {
	m.operationErrors.WithLabelValues(operation, errorType).Inc()
}

func (m *Metrics) recordOperationDuration(operation string, duration float64) {
	m.operationDuration.WithLabelValues(operation).Observe(duration)
}

func (m *Metrics) recordChartVersions(chart, namespace string, count float64) {
	m.chartVersions.WithLabelValues(chart, namespace).Set(count)
}

func (m *Metrics) recordChartOverdue(chart, namespace, currentVersion string, overdueCount float64) {
	m.chartOverdue.WithLabelValues(chart, namespace, currentVersion).Set(overdueCount)
}

func (m *Metrics) recordRepoError(repo, operation string) {
	m.repoErrors.WithLabelValues(repo, operation).Inc()
}

func (m *Metrics) recordRepoSync(repo string, timestamp float64) {
	m.lastRepoSync.WithLabelValues(repo).Set(timestamp)
}

func (m *Metrics) updateUptimeMetric(startTime time.Time) {
	m.serverUptime.Set(time.Since(startTime).Seconds())
}
