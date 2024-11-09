package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"

	customerrors "github.com/cropalato/helm-monitor/pkg/errors"
)

// NewManager creates a new repository manager
func NewManager(settings *cli.EnvSettings, logger *zap.Logger) *Manager {
	return &Manager{
		settings: settings,
		logger:   logger,
	}
}

// LoadRepositories loads all configured Helm repositories
func (m *Manager) LoadRepositories() ([]*repo.ChartRepository, error) {
	start := time.Now()
	m.logger.Debug("loading repositories")

	file, err := repo.LoadFile(m.settings.RepositoryConfig)
	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			m.logger.Warn("repository config file does not exist",
				zap.String("path", m.settings.RepositoryConfig))
			return nil, &customerrors.RepositoryError{
				Op:   "load_config",
				Repo: m.settings.RepositoryConfig,
				Err:  err,
			}
		}
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	var repositories []*repo.ChartRepository
	for _, cfg := range file.Repositories {
		r, err := repo.NewChartRepository(cfg, getter.All(m.settings))
		if err != nil {
			m.logger.Warn("failed to load repository index",
				zap.String("repo", cfg.Name),
				zap.Error(err))
			continue
		}
		repositories = append(repositories, r)
	}

	m.logger.Debug("repositories loaded",
		zap.Int("count", len(repositories)),
		zap.Duration("duration", time.Since(start)))

	return repositories, nil
}

// SyncRepositories synchronizes all repository indexes
func (m *Manager) SyncRepositories() (*SyncResult, error) {
	start := time.Now()
	m.logger.Debug("starting repository sync")

	result := &SyncResult{
		Failed: make(map[string]string),
	}

	repos, err := m.LoadRepositories()
	if err != nil {
		return nil, fmt.Errorf("failed to load repositories: %w", err)
	}

	for _, repo := range repos {
		if err := m.syncRepository(repo); err != nil {
			result.Failed[repo.Config.Name] = err.Error()
			continue
		}
		result.Successful = append(result.Successful, repo.Config.Name)
	}

	m.logger.Debug("repository sync completed",
		zap.Int("successful", len(result.Successful)),
		zap.Int("failed", len(result.Failed)),
		zap.Duration("duration", time.Since(start)))

	return result, nil
}

// GetRepositoryInfo returns information about all configured repositories
func (m *Manager) GetRepositoryInfo() ([]RepoInfo, error) {
	repos, err := m.LoadRepositories()
	if err != nil {
		return nil, err
	}

	info := make([]RepoInfo, 0, len(repos))
	for _, r := range repos {
		cacheFile := filepath.Join(m.settings.RepositoryCache, helmpath.CacheIndexFile(r.Config.Name))

		repoInfo := RepoInfo{
			Name:         r.Config.Name,
			URL:          r.Config.URL,
			CacheFile:    cacheFile,
			HasIndexFile: r.IndexFile != nil,
		}

		if r.IndexFile != nil {
			repoInfo.LastSynced = r.IndexFile.Generated
			repoInfo.ChartCount = len(r.IndexFile.Entries)
		}

		info = append(info, repoInfo)
	}

	return info, nil
}

// Private helper methods

func (m *Manager) syncRepository(repo *repo.ChartRepository) error {
	m.logger.Debug("syncing repository",
		zap.String("name", repo.Config.Name),
		zap.String("url", repo.Config.URL))

	// Load the downloaded index
	_, err := repo.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("failed to load repository index: %w", err)
	}

	return nil
}

func (m *Manager) getterProviders() getter.Providers {
	return getter.All(m.settings)
}

func (r *Repository) loadIndex(cacheDir string) error {
	cacheFile := filepath.Join(cacheDir, helmpath.CacheIndexFile(r.Name))
	index, err := repo.LoadIndexFile(cacheFile)
	if err != nil {
		return err
	}
	r.IndexFile = index
	return nil
}
