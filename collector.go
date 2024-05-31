package main

import (
	"context"
	"log/slog"

	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ctx    context.Context
	client *hitron.CableModem
	rc     routerCollector
	cc     cmCollector
	wc     wifiCollector

	up prometheus.Gauge

	config config
}

func newCollector(ctx context.Context, conf config) *collector {
	c := &collector{ctx: ctx, config: conf}
	c.rc = newRouterCollector(ctx, c.getClient)
	c.cc = newCMCollector(ctx, c.getClient)
	c.wc = newWiFiCollector(ctx, c.getClient)

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
		slog.ErrorContext(c.ctx, "Error creating client", "err", err)
		exporterClientErrors.Inc()

		return
	}

	err = c.client.Login(c.ctx)
	if err != nil {
		slog.ErrorContext(c.ctx, "Error logging in", "err", err)
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
