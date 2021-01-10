package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	metricsNS = "hitron_coda"
)

var (
	configFile    = kingpin.Flag("config.file", "Path to configuration file.").Default("hitron_coda.yml").String()
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9766").String()
	// dryRun        = kingpin.Flag("dry-run", "Only verify configuration is valid and exit.").Default("false").Bool()

	// Metrics about the exporter itself.
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
	sc = &safeConfig{
		C: &config{},
	}
	reloadCh chan chan error
)

func init() {
	prometheus.MustRegister(exporterDuration, exporterRequestErrors)
	prometheus.MustRegister(version.NewCollector(metricsNS + "_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request, logger log.Logger) {
	level.Debug(logger).Log("msg", "Starting scrape")

	start := time.Now()

	sc.RLock()
	conf := *sc.C
	sc.RUnlock()

	registry := prometheus.NewRegistry()
	collector := newCollector(r.Context(), conf, logger)
	registry.MustRegister(collector)

	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)

	duration := time.Since(start).Seconds()
	exporterDuration.Observe(duration)
	level.Debug(logger).Log("msg", "Finished scrape", "duration_seconds", duration)
}

func updateConfiguration(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		rc := make(chan error)
		reloadCh <- rc

		if err := <-rc; err != nil {
			http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "POST method expected", http.StatusMethodNotAllowed)
	}
}

func handleHUP(logger log.Logger) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	reloadCh = make(chan chan error)

	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "err", err)
				} else {
					level.Info(logger).Log("msg", "Loaded config file")
				}
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(*configFile); err != nil {
					level.Error(logger).Log("msg", "Error reloading config", "err", err)
					rc <- err
				} else {
					level.Info(logger).Log("msg", "Loaded config file")
					rc <- nil
				}
			}
		}
	}()
}

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting hitron_coda_exporter", "version", version.Info())
	level.Info(logger).Log("build_context", version.BuildContext())

	// Bail early if the config is bad.
	err := sc.ReloadConfig(*configFile)
	if err != nil {
		level.Error(logger).Log("msg", "Error parsing config file", "err", err)
		os.Exit(1)
	}

	handleHUP(logger)

	http.Handle("/metrics", promhttp.Handler())
	// Endpoint to do scrapes.
	http.HandleFunc("/scrape", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, logger)
	})
	http.HandleFunc("/-/reload", updateConfiguration) // Endpoint to reload configuration.

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
            <head>
            <title>Hitron CODA Cable Modem Exporter</title>
            <style>
            label{
            display:inline-block;
            width:75px;
            }
            form label {
            margin: 10px;
            }
            form input {
            margin: 10px;
            }
            </style>
            </head>
            <body>
            <h1>Hitron CODA Cable Modem Exporter</h1>
            <form action="/scrape">
            <input type="submit" value="/scrape">
            </form>
            </body>
            </html>`))
	})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)

	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(1)
	}
}
