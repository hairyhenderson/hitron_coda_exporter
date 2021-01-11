package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hairyhenderson/hitron_coda_exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	configFile    = kingpin.Flag("config.file", "Path to configuration file.").Default("hitron_coda.yml").String()
	listenAddress = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9779").String()
	// dryRun        = kingpin.Flag("dry-run", "Only verify configuration is valid and exit.").Default("false").Bool()

	sc = &safeConfig{
		C: &config{},
	}
	reloadCh chan chan error
)

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

func initRoutes(logger log.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	// Endpoint to do scrapes.
	mux.HandleFunc("/scrape", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, logger)
	})
	mux.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			level.Debug(logger).Log("msg", "Reloading config from HTTP endpoint")

			rc := make(chan error)
			reloadCh <- rc

			if err := <-rc; err != nil {
				http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
			}
		default:
			http.Error(w, "POST method expected", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

func main() {
	exitCode := 0

	defer func() { os.Exit(exitCode) }()

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Version)
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	initExporterMetrics()

	level.Info(logger).Log("msg", "Starting hitron_coda_exporter", "version", version.Version, "commit", version.GitCommit)

	// Bail early if the config is bad.
	err := sc.ReloadConfig(*configFile)
	if err != nil {
		level.Error(logger).Log("msg", "Error parsing config file", "err", err)

		exitCode = 1

		return
	}

	handleHUP(logger)

	mux := initRoutes(logger)

	level.Info(logger).Log("msg", "Listening on", "address", *listenAddress)

	if err := http.ListenAndServe(*listenAddress, mux); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)

		exitCode = 1
	}
}
