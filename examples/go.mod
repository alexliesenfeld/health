module github.com/alexliesenfeld/health/examples

go 1.13

require (
	github.com/alexliesenfeld/health v0.0.0
	github.com/google/uuid v1.3.0
	github.com/mattn/go-sqlite3 v1.14.8
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/uber/jaeger-client-go v2.29.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	go.uber.org/atomic v1.9.0 // indirect
)

replace github.com/alexliesenfeld/health => ../
