package main

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

// routerCollector tracks interesting metrics from the hitron Router* APIs
type routerCollector struct {
	ctx     context.Context
	client  func() *hitron.CableModem
	logger  log.Logger
	sysInfo struct {
		systemTimeSeconds       prometheus.Gauge
		lanReceiveBytesTotal    *prometheus.CounterVec
		lanTransmitBytesTotal   *prometheus.CounterVec
		wanReceiveBytesTotal    *prometheus.CounterVec
		wanTransmitBytesTotal   *prometheus.CounterVec
		wanReceivePacketsTotal  *prometheus.CounterVec
		wanTransmitPacketsTotal *prometheus.CounterVec
		systemLanUptimeSeconds  *prometheus.GaugeVec
		systemWanUptimeSeconds  *prometheus.GaugeVec
	}
}

//nolint:funlen
func newRouterCollector(ctx context.Context, logger log.Logger, clientProvider func() *hitron.CableModem) routerCollector {
	c := routerCollector{ctx: ctx, logger: logger, client: clientProvider}

	sub := "router"

	c.sysInfo.systemTimeSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "system_time_seconds",
		Help:      "The router's current system time (UTC, seconds past the epoch)",
	})

	c.sysInfo.lanReceiveBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "lan_receive_bytes_total",
		Help:      "Number of bytes received on the LAN interface",
	}, []string{"lan_name"})
	c.sysInfo.lanTransmitBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "lan_transmit_bytes_total",
		Help:      "Number of bytes transmitted on the LAN interface",
	}, []string{"lan_name"})
	c.sysInfo.systemLanUptimeSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "lan_uptime_seconds",
		Help:      "The LAN interface's uptime in seconds",
	}, []string{"lan_name"})

	c.sysInfo.wanReceiveBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "wan_receive_bytes_total",
		Help:      "Number of bytes received on the WAN interface",
	}, []string{"wan_name"})
	c.sysInfo.wanTransmitBytesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "wan_transmit_bytes_total",
		Help:      "Number of bytes transmitted on the WAN interface",
	}, []string{"wan_name"})
	c.sysInfo.wanReceivePacketsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "wan_receive_packets_total",
		Help:      "Number of packets received on the WAN interface",
	}, []string{"wan_name"})
	c.sysInfo.wanTransmitPacketsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "wan_transmit_packets_total",
		Help:      "Number of packets transmitted on the WAN interface",
	}, []string{"wan_name"})
	c.sysInfo.systemWanUptimeSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "wan_uptime_seconds",
		Help:      "The WAN interface's uptime in seconds",
	}, []string{"wan_name"})

	return c
}

// Describe implements Prometheus.Collector.
func (c routerCollector) Describe(ch chan<- *prometheus.Desc) {
	c.sysInfo.lanReceiveBytesTotal.Describe(ch)
}

// Collect implements Prometheus.Collector.
func (c routerCollector) Collect(ch chan<- prometheus.Metric) {
	si, err := c.client().RouterSysInfo(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"router_error", "Error scraping target", nil, nil), err)

		return
	}

	c.sysInfo.systemTimeSeconds.Set(float64(si.SystemTime.UnixNano()) / float64(time.Nanosecond))
	c.sysInfo.systemTimeSeconds.Collect(ch)

	c.sysInfo.lanReceiveBytesTotal.WithLabelValues(si.LANName).Add(float64(si.LanRx))
	c.sysInfo.lanReceiveBytesTotal.Collect(ch)

	c.sysInfo.lanTransmitBytesTotal.WithLabelValues(si.LANName).Add(float64(si.LanTx))
	c.sysInfo.lanTransmitBytesTotal.Collect(ch)

	c.sysInfo.systemLanUptimeSeconds.WithLabelValues(si.LANName).Set(si.SystemLanUptime.Seconds())
	c.sysInfo.systemLanUptimeSeconds.Collect(ch)

	c.sysInfo.wanReceiveBytesTotal.WithLabelValues(si.WanName).Add(float64(si.WanRx))
	c.sysInfo.wanReceiveBytesTotal.Collect(ch)

	c.sysInfo.wanTransmitBytesTotal.WithLabelValues(si.WanName).Add(float64(si.WanTx))
	c.sysInfo.wanTransmitBytesTotal.Collect(ch)

	c.sysInfo.wanReceivePacketsTotal.WithLabelValues(si.WanName).Add(float64(si.WanRxPkts))
	c.sysInfo.wanReceivePacketsTotal.Collect(ch)

	c.sysInfo.wanTransmitPacketsTotal.WithLabelValues(si.WanName).Add(float64(si.WanTxPkts))
	c.sysInfo.wanTransmitPacketsTotal.Collect(ch)

	c.sysInfo.systemWanUptimeSeconds.WithLabelValues(si.WanName).Set(si.SystemWanUptime.Seconds())
	c.sysInfo.systemWanUptimeSeconds.Collect(ch)
}
