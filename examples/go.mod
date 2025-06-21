module github.com/alexliesenfeld/health/examples

go 1.21
toolchain go1.22.5

require (
	github.com/InVisionApp/go-health v2.1.0+incompatible
	github.com/alexliesenfeld/health v0.0.0
	github.com/etherlabsio/healthcheck/v2 v2.0.0
	github.com/google/uuid v1.6.0
	github.com/hellofresh/health-go/v4 v4.7.0
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/mattn/go-sqlite3 v1.14.23
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/onsi/gomega v1.34.2 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0 // indirect
)

replace github.com/alexliesenfeld/health => ../
