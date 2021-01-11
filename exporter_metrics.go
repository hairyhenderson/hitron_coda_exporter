package main

import (
	"fmt"
	"runtime"

	"github.com/hairyhenderson/hitron_coda_exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus"
)

const metricsNS = "hitron_coda"

var (
	// Metrics about the exporter itself.
	buildInfo = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: metricsNS,
			Name:      "build_info",
			Help: fmt.Sprintf(
				"A metric with a constant '1' value labeled by version, revision and goversion from which %s was built.",
				metricsNS,
			),
			ConstLabels: prometheus.Labels{
				"version":   version.Version,
				"revision":  version.GitCommit,
				"goversion": runtime.Version(),
			},
		},
		func() float64 { return 1 },
	)
	exporterDuration = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: metricsNS,
			Name:      "collection_duration_seconds",
			Help:      "Duration of collections by the Hitron CODA exporter",
		},
	)
	exporterRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNS,
			Name:      "request_errors_total",
			Help:      "Errors in requests to the Hitron CODA exporter",
		},
	)
)

func initExporterMetrics() {
	prometheus.MustRegister(buildInfo)
	prometheus.MustRegister(exporterDuration, exporterRequestErrors)
}
