//
// metrics.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var helm_version_metric []map[string]prometheus.Gauge

func recordMetrics(tmpTest map[string]prometheus.Gauge) {
	go func() {
		for {
			if len(tmpTest) != 0 {
				for _, metric := range helmMetrics {
					for k, v := range metric {
						tmpTest[v.Namespace+"/"+k].Set(v.Overdue)
					}
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
}

func exposeMetric() {
	tmpTest := make(map[string]prometheus.Gauge)
	for {
		if len(helmMetrics) != 0 {
			break
		}
		fmt.Println("Waiting for metrics...")
		time.Sleep(2 * time.Second)
	}
	for _, metric := range helmMetrics {
		for k, v := range metric {
			tmpTest[v.Namespace+"/"+k] = prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        "helm_chart_overdue",
				Help:        "The number of versions the chart is overdue.",
				ConstLabels: prometheus.Labels{"chart": k, "version": v.ChartVersion, "namespace": v.Namespace},
			})
			prometheus.MustRegister(tmpTest[v.Namespace+"/"+k])

		}
	}
	recordMetrics(tmpTest)

	fmt.Println("listening http://0.0.0.0:2112/metrics ...")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
