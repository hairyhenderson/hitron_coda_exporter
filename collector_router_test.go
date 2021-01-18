package main

import (
	"net"
	"testing"

	hitron "github.com/hairyhenderson/hitron_coda"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRouterSysInfoLabels(t *testing.T) {
	si := hitron.RouterSysInfo{
		PrivLanIP:  net.IPv4(192, 168, 0, 1),
		PrivLanNet: &net.IPNet{Mask: net.IPv4Mask(255, 255, 255, 0)},
		WanIP:      []net.IP{net.IPv4(127, 200, 100, 10), net.ParseIP("2001:3::cafe:dead:beef")},
		DNS:        []net.IP{net.IPv4(127, 0, 0, 1), net.ParseIP("2001:3::2")},
		RFMac:      net.HardwareAddr{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE},
		RouterMode: "fancy!",
	}
	loc := hitron.RouterLocation{LocationText: "attic"}
	expected := prometheus.Labels{
		"lan_ip":      "192.168.0.1/24",
		"wan_ip4":     "127.200.100.10",
		"wan_ip6":     "2001:3::cafe:dead:beef",
		"dns4":        "127.0.0.1",
		"dns6":        "2001:3::2",
		"rf_mac":      "de:ad:be:ef:ca:fe",
		"router_mode": "fancy!",
		"location":    "attic",
	}
	out := routerSysInfoLabels(si, loc)
	assert.Equal(t, expected, out)
}
