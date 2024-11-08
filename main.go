//
// main.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	//"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Masterminds/semver/v3"

	"golang.org/x/exp/constraints"
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

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

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

func refreshHelmMetrics() {
	go func() {
		backoff := time.Second * 5
		maxBackoff := time.Minute * 5
		for {
			helmMetrics, err = getHelmStatus()
			if err != nil {
				log.Printf("Error running getHelmStatus(): %v. Retrying in %v", err, backoff)
				time.Sleep(backoff)
				backoff = min(backoff*2, maxBackoff)
				continue
			}
			backoff = time.Second * 5
			time.Sleep(backoff)
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
	items, err := listCharts()
	if err != nil {
		return nil, fmt.Errorf("failed to list charts: %w", err)
	}
	for _, item := range items {
		if !elementExists(chartList, item.ChartName) {
			chartList = append(chartList, item.ChartName)
		}
	}

	//Get all versions charts available
	chartVersions, err = searchChartVersions(chartList)
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
	log.Print("Starting the service ...")
	refreshHelmMetrics()
	exposeMetric()
	// errChan := make(chan error, 1)
	// go func() {
	// 	if err := refreshHelmMetrics(); err != nil {
	// 		errChan <- fmt.Errorf("metrics refresh failed: %w", err)
	// 	}
	// }()

	// go func() {
	// 	if err := exposeMetric(); err != nil {
	// 		errChan <- fmt.Errorf("metrics server failed: %w", err)
	// 	}
	// }()

	// select {
	// case err := <-errChan:
	// 	log.Fatalf("Service error: %v", err)
	// }
}
