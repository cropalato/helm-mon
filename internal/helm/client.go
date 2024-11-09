package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"helm.sh/helm/v3/cmd/helm/search"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"

	customerrors "github.com/cropalato/helm-monitor/pkg/errors"
)

// NewClient creates a new Helm client instance
func NewClient(settings *cli.EnvSettings, logger *zap.Logger) *Client {
	return &Client{
		settings: settings,
		logger:   logger,
	}
}

// ListCharts returns a list of all installed Helm releases
func (c *Client) ListCharts(filter *ReleaseFilter) ([]ReleaseElement, error) {
	cfg := new(action.Configuration)

	if err := cfg.Init(c.settings.RESTClientGetter(), filter.Namespace, os.Getenv("HELM_DRIVER"), zap.S().Debugf); err != nil {
		c.logger.Error("failed to initialize helm configuration",
			zap.Error(err),
			zap.String("namespace", filter.Namespace),
			zap.String("driver", os.Getenv("HELM_DRIVER")))
		return nil, &customerrors.HelmError{
			Op:  "init_config",
			Err: err,
			Details: map[string]interface{}{
				"namespace": filter.Namespace,
				"driver":    os.Getenv("HELM_DRIVER"),
			},
		}
	}

	client := action.NewList(cfg)
	client.SetStateMask()

	releases, err := client.Run()
	if err != nil {
		c.logger.Error("failed to list releases",
			zap.Error(err))
		return nil, &customerrors.HelmError{
			Op:  "list_releases",
			Err: err,
		}
	}

	elements := make([]ReleaseElement, 0, len(releases))
	for _, r := range releases {
		// Apply filters
		if !c.shouldIncludeRelease(r.Name, r.Chart.Metadata.Name, filter) {
			continue
		}

		element := ReleaseElement{
			Name:         r.Name,
			Namespace:    r.Namespace,
			Revision:     strconv.Itoa(r.Version),
			Status:       r.Info.Status.String(),
			ChartName:    r.Chart.Metadata.Name,
			ChartVersion: r.Chart.Metadata.Version,
			AppVersion:   r.Chart.Metadata.AppVersion,
		}
		elements = append(elements, element)

		c.logger.Debug("processed release",
			zap.String("name", element.Name),
			zap.String("namespace", element.Namespace),
			zap.String("version", element.ChartVersion))
	}

	return elements, nil
}

// SearchChartVersions searches for available versions of specified charts
func (c *Client) SearchChartVersions(charts []string) ([]map[string][]ChartVersion, error) {
	c.logger.Debug("searching chart versions",
		zap.Strings("charts", charts))

	rf, err := repo.LoadFile(c.settings.RepositoryConfig)
	if isNotExist(err) || len(rf.Repositories) == 0 {
		c.logger.Error("no repositories configured",
			zap.String("config_path", c.settings.RepositoryConfig))
		return nil, &customerrors.RepositoryError{
			Op:   "load_repo_file",
			Repo: c.settings.RepositoryConfig,
			Err:  errors.New("no repositories configured"),
		}
	}

	chartList := make([]map[string][]ChartVersion, len(charts))
	i := search.NewIndex()

	// Load repository indexes
	for _, re := range rf.Repositories {
		n := re.Name
		f := filepath.Join(c.settings.RepositoryCache, helmpath.CacheIndexFile(n))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			c.logger.Warn("repository is corrupt or missing",
				zap.String("repo", re.Name),
				zap.String("cache_file", f),
				zap.Error(err))
			continue
		}
		i.AddRepo(n, ind, true)
	}

	res := i.All()
	for i, chartTarget := range charts {
		chartList[i] = map[string][]ChartVersion{chartTarget: nil}
		versions := []ChartVersion{}

		for _, tmp := range res {
			if tmp.Chart.Name != chartTarget {
				continue
			}
			version := ChartVersion{
				AppVersion:   tmp.Chart.AppVersion,
				ChartVersion: tmp.Chart.Version,
			}
			versions = append(versions, version)
		}
		chartList[i][chartTarget] = versions
		c.logger.Info("processed chart versions",
			zap.String("chart", chartTarget),
			zap.Int("version_count", len(versions)))
	}

	return chartList, nil
}

// GetChartStatus returns the current status of installed charts
func (c *Client) GetChartStatus(filter *ReleaseFilter) ([]ChartStatus, error) {
	releases, err := c.ListCharts(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list charts: %w", err)
	}

	// Get unique chart names
	chartNames := make(map[string]bool)
	for _, release := range releases {
		chartNames[release.ChartName] = true
	}

	// Get available versions for all charts
	charts := make([]string, 0, len(chartNames))
	for chartName := range chartNames {
		charts = append(charts, chartName)
	}

	versions, err := c.SearchChartVersions(charts)
	if err != nil {
		return nil, fmt.Errorf("failed to search chart versions: %w", err)
	}

	// Process status for each chart
	status := make([]ChartStatus, 0, len(charts))
	for _, chartVersions := range versions {
		for chartName, chartVers := range chartVersions {
			chartStatus := c.processChartStatus(chartName, chartVers, releases)
			status = append(status, chartStatus)
		}
	}

	return status, nil
}

// Helper functions

func (c *Client) processChartStatus(chartName string, versions []ChartVersion, releases []ReleaseElement) ChartStatus {
	status := ChartStatus{
		Name: chartName,
	}

	// Find latest version
	var latestVersion *semver.Version
	for _, ver := range versions {
		v, err := semver.NewVersion(ver.ChartVersion)
		if err != nil {
			continue
		}
		if latestVersion == nil || v.GreaterThan(latestVersion) {
			latestVersion = v
			status.LatestVersion = ver.ChartVersion
		}
	}

	// Process releases using this chart
	for _, release := range releases {
		if release.ChartName != chartName {
			continue
		}

		// Update current version if not set
		if status.CurrentVersion == "" {
			status.CurrentVersion = release.ChartVersion
		}

		constraint, err := semver.NewConstraint(">" + release.ChartVersion)
		if err != nil {
			c.logger.Error("Invalid version format",
				zap.Error(err),
				zap.String("namespace", release.Namespace),
				zap.String("chart", release.ChartName))

		}
		count := float64(0)
		for _, ver := range versions {
			v, err := semver.NewVersion(ver.ChartVersion)
			if err != nil {
				continue
			}
			if constraint.Check(v) {
				count++
			}
		}
		// Add release info
		releaseInfo := ReleaseInfo{
			ReleaseName: release.Name,
			Namespace:   release.Namespace,
			Version:     release.ChartVersion,
			Status:      release.Status,
			Overdue:     count,
		}
		status.Releases = append(status.Releases, releaseInfo)

		// Check if outdated
		currentVer, err := semver.NewVersion(release.ChartVersion)
		if err == nil && latestVersion != nil {
			if currentVer.LessThan(latestVersion) {
				status.IsOutdated = true
			}
		}
	}

	return status
}

func (c *Client) shouldIncludeRelease(releaseName, chartName string, filter *ReleaseFilter) bool {
	if filter == nil {
		return true
	}

	// Check excluded names
	for _, excluded := range filter.ExcludeNames {
		if strings.Contains(releaseName, excluded) {
			return false
		}
	}

	// Check included chart names
	if len(filter.ChartNames) > 0 {
		for _, included := range filter.ChartNames {
			if chartName == included {
				return true
			}
		}
		return false
	}

	return true
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}

// func debug(format string, v ...interface{}) {
// 	if settings.Debug {
// 		format = fmt.Sprintf("[debug] %s\n", format)
// 		log.Output(2, fmt.Sprintf(format, v...))
// 	}
// }
