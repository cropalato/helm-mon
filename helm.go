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

func listCharts() ([]releaseElement, error) {
	cfg := new(action.Configuration)
	client := action.NewList(cfg)
	if err := cfg.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
		return nil, err
	}
	client.SetStateMask()
	results, err := client.Run()
	if err != nil {
		return nil, err
	}
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
	}
	return elements, nil
}

//func searchChartVersions(chartName string) ([]chartInfo, error) {
func searchChartVersions(charts []string) ([]map[string][]chartVersion, error) {
	//Load repo file
	rf, err := repo.LoadFile(settings.RepositoryConfig)
	if isNotExist(err) || len(rf.Repositories) == 0 {
		return nil, errors.New("no repositories configured")
	}

	var chartList = make([]map[string][]chartVersion, len(charts))
	var version chartVersion
	//chart := make(map[string][]chartVersion)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		f := filepath.Join(settings.RepositoryCache, helmpath.CacheIndexFile(n))
		ind, err := repo.LoadIndexFile(f)
		if err != nil {
			warning("Repo %q is corrupt or missing. Try 'helm repo update'.", n)
			warning("%s", err)
			continue
		}

		//should replace the condition
		i.AddRepo(n, ind, true)
	}
	res := i.All()
	//search.SortScore(res)
	for i, chartTarget := range charts {
		chartList[i] = map[string][]chartVersion{chartTarget: nil}
		for _, tmp := range res {
			if tmp.Chart.Name != chartTarget {
				continue
			}
			version.AppVersion = tmp.Chart.AppVersion
			version.ChartVersion = tmp.Chart.Version
			chartList[i][chartTarget] = append(chartList[i][chartTarget], version)
			/*
				chart[tmp.Chart.Name]
				chart.Versions = append(chart.Versions, version)
				chartList = append(chartList, chart)
			*/
		}
	}
	return chartList, nil
}
