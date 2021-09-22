//
// metrics.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			opsQueued.Set(2)
			time.Sleep(2 * time.Second)
		}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
	opsQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "helm_newer_chart",
		Help:        "Number of blob storage operations waiting to be processed.",
		ConstLabels: prometheus.Labels{"chart": "vault", "version": "1.2.3"},
	})
)

func exposeMetric() {
	prometheus.MustRegister(opsQueued)
	recordMetrics()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
