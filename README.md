# Hitron CODA-4x8x Exporter

A Prometheus exporter for the Hitron CODA-4x8x DOCSIS 3.1 cable modem series.
It gathers metrics on demand using the HTTP API. The username/password must be
configured.

This is tested on a Hitron CODA-4680 with firmware `7.1.1.2.2b9`, untested on
other models and releases.

To configure Prometheus to scrape from this exporter:

```yaml
  - job_name: 'coda4680'
    static_configs:
      - targets:
        - 'localhost:9764'
```
