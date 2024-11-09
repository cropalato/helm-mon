package repository

import (
	"time"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// Manager handles Helm repository operations
type Manager struct {
	settings *cli.EnvSettings
	logger   *zap.Logger
}

// RepoInfo contains information about a Helm repository
type RepoInfo struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	LastSynced   time.Time `json:"last_synced"`
	ChartCount   int       `json:"chart_count"`
	CacheFile    string    `json:"cache_file"`
	HasIndexFile bool      `json:"has_index_file"`
}

// Repository represents a Helm chart repository with its index
type Repository struct {
	*repo.Entry
	IndexFile *repo.IndexFile
}

// SyncResult contains the result of a repository sync operation
type SyncResult struct {
	Successful []string          `json:"successful"`
	Failed     map[string]string `json:"failed"`
}
