//
// main.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	//"encoding/json"
	"github.com/Masterminds/semver/v3"
	"log"
	"time"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type chartOverdue struct {
	ChartVersion string  `json:"chart_version"`
	Namespace    string  `json:"namespace"`
	Overdue      float64 `json:"n_overdue"`
}

var settings = cli.New()
var helmMetrics []map[string]chartOverdue
var err error

func listRepos() {

	f, err := repo.LoadFile(cli.New().RepositoryConfig)
	if err == nil && len(f.Repositories) > 0 {
		for _, repo := range f.Repositories {
			log.Printf("%s - %s\n", repo.Name, repo.URL)
		}
	}
}

func elementExists(list []string, item string) bool {
	for _, tmp := range list {
		if tmp == item {
			return true
		}
	}
	return false
}

func refreshHelmMetrics() {
	go func() {
		for {
			helmMetrics, err = getHelmStatus()
			if err != nil {
				log.Fatal("Error running getHelmStatus().")
			}
			time.Sleep(20 * time.Second)
		}
	}()
}

func getHelmStatus() ([]map[string]chartOverdue, error) {
	var chartVersions []map[string][]chartVersion
	var chartList []string
	var count float64
	var chartStatus chartOverdue
	var tmpHelmMetrics []map[string]chartOverdue
	var newStatus map[string]chartOverdue

	//Get list with all installed charts
	items, _ := listCharts()
	for _, item := range items {
		if !elementExists(chartList, item.ChartName) {
			chartList = append(chartList, item.ChartName)
		}
	}

	//Get all versions charts available
	chartVersions, _ = searchChartVersions(chartList)
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
	log.Print("Starting the service ...")
	refreshHelmMetrics()
	exposeMetric()
}
