package main

import (
	"context"
	"fmt"
	"log/slog"

	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

// wifiCollector tracks interesting metrics from the hitron CM* APIs
type wifiCollector struct {
	ctx    context.Context
	client func() *hitron.CableModem

	clientStats struct {
		rssi      *prometheus.GaugeVec
		dataRate  *prometheus.GaugeVec
		bandwidth *prometheus.GaugeVec
	}
}

func newWiFiCollector(ctx context.Context, clientProvider func() *hitron.CableModem) wifiCollector {
	c := wifiCollector{ctx: ctx, client: clientProvider}

	sub := "wifi"

	clientLabels := []string{"band", "hostname", "phy_mode", "ssid", "mac_addr"}
	c.clientStats.rssi = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "client_rssi_db",
		Help:      "Received Signal Strength Indicator. Estimated measure of power level that a client is receiving from AP, in dB",
	}, clientLabels)
	c.clientStats.dataRate = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "client_data_rate_bytes_per_second",
		Help:      "Data rate for this client, in bytes per second (converted from bits/sec)",
	}, clientLabels)
	c.clientStats.bandwidth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "client_bandwidth_hertz",
		Help:      "Channel bandwidth, in hertz",
	}, clientLabels)

	return c
}

// Describe implements Prometheus.Collector.
func (c wifiCollector) Describe(ch chan<- *prometheus.Desc) {
	c.clientStats.rssi.Describe(ch)
	c.clientStats.dataRate.Describe(ch)
	c.clientStats.bandwidth.Describe(ch)
}

// Collect implements Prometheus.Collector.
func (c wifiCollector) Collect(ch chan<- prometheus.Metric) {
	client := c.client()
	if client == nil {
		err := fmt.Errorf("client not initialized: %v", client)
		slog.ErrorContext(c.ctx, "Error scraping target", "err", err)
		exporterClientErrors.Inc()

		return
	}

	wc, err := client.WiFiClient(c.ctx)
	if err != nil {
		slog.ErrorContext(c.ctx, "Error scraping WiFiClient", "err", err)
		exporterRequestErrors.Inc()

		return
	}

	for _, cl := range wc.Clients {
		l := prometheus.Labels{
			"band":     cl.Band,
			"hostname": cl.Hostname,
			"phy_mode": cl.PhyMode,
			"ssid":     cl.SSID,
			"mac_addr": cl.MACAddr.String(),
		}

		c.clientStats.rssi.With(l).Set(float64(cl.RSSI))
		//nolint:gomnd
		c.clientStats.dataRate.With(l).Set(float64(cl.DataRate) / 8)
		c.clientStats.bandwidth.With(l).Set(float64(cl.Bandwidth))
	}

	c.clientStats.rssi.Collect(ch)
	c.clientStats.dataRate.Collect(ch)
	c.clientStats.bandwidth.Collect(ch)
}
