package helm

import (
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/cli"
)

// Client represents a Helm client with configuration and logger
type Client struct {
	settings *cli.EnvSettings
	logger   *zap.Logger
}

// ReleaseElement represents a Helm release with its metadata
type ReleaseElement struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Revision     string `json:"revision"`
	Status       string `json:"status"`
	ChartName    string `json:"chart_name"`
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`
}

// ChartVersion represents version information for a Helm chart
type ChartVersion struct {
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`
}

// ChartOverdue represents version status for a Helm chart
type ChartOverdue struct {
	ChartVersion string  `json:"chart_version"`
	Namespace    string  `json:"namespace"`
	Overdue      float64 `json:"n_overdue"`
}

// ReleaseFilter defines criteria for filtering Helm releases
type ReleaseFilter struct {
	Namespace    string   `json:"namespace"`
	ChartNames   []string `json:"chart_names"`
	ExcludeNames []string `json:"exclude_names"`
}

// ChartStatus represents the current status of a chart
type ChartStatus struct {
	Name           string        `json:"name"`
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version"`
	IsOutdated     bool          `json:"is_outdated"`
	Releases       []ReleaseInfo `json:"releases"`
}

// ReleaseInfo contains information about a specific release
type ReleaseInfo struct {
	ReleaseName  string  `json:"release_name"`
	Namespace    string  `json:"namespace"`
	Version      string  `json:"version"`
	Status       string  `json:"status"`
	LastDeployed string  `json:"last_deployed"`
	Overdue      float64 `json:"n_overdue"`
}
