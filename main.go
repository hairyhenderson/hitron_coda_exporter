package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hairyhenderson/hitron_coda_exporter/internal/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	sc = &safeConfig{
		C: &config{},
	}
	reloadCh chan chan error
)

func handler(w http.ResponseWriter, r *http.Request) {
	slog.DebugContext(r.Context(), "Starting scrape")

	start := time.Now()

	sc.RLock()
	conf := *sc.C
	sc.RUnlock()

	registry := prometheus.NewRegistry()
	collector := newCollector(r.Context(), conf)
	registry.MustRegister(collector)

	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)

	duration := time.Since(start).Seconds()
	exporterDurationSummary.Observe(duration)
	exporterDuration.Observe(duration)

	slog.DebugContext(r.Context(), "Finished scrape", slog.Float64("duration_seconds", duration))
}

func handleHUP(configFile string) {
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	reloadCh = make(chan chan error)

	go func() {
		for {
			select {
			case <-hup:
				if err := sc.ReloadConfig(configFile); err != nil {
					slog.Error("Error reloading config", "err", err)
				} else {
					slog.Info("Loaded config file")
				}
			case rc := <-reloadCh:
				if err := sc.ReloadConfig(configFile); err != nil {
					slog.Error("Error reloading config", "err", err)
					rc <- err
				} else {
					slog.Info("Loaded config file")
					rc <- nil
				}
			}
		}
	}()
}

func initRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	// Endpoint to do scrapes.
	mux.HandleFunc("/scrape", func(w http.ResponseWriter, r *http.Request) {
		handler(w, r)
	})
	mux.HandleFunc("/-/reload", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			slog.DebugContext(r.Context(), "Reloading config from HTTP endpoint")

			rc := make(chan error)
			reloadCh <- rc

			if err := <-rc; err != nil {
				http.Error(w, fmt.Sprintf("failed to reload config: %s", err), http.StatusInternalServerError)
			}
		default:
			http.Error(w, "POST method expected", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
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

	level := "info"
	format := "logfmt"
	configFile := "hitron_coda.yml"
	listenAddress := ":9780"

	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Version)
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.Flag("log.level", "log level (debug, info, warn, error)").Default("info").StringVar(&level)
	kingpin.Flag("log.format", "log format (logfmt, json)").Default("logfmt").StringVar(&format)
	kingpin.Flag("config.file", "Path to configuration file.").Default("hitron_coda.yml").StringVar(&configFile)
	kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9780").StringVar(&listenAddress)

	kingpin.Parse()

	initExporterMetrics()

	initLogger(level, format)

	slog.Info("Starting hitron_coda_exporter", "version", version.Version, "commit", version.GitCommit)

	// Bail early if the config is bad.
	err := sc.ReloadConfig(configFile)
	if err != nil {
		slog.Error("Error parsing config file", "err", err)

		exitCode = 1

		return
	}

	handleHUP(configFile)

	mux := initRoutes()

	slog.Info("Listening on", "address", listenAddress)

	srv := &http.Server{
		Addr:    listenAddress,
		Handler: mux,
		//nolint:gomnd
		ReadHeaderTimeout: 2 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Error starting HTTP server", "err", err)

		exitCode = 1
	}
}

func initLogger(level, format string) {
	lvl := slog.LevelInfo

	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}
