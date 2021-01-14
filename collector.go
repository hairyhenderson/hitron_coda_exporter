package main

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

type collector struct {
	ctx    context.Context
	config config
	logger log.Logger

	client *hitron.CableModem
	rc     routerCollector
	cc     cmCollector
	wc     wifiCollector
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
}

// Collect implements Prometheus.Collector.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	var err error
	c.client, err = hitron.New(c.config.Host, c.config.Username, c.config.Password)

	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"_error", "Error scraping target", nil, nil), err)

		return
	}

	err = c.client.Login(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"_error", "Error scraping target", nil, nil), err)

		return
	}

	defer c.client.Logout(c.ctx)

	c.rc.Collect(ch)
	c.cc.Collect(ch)
	c.wc.Collect(ch)
}
