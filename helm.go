//
// helm.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	//"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"helm.sh/helm/v3/cmd/helm/search"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

type releaseElement struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Revision     string `json:"revision"`
	Updated      string `json:"updated"`
	Status       string `json:"status"`
	ChartName    string `json:"chart_name"`
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`
}

type chartVersion struct {
	ChartVersion string `json:"chart_version"`
	AppVersion   string `json:"app_version"`
}
type chartInfo struct {
	ChartName string         `json:"name"`
	Versions  []chartVersion `json:"versions"`
}

type releaseListWriter struct {
	releases []releaseElement
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}

func listCharts(logger *zap.Logger) ([]releaseElement, error) {
	cfg := new(action.Configuration)
	client := action.NewList(cfg)
	if err := cfg.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
		logger.Error("failed to initialize helm configuration",
			zap.Error(err),
			zap.String("namespace", ""),
			zap.String("driver", os.Getenv("HELM_DRIVER")))
		return nil, &HelmError{
			Op:  "init_config",
			Err: err,
			Details: map[string]interface{}{
				"driver": os.Getenv("HELM_DRIVER"),
			},
		}
	}
	client.SetStateMask()
	results, err := client.Run()
	if err != nil {
		logger.Error("failed to list releases",
			zap.Error(err))
		return nil, &HelmError{
			Op:  "list_releases",
			Err: err,
		}
	}
	logger.Info("successfully listed helm releases",
		zap.Int("count", len(results)))
	elements := make([]releaseElement, 0, len(results))
	for _, r := range results {
		element := releaseElement{
			Name:         r.Name,
			Namespace:    r.Namespace,
			Revision:     strconv.Itoa(r.Version),
			Status:       r.Info.Status.String(),
			ChartName:    r.Chart.Metadata.Name,
			ChartVersion: r.Chart.Metadata.Version,
			AppVersion:   r.Chart.Metadata.AppVersion,
		}
		elements = append(elements, element)
		logger.Debug("processed release",
			zap.String("name", element.Name),
			zap.String("namespace", element.Namespace),
			zap.String("version", element.ChartVersion))
	}
	return elements, nil
}

// func searchChartVersions(chartName string) ([]chartInfo, error) {
func searchChartVersions(logger *zap.Logger, charts []string) ([]map[string][]chartVersion, error) {
	logger.Debug("searching chart versions",
		zap.Strings("charts", charts))
	//Load repo file
	rf, err := repo.LoadFile(settings.RepositoryConfig)
	if isNotExist(err) || len(rf.Repositories) == 0 {
		logger.Error("no repositories configured",
			zap.String("config_path", settings.RepositoryConfig))
		return nil, &RepositoryError{
			Op:   "load_repo_file",
			Repo: settings.RepositoryConfig,
			Err:  errors.New("no repositories configured"),
		}
	}

	chartList := make([]map[string][]chartVersion, len(charts))
	//chart := make(map[string][]chartVersion)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := filepath.Join(settings.RepositoryCache, helmpath.CacheIndexFile(n))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			logger.Warn("repository is corrupt or missing",
				zap.String("repo", re.Name),
				zap.String("cache_file", f),
				zap.Error(err))
			continue
		}

		//should replace the condition
		i.AddRepo(n, ind, true)
	}
	res := i.All()
	//search.SortScore(res)
	for i, chartTarget := range charts {
		chartList[i] = map[string][]chartVersion{chartTarget: nil}
		versions := []chartVersion{}

		for _, tmp := range res {
			if tmp.Chart.Name != chartTarget {
				continue
			}
			version := chartVersion{
				AppVersion:   tmp.Chart.AppVersion,
				ChartVersion: tmp.Chart.Version,
			}
			versions = append(versions, version)
		}
		chartList[i][chartTarget] = versions
		logger.Info("processed chart versions",
			zap.String("chart", chartTarget),
			zap.Int("version_count", len(versions)))
	}
	return chartList, nil
}
