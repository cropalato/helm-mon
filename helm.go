//
// helm.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"helm.sh/helm/v3/pkg/action"
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

type releaseListWriter struct {
	releases []releaseElement
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
