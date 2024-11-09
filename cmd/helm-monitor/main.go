package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cropalato/helm-monitor/internal/helm"
	"github.com/cropalato/helm-monitor/internal/metrics"
	"github.com/cropalato/helm-monitor/internal/repository"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"helm.sh/helm/v3/pkg/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Config holds the application configuration
type Config struct {
	Debug          bool
	MetricsPort    int
	RefreshRate    time.Duration
	KubeConfig     string
	KubeContext    string
	RepositoryFile string
	CacheDir       string
}

// Application represents the main application instance
type Application struct {
	config    *Config
	logger    *zap.Logger
	settings  *cli.EnvSettings
	helm      *helm.Client
	repo      *repository.Manager
	metrics   *metrics.Server
	stopChan  chan struct{}
	readyChan chan struct{}
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Initialize application
	app, err := NewApplication(config)
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer app.Cleanup()

	// Run the application
	if err := app.Run(); err != nil {
		app.logger.Error("Application failed", zap.Error(err))
		os.Exit(1)
	}
}

// NewApplication creates and initializes a new application instance
func NewApplication(config *Config) (*Application, error) {
	// Initialize logger
	logger, err := initLogger(config.Debug)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize Helm settings
	settings := cli.New()
	settings.Debug = config.Debug
	if config.KubeConfig != "" {
		settings.KubeConfig = config.KubeConfig
	}
	if config.KubeContext != "" {
		settings.KubeContext = config.KubeContext
	}
	if config.RepositoryFile != "" {
		settings.RepositoryConfig = config.RepositoryFile
	}
	if config.CacheDir != "" {
		settings.RepositoryCache = config.CacheDir
	}

	// Initialize components
	helmClient := helm.NewClient(settings, logger)
	repoManager := repository.NewManager(settings, logger)
	metricsServer := metrics.NewServer(helmClient, logger)

	return &Application{
		config:    config,
		logger:    logger,
		settings:  settings,
		helm:      helmClient,
		repo:      repoManager,
		metrics:   metricsServer,
		stopChan:  make(chan struct{}),
		readyChan: make(chan struct{}),
	}, nil
}

// Run starts the application and blocks until shutdown
func (a *Application) Run() error {
	a.logger.Info("Starting helm-monitor",
		zap.Bool("debug", a.config.Debug),
		zap.Int("metrics_port", a.config.MetricsPort),
		zap.Duration("refresh_rate", a.config.RefreshRate))

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start repository manager
	if err := a.startRepositoryManager(ctx); err != nil {
		return fmt.Errorf("failed to start repository manager: %w", err)
	}

	// Start metrics server
	if err := a.metrics.Start(ctx); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	// Signal that the application is ready
	close(a.readyChan)
	a.logger.Info("Application is ready")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		a.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-ctx.Done():
		a.logger.Info("Context cancelled")
	}

	// Initiate graceful shutdown
	return a.shutdown(ctx)
}

// Cleanup performs cleanup operations
func (a *Application) Cleanup() {
	if a.logger != nil {
		a.logger.Sync()
	}
}

// Private helper functions

func (a *Application) startRepositoryManager(ctx context.Context) error {
	// Initial repository sync
	result, err := a.repo.SyncRepositories()
	if err != nil {
		return fmt.Errorf("initial repository sync failed: %w", err)
	}

	a.logger.Info("Initial repository sync completed",
		zap.Int("successful", len(result.Successful)),
		zap.Int("failed", len(result.Failed)))

	// Start periodic sync
	go func() {
		ticker := time.NewTicker(a.config.RefreshRate)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := a.repo.SyncRepositories(); err != nil {
					a.logger.Error("Repository sync failed", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (a *Application) shutdown(ctx context.Context) error {
	a.logger.Info("Initiating graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown metrics server
	if err := a.metrics.Shutdown(shutdownCtx); err != nil {
		a.logger.Error("Failed to shutdown metrics server", zap.Error(err))
	}

	a.logger.Info("Shutdown completed")
	return nil
}

func initLogger(debug bool) (*zap.Logger, error) {
	config := zap.NewProductionConfig()

	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = "stacktrace"
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"

	if debug {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		config.Development = true
	}

	return config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Fields(
			zap.String("service", "helm-monitor"),
			zap.String("version", "1.0.0"),
		),
	)
}

func parseFlags() *Config {
	config := &Config{}

	flag.BoolVar(&config.Debug, "debug", false, "Enable debug logging")
	flag.IntVar(&config.MetricsPort, "metrics-port", 2112, "Metrics server port")
	flag.DurationVar(&config.RefreshRate, "refresh-rate", 20*time.Second, "Metrics refresh rate")
	flag.StringVar(&config.KubeConfig, "kubeconfig", "", "Path to kubeconfig file")
	flag.StringVar(&config.KubeContext, "context", "", "Kubernetes context to use")
	flag.StringVar(&config.RepositoryFile, "repo-file", "", "Path to repositories.yaml file")
	flag.StringVar(&config.CacheDir, "cache-dir", "", "Path to repository cache directory")

	flag.Parse()
	return config
}
