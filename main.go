//
// main.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"encoding/json"
	"fmt"
	//	"log"
	//	"os"
	//	"path/filepath"
	//	"strconv"

	//	"helm.sh/helm/v3/cmd/helm/search"
	//	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	//	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

/*type releaseElement struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}*/

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

func main() {
	elements, _ := listCharts()
	s, _ := json.MarshalIndent(elements, "", "  ")
	fmt.Printf("%s\n", string(s))

	/*
		rf, err := repo.LoadFile(settings.RepositoryConfig)
		if os.IsNotExist(err) || len(rf.Repositories) == 0 {
			fmt.Println("no repositories configured")
			os.Exit(1)
		}

		i := search.NewIndex()
		for _, re := range rf.Repositories {
			n := re.Name
			f := filepath.Join(settings.RepositoryCache, helmpath.CacheIndexFile(n))
			ind, err := repo.LoadIndexFile(f)
			if err != nil {
				fmt.Printf("Repo %q is corrupt or missing. Try 'helm repo update'.\n", n)
				fmt.Printf("%s\n", err)
				continue
			}

			i.AddRepo(n, ind, true)
		}

		var res []*search.Result
		res = i.All()
		search.SortScore(res)

		for _, r := range res {
			fmt.Printf("%s - %s\n", r.Name, r.Chart.Version)
		}

		listRepos()

		actionConfig := new(action.Configuration)
		// You can pass an empty string instead of settings.Namespace() to list
		// all namespaces
		if err := actionConfig.Init(settings.RESTClientGetter(), "vault", os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
			// if err := actionConfig.Init(settings.RESTClientGetter(), "", "", log.Printf); err != nil {
			log.Printf("%+v", err)
			os.Exit(1)
		}

		client := action.NewList(actionConfig)
		// Only list deployed
		client.Deployed = true
		results, err := client.Run()
		if err != nil {
			log.Printf("%+v", err)
			os.Exit(1)
		}

		for _, r := range results {
			element := releaseElement{
				Name:       r.Name,
				Namespace:  r.Namespace,
				Revision:   strconv.Itoa(r.Version),
				Status:     r.Info.Status.String(),
				Chart:      fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version),
				AppVersion: r.Chart.Metadata.AppVersion,
			}
			b, err := json.MarshalIndent(element, "", "  ")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(string(b))
		}
		// exposeMetric()
	*/
}
