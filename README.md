![Build](https://github.com/hairyhenderson/hitron_coda_exporter/workflows/Build/badge.svg)
![Docker Build](https://github.com/hairyhenderson/hitron_coda_exporter/workflows/Docker%20Build/badge.svg)

# Hitron CODA-4x8x Exporter

A Prometheus exporter for the [Hitron CODA-4x8x](http://hitron-americas.com/products/service-providers/coda-4680-cable-modem-router/)
DOCSIS 3.1 cable modem/router series.
It gathers metrics on demand using the HTTP API. The username/password must be
configured in a configuration file.

This is tested on a Hitron CODA-4680 with firmware `7.1.1.2.2b9`, untested on
other models and releases.

## Installation

You can build a binary for your system with `go get github.com/hairyhenderson/hitron_coda_exporter`,
or you can use a [pre-built Docker image](https://hub.docker.com/r/hairyhenderson/hitron_coda_exporter):

```console
$ docker run hairyhenderson/hitron_coda_exporter
```

There are two variants: `:latest` and `:alpine` - no difference except the
latter is based on Alpine and contains a shell. The former is a `FROM scratch`
image, containing only the binary.

The image is built for multiple platforms and architectures:

- `linux/amd64` (x86_64)
- `linux/arm64` (64-bit ARM/aarch64)
- `linux/arm/v6` (32-bit ARM v6, like Raspberry Pi Zero)
- `linux/arm/v7` (32-bit ARM v7, like Raspbarry Pi 2B)
- `windows/amd64` (Windows, based on Windows Nano Server)

See https://hub.docker.com/r/hairyhenderson/hitron_coda_exporter for full
details.

## Usage

First, you need a configuration file with the address, username, and password
for the device. The default name is `hitron_coda.yml`, but it can be configured
with the `--config.file` flag.

_`hitron_coda.yml`:_
```yaml
host: 192.168.0.1
username: cusadmin
password: mypassword
```

To run the exporter:

```console
$ docker run -v /tmp/hitron_coda.yml:/hitron_coda.yml -p 9779:9779 hairyhenderson/hitron_coda_exporter
...
```

Now you can visit http://localhost:9779/scrape to have the exporter scrape
metrics from the device.

To configure Prometheus to scrape from this exporter, use a
[scrape_config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config)
like this one:

```yaml
  - job_name: 'coda4680'
    metrics_path: /scrape
    static_configs:
      - targets:
        - 'localhost:9779'
```

## License

[The MIT License](http://opensource.org/licenses/MIT)

Copyright (c) 2020-2021 Dave Henderson
