package main

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ctx    context.Context
	logger log.Logger
	client *hitron.CableModem
	rc     routerCollector
	cc     cmCollector
	wc     wifiCollector

	up prometheus.Gauge

	config config
}

type debugLogAdapter struct {
	log.Logger
}

func (l debugLogAdapter) Logf(format string, args ...interface{}) {
	l.Log("msg", fmt.Sprintf(format, args...))
}

func newCollector(ctx context.Context, conf config, logger log.Logger) *collector {
	debugLogger := debugLogAdapter{level.Debug(logger)}
	ctx = hitron.ContextWithDebugLogger(ctx, debugLogger)

	c := &collector{ctx: ctx, config: conf, logger: logger}
	c.rc = newRouterCollector(ctx, logger, c.getClient)
	c.cc = newCMCollector(ctx, logger, c.getClient)
	c.wc = newWiFiCollector(ctx, logger, c.getClient)

	c.up = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Name:      "up",
		Help:      "Whether the device is reachable (1), or not (0)",
	})

	return c
}

func (c *collector) getClient() *hitron.CableModem {
	return c.client
}

// Describe implements Prometheus.Collector.
func (c collector) Describe(ch chan<- *prometheus.Desc) {
	c.rc.Describe(ch)
	c.cc.Describe(ch)
	c.wc.Describe(ch)

	c.up.Describe(ch)
}

// Collect implements Prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	// Assume the worst...
	c.up.Set(0)
	defer c.up.Collect(ch)

	var err error
	c.client, err = hitron.New(c.config.Host, c.config.Username, c.config.Password)

	if err != nil {
		level.Error(c.logger).Log("msg", "Error scraping target", "err", err)
		exporterClientErrors.Inc()

		return
	}

	err = c.client.Login(c.ctx)
	if err != nil {
		level.Error(c.logger).Log("msg", "Error scraping target", "err", err)
		exporterClientErrors.Inc()

		return
	}

	defer c.client.Logout(c.ctx)

	c.rc.Collect(ch)
	c.cc.Collect(ch)
	c.wc.Collect(ch)

	// collect is deferred
	c.up.Set(1)
}
