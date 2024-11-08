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

func refreshHelmMetrics(ctx context.Context, errChan chan<- error) error {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case <-ticker.C:
				helmMetrics, err = getHelmStatus()
				if err != nil {
					log.Printf("Error getting helm status: %v", err)
					// Optionally send to error channel if you want to terminate the program
					// errChan <- fmt.Errorf("failed to get helm status: %w", err)
					// return
				}
			}
		}
	}()
	return nil
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	if err := refreshHelmMetrics(ctx, errChan); err != nil {
		log.Fatalf("Failed to start metrics refresh: %v", err)
	}

	if err := exposeMetric(ctx); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}

	select {
	case err := <-errChan:
		log.Printf("Service error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	}

	// Initiate graceful shutdown
	cancel()
	log.Println("Shutting down services...")
	time.Sleep(time.Second * 5) // Give time for cleanup
}
