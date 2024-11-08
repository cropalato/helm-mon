//
// main.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	//"encoding/json"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Custom error types for better error handling
type (
	// HelmError wraps helm-related errors with context
	HelmError struct {
		Op      string                 // Operation that failed
		Err     error                  // Original error
		Chart   string                 // Chart name if applicable
		Details map[string]interface{} // Additional context
	}

	// RepositoryError wraps repository-related errors
	RepositoryError struct {
		Op   string
		Repo string
		Err  error
	}

	// MetricsError wraps metrics-related errors
	MetricsError struct {
		Op         string
		Err        error
		MetricName string
	}
	chartOverdue struct {
		ChartVersion string  `json:"chart_version"`
		Namespace    string  `json:"namespace"`
		Overdue      float64 `json:"n_overdue"`
	}
)

// Error implementations
func (e *HelmError) Error() string {
	if e.Chart != "" {
		return fmt.Sprintf("helm operation %s failed for chart %s: %v", e.Op, e.Chart, e.Err)
	}
	return fmt.Sprintf("helm operation %s failed: %v", e.Op, e.Err)
}

func (e *RepositoryError) Error() string {
	return fmt.Sprintf("repository operation %s failed for repo %s: %v", e.Op, e.Repo, e.Err)
}

func (e *MetricsError) Error() string {
	return fmt.Sprintf("metrics operation %s failed for %s: %v", e.Op, e.MetricName, e.Err)
}

// Logger initialization with custom configuration
func initLogger(debug bool) (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	// Customize logging configuration
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"

	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(
			zap.String("service", "helm-monitor"),
			zap.String("version", "1.0.0"),
		),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize logger")
	}

	return logger, nil
}

var settings = cli.New()
var helmMetrics []map[string]chartOverdue
var err error

func listRepos() error {

	f, err := repo.LoadFile(cli.New().RepositoryConfig)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}
	if len(f.Repositories) == 0 {
		return fmt.Errorf("no repositories configured")
	}
	for _, repo := range f.Repositories {
		log.Printf("%s - %s\n", repo.Name, repo.URL)
	}
	return nil
}

func elementExists(list []string, item string) bool {
	for _, tmp := range list {
		if tmp == item {
			return true
		}
	}
	return false
}

func refreshHelmMetrics(ctx context.Context, logger *zap.Logger, errChan chan<- error) error {
	logger.Info("starting metrics refresh routine")
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("stopping metrics refresh routine",
					zap.String("reason", "context cancelled"))
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				start := time.Now()
				logger.Debug("refreshing helm metrics")
				status, err := getHelmStatus(logger)
				if err != nil {
					recordOperationError("refresh_metrics", "status_failed")
					logger.Error("failed to refresh metrics",
						zap.Error(err),
						zap.Time("timestamp", time.Now()))
					// Record metric for failed refresh
					continue
				}
				helmMetrics = status

				// Update overdue metrics for each chart
				for _, metric := range status {
					for chartName, chartData := range metric {
						recordChartOverdue(
							chartName,
							chartData.Namespace,
							chartData.ChartVersion,
							chartData.Overdue,
						)
					}
				}
				recordOperationDuration("refresh_metrics", time.Since(start).Seconds())
				logger.Debug("metrics refresh completed",
					zap.Int("metric_count", len(status)),
					zap.Time("timestamp", time.Now()))
			}
		}
	}()
	return nil
}

func getHelmStatus(logger *zap.Logger) ([]map[string]chartOverdue, error) {
	var chartVersions []map[string][]chartVersion
	var chartList []string
	var count float64
	var chartStatus chartOverdue
	var tmpHelmMetrics []map[string]chartOverdue
	var newStatus map[string]chartOverdue

	//Get list with all installed charts
	items, err := listCharts(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to list charts: %w", err)
	}
	for _, item := range items {
		if !elementExists(chartList, item.ChartName) {
			chartList = append(chartList, item.ChartName)
		}
	}

	//Get all versions charts available
	chartVersions, err = searchChartVersions(logger, chartList)
	if err != nil {
		return nil, fmt.Errorf("failed to search chart versions: %w", err)
	}
	//s, _ := json.MarshalIndent(chartVersions, "", "  ")
	//fmt.Printf("%s\n", string(s))

	// check the versions
	for _, tmp := range chartVersions {
		for key, value := range tmp {
			for _, chart := range items {
				if chart.ChartName == key {
					constraint, err := semver.NewConstraint(">" + chart.ChartVersion)
					if err != nil {
						log.Fatal(err, "an invalid version/constraint format")
					}
					count = 0
					if value != nil {
						for _, version := range value {
							v, err := semver.NewVersion(version.ChartVersion)
							if err != nil {
								log.Fatal(err, "an invalid version/constraint format")
							}
							if constraint.Check(v) {
								count++
							}
						}
						chartStatus.Overdue = count
					} else {
						chartStatus.Overdue = -1
					}
					chartStatus.ChartVersion = chart.ChartVersion
					chartStatus.Namespace = chart.Namespace
					newStatus = map[string]chartOverdue{
						chart.ChartName: chartStatus,
					}
					tmpHelmMetrics = append(tmpHelmMetrics, newStatus)
				}
			}
		}
	}
	return tmpHelmMetrics, nil
}

func main() {
	// Initialize logger
	logger, err := initLogger(settings.Debug)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("starting helm-monitor service",
		zap.Bool("debug", settings.Debug))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	if err := refreshHelmMetrics(ctx, logger, errChan); err != nil {
		logger.Fatal("failed to start metrics refresh",
			zap.Error(err))
	}

	if err := exposeMetric(ctx, logger); err != nil {
		logger.Fatal("failed to start metrics server",
			zap.Error(err))
	}

	select {
	case err := <-errChan:
		logger.Error("Service error",
			zap.Error(err),
			zap.Time("timestamp", time.Now()))
	case sig := <-sigChan:
		logger.Debug("Received signal",
			zap.String("signal", sig.String()),
			zap.Time("timestamp", time.Now()))
	}

	logger.Info("helm-monitor service started successfully")

	// Initiate graceful shutdown
	cancel()
	logger.Info("Shutting down services...")
	time.Sleep(time.Second * 5) // Give time for cleanup
}
