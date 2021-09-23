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
	//"github.com/Masterminds/semver/v3"
	//	"log"
	//"os"
	//	"path/filepath"
	//	"strconv"

	//	"helm.sh/helm/v3/cmd/helm/search"
	//	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	//	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type repositoryElement struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var settings = cli.New()

func listRepos() {

	f, err := repo.LoadFile(cli.New().RepositoryConfig)
	if err == nil && len(f.Repositories) > 0 {
		for _, repo := range f.Repositories {
			fmt.Printf("%s - %s\n", repo.Name, repo.URL)
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

func main() {
	var chartVersions []map[string][]chartVersion
	var chartList []string

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
		for k, v := range tmp {
			if v == nil {
				fmt.Printf("We didn't detected chart versions for %s\n", k)
				continue
			}
			for _, chart := range items {
				if chart.ChartName == k {
					for _, version := range v {
						fmt.Println(version.ChartVersion)
					}
				}
			}
		}
	}
}
