package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
)

// cmCollector tracks interesting metrics from the hitron CM* APIs
type cmCollector struct {
	ctx     context.Context
	client  func() *hitron.CableModem
	logger  log.Logger
	sysInfo struct {
		usDataRate       prometheus.Gauge
		dsDataRate       prometheus.Gauge
		dhcpLeaseSeconds *prometheus.GaugeVec
	}
	dsInfo struct {
		frequency      *prometheus.GaugeVec
		signalStrength *prometheus.GaugeVec
		snr            *prometheus.GaugeVec
		receivedBytes  *prometheus.CounterVec
		corrected      *prometheus.CounterVec
		uncorrected    *prometheus.CounterVec
	}
	usInfo struct {
		frequency      *prometheus.GaugeVec
		signalStrength *prometheus.GaugeVec
		bandwidth      *prometheus.GaugeVec
	}
	dsOfdm struct {
		subcarrierFreq *prometheus.GaugeVec
		plcPower       *prometheus.GaugeVec
	}
	usOfdm struct {
		digAtten    *prometheus.GaugeVec
		digAttenBo  *prometheus.GaugeVec
		channelBw   *prometheus.GaugeVec
		repPower    *prometheus.GaugeVec
		targetPower *prometheus.GaugeVec
	}
}

//nolint:funlen
func newCMCollector(ctx context.Context, logger log.Logger, clientProvider func() *hitron.CableModem) cmCollector {
	c := cmCollector{ctx: ctx, logger: logger, client: clientProvider}

	sub := "cm"

	c.sysInfo.usDataRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_data_rate_bytes_per_second",
		Help:      "WAN upstream data rate, in bytes per second",
	})
	c.sysInfo.dsDataRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_data_rate_bytes_per_second",
		Help:      "WAN downstream data rate, in bytes per second",
	})
	c.sysInfo.dhcpLeaseSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "dhcp_lease_duration_seconds",
		Help:      "Lease duration for DHCP on WAN interface",
	}, []string{"ip", "mac_addr"})

	portInfoLabels := []string{"port", "channel"}
	c.dsInfo.frequency = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_frequency_hertz",
		Help:      "Downstream port frequency",
	}, portInfoLabels)
	c.dsInfo.signalStrength = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_signal_strength_dbmv",
		Help:      "Downstream data channel signal strength, in dBmV (decibels above/below 1 millivolt)",
	}, portInfoLabels)
	c.dsInfo.snr = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_signal_noise_ratio_db",
		Help:      "Downstream data channel signal-to-noise ratio, in dB",
	}, portInfoLabels)
	c.dsInfo.receivedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_received_bytes",
		Help:      "Number of octets/bytes received",
	}, portInfoLabels)
	c.dsInfo.corrected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_corrected_blocks",
		Help:      "Number of blocks received that required correction due to corruption, and were corrected",
	}, portInfoLabels)
	c.dsInfo.uncorrected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_uncorrected_blocks",
		Help:      "Number of blocks received that required correction due to corruption, but were unable to be corrected",
	}, portInfoLabels)

	c.usInfo.frequency = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_frequency_hertz",
		Help:      "Upstream port frequency",
	}, portInfoLabels)
	c.usInfo.signalStrength = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_signal_strength_dbmv",
		Help:      "Upstream data channel signal strength, in dBmV (decibels above/below 1 millivolt)",
	}, portInfoLabels)
	c.usInfo.bandwidth = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_bandwidth_bytes_per_second",
		Help:      "Upstream data channel bandwidth, in bytes per second",
	}, portInfoLabels)

	dsOfdmLabels := []string{"receiver", "fft_type"}
	c.dsOfdm.subcarrierFreq = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_ofdm_subcarrier_freq_hertz",
		Help:      "Downstream frequency in Hz of the first OFDM subcarrier",
	}, dsOfdmLabels)
	c.dsOfdm.plcPower = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "downstream_ofdm_plc_power_dbmv",
		Help:      "Power level device was instructed to use on this OFDM connection by the Physical Link Channel, in dB above/below 1mV",
	}, dsOfdmLabels)

	usOfdmLabels := []string{"channel", "enabled", "fft_size"}
	c.usOfdm.digAtten = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_ofdm_digital_attenuation_db",
		Help:      "The digital attenuation (signal loss) of the transmission medium on which the channel's signal is carried, in decibels (dB).",
	}, usOfdmLabels)
	c.usOfdm.digAttenBo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_ofdm_measured_digital_attenuation_db",
		Help:      "The measured digital attenuation of the channel's signal, in decibels (dB).",
	}, usOfdmLabels)
	c.usOfdm.channelBw = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_ofdm_channel_bandwidth_hz",
		Help:      "Bandwidth of this channel, expressed as the number of subchannels multiplied by the channel's FFT size, in hertz (Hz).",
	}, usOfdmLabels)
	c.usOfdm.repPower = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_ofdm_reported_power_qdbmv",
		Help:      "Reported power of this channel, in quarter-dB above/below 1mV (quarter-dBmV).",
	}, usOfdmLabels)
	c.usOfdm.targetPower = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNS,
		Subsystem: sub,
		Name:      "upstream_ofdm_target_power_qdbmv",
		Help:      "Target power (P1.6r_n, or power spectral density in 1.6MHz) of this channel, in quarter-dB above/below 1mV (quarter-dBmV).",
	}, usOfdmLabels)

	return c
}

// Describe implements Prometheus.Collector.
func (c cmCollector) Describe(ch chan<- *prometheus.Desc) {
	c.sysInfo.usDataRate.Describe(ch)
	c.sysInfo.dsDataRate.Describe(ch)
	c.sysInfo.dhcpLeaseSeconds.Describe(ch)

	c.dsInfo.frequency.Describe(ch)
	c.dsInfo.signalStrength.Describe(ch)
	c.dsInfo.snr.Describe(ch)
	c.dsInfo.receivedBytes.Describe(ch)
	c.dsInfo.corrected.Describe(ch)
	c.dsInfo.uncorrected.Describe(ch)

	c.usInfo.frequency.Describe(ch)
	c.usInfo.signalStrength.Describe(ch)
	c.usInfo.bandwidth.Describe(ch)

	c.dsOfdm.plcPower.Describe(ch)
	c.dsOfdm.subcarrierFreq.Describe(ch)

	c.usOfdm.channelBw.Describe(ch)
	c.usOfdm.digAtten.Describe(ch)
	c.usOfdm.digAttenBo.Describe(ch)
	c.usOfdm.repPower.Describe(ch)
	c.usOfdm.targetPower.Describe(ch)
}

// Collect implements Prometheus.Collector.
func (c cmCollector) Collect(ch chan<- prometheus.Metric) {
	client := c.client()
	if client == nil {
		err := fmt.Errorf("client not initialized: %v", client)
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	c.collectSysInfo(ch, client)
	c.collectDsInfo(ch, client)
	c.collectUsInfo(ch, client)
	c.collectOfdm(ch, client)
}

func (c cmCollector) collectSysInfo(ch chan<- prometheus.Metric, client *hitron.CableModem) {
	si, err := client.CMSysInfo(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	// bytes not bits
	//nolint:gomnd
	c.sysInfo.usDataRate.Set(float64(si.UsDataRate) / 8)
	c.sysInfo.usDataRate.Collect(ch)

	// bytes not bits
	//nolint:gomnd
	c.sysInfo.dsDataRate.Set(float64(si.DsDataRate) / 8)
	c.sysInfo.dsDataRate.Collect(ch)

	c.sysInfo.dhcpLeaseSeconds.
		WithLabelValues(si.IP.String(), si.MacAddr.String()).
		Set(si.Lease.Seconds())
	c.sysInfo.dhcpLeaseSeconds.Collect(ch)
}

func (c cmCollector) collectDsInfo(ch chan<- prometheus.Metric, client *hitron.CableModem) {
	dsinfo, err := client.CMDsInfo(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	for _, port := range dsinfo.Ports {
		l := prometheus.Labels{"port": port.PortID, "channel": port.ChannelID}

		c.dsInfo.frequency.With(l).Set(float64(port.Frequency))
		c.dsInfo.signalStrength.With(l).Set(port.SignalStrength)
		c.dsInfo.snr.With(l).Set(port.SNR)
		c.dsInfo.receivedBytes.With(l).Add(float64(port.DsOctets))
		c.dsInfo.corrected.With(l).Add(float64(port.Correcteds))
		c.dsInfo.uncorrected.With(l).Add(float64(port.Uncorrect))
	}

	c.dsInfo.frequency.Collect(ch)
	c.dsInfo.signalStrength.Collect(ch)
	c.dsInfo.snr.Collect(ch)
	c.dsInfo.receivedBytes.Collect(ch)
	c.dsInfo.corrected.Collect(ch)
	c.dsInfo.uncorrected.Collect(ch)
}

func (c cmCollector) collectUsInfo(ch chan<- prometheus.Metric, client *hitron.CableModem) {
	usinfo, err := client.CMUsInfo(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	for _, port := range usinfo.Ports {
		l := prometheus.Labels{"port": port.PortID, "channel": port.ChannelID}

		c.usInfo.frequency.With(l).Set(float64(port.Frequency))
		c.usInfo.signalStrength.With(l).Set(port.SignalStrength)
		// we want bytes/sec here, not bits/sec
		//nolint:gomnd
		c.usInfo.bandwidth.With(l).Set(float64(port.Bandwidth) / 8)
	}

	c.usInfo.frequency.Collect(ch)
	c.usInfo.signalStrength.Collect(ch)
	c.usInfo.bandwidth.Collect(ch)
}

func (c cmCollector) collectOfdm(ch chan<- prometheus.Metric, client *hitron.CableModem) {
	usofdm, err := client.CMUsOfdm(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	for _, channel := range usofdm.Channels {
		l := prometheus.Labels{
			"channel":  strconv.Itoa(channel.ID),
			"enabled":  strconv.FormatBool(channel.Enable),
			"fft_size": channel.FFTSize,
		}

		c.usOfdm.channelBw.With(l).Set(channel.ChannelBw)
		c.usOfdm.digAtten.With(l).Set(channel.DigAtten)
		c.usOfdm.digAttenBo.With(l).Set(channel.DigAttenBo)
		c.usOfdm.repPower.With(l).Set(channel.RepPower)
		c.usOfdm.targetPower.With(l).Set(channel.RepPower1_6)
	}

	c.usOfdm.channelBw.Collect(ch)
	c.usOfdm.digAtten.Collect(ch)
	c.usOfdm.digAttenBo.Collect(ch)
	c.usOfdm.repPower.Collect(ch)
	c.usOfdm.targetPower.Collect(ch)

	dsofdm, err := client.CMDsOfdm(c.ctx)
	if err != nil {
		level.Info(c.logger).Log("msg", "Error scraping target", "err", err)
		ch <- prometheus.NewInvalidMetric(prometheus.NewDesc(metricsNS+"cm_error", "Error scraping target", nil, nil), err)

		return
	}

	for _, receiver := range dsofdm.Receivers {
		l := prometheus.Labels{
			"receiver": strconv.Itoa(receiver.ID),
			"fft_type": receiver.FFTType,
		}

		c.dsOfdm.plcPower.With(l).Set(receiver.PLCPower)
		c.dsOfdm.subcarrierFreq.With(l).Set(float64(receiver.SubcarrierFreq))
	}

	c.dsOfdm.plcPower.Collect(ch)
	c.dsOfdm.subcarrierFreq.Collect(ch)
}
