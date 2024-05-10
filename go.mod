module github.com/hairyhenderson/hitron_coda_exporter

go 1.19

require (
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/go-kit/log v0.2.1
	github.com/hairyhenderson/hitron_coda v0.0.0-20230225142238-f4b412fad70a
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/common v0.48.0
	github.com/stretchr/testify v1.9.0
	gopkg.in/yaml.v3 v3.0.1
)

// until https://github.com/prometheus/common/pull/458 is merged
replace github.com/prometheus/common => github.com/hairyhenderson/common v0.0.0-20230225150330-8fe65a7cb7a9

require (
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
