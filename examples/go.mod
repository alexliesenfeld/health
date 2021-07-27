module github.com/alexliesenfeld/health/examples

go 1.13

require (
	github.com/InVisionApp/go-health v2.1.0+incompatible
	github.com/alexliesenfeld/health v0.5.0
	github.com/etherlabsio/healthcheck/v2 v2.0.0
	github.com/google/uuid v1.3.0
	github.com/hellofresh/health-go/v4 v4.4.0
	github.com/heptiolabs/healthcheck v0.0.0-20180807145615-6ff867650f40
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/sirupsen/logrus v1.8.1
)

replace github.com/alexliesenfeld/health => ../
