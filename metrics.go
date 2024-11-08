//
// metrics.go
// Copyright (C) 2021 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var helm_version_metric []map[string]prometheus.Gauge

func recordMetrics(ctx context.Context, tmpTest map[string]prometheus.Gauge) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(tmpTest) != 0 {
				for _, metric := range helmMetrics {
					for k, v := range metric {
						tmpTest[v.Namespace+"/"+k].Set(v.Overdue)
					}
				}
			}
		}
	}
}

func exposeMetric(ctx context.Context) error {
	tmpTest := make(map[string]prometheus.Gauge)

	timeout := time.After(time.Minute * 2)
	for {
		if len(helmMetrics) != 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for initial metrics")
		default:
			log.Println("Waiting for metrics...")
			time.Sleep(2 * time.Second)
		}
	}
	for _, metric := range helmMetrics {
		for k, v := range metric {
			metricName := v.Namespace + "/" + k
			tmpTest[metricName] = prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "helm_chart_overdue",
				Help: "The number of versions the chart is overdue.",
				ConstLabels: prometheus.Labels{
					"chart":     k,
					"version":   v.ChartVersion,
					"namespace": v.Namespace,
				},
			})
			if err := prometheus.Register(tmpTest[metricName]); err != nil {
				return fmt.Errorf("failed to register metric %s: %w", metricName, err)
			}
		}
	}
	go recordMetrics(ctx, tmpTest)

	fmt.Println("listening http://0.0.0.0:2112/metrics ...")
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    ":2112",
		Handler: mux,
	}

	// Channel to capture server errors
	serverError := make(chan error, 1)

	go func() {
		log.Println("listening http://0.0.0.0:2112/metrics ...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverError <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverError:
		return err
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down metrics server: %w", err)
		}
		return ctx.Err()
	}
}
