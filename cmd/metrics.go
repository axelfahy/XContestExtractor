package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	documentsTotal             prometheus.Counter
	duplicatesTotal            prometheus.Counter
	errorsTotal                prometheus.Counter
	httpRequestDurationSeconds prometheus.Summary
	httpRequestsTotal          prometheus.Counter
	runsTotal                  prometheus.Counter
)

func initPrometheus(env envConfig, mux *http.ServeMux) {
	documentsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "documents_total",
		Help:      "Number of documents inserted.",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(documentsTotal)

	duplicatesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "duplicates_total",
		Help:      "Number of duplicates documents.",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(duplicatesTotal)

	errorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "errors_total",
		Help:      "Number of errors.",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(errorsTotal)

	httpRequestDurationSeconds = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:      "http_request_duration_seconds",
		Help:      "Duration of http requests",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(httpRequestDurationSeconds)

	httpRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "http_requests_total",
		Help:      "Number of http requests.",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(httpRequestsTotal)

	runsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "runs_total",
		Help:      "Number of runs.",
		Namespace: env.MetricsNamespace,
		Subsystem: env.MetricsSubsystem,
	})
	prometheus.MustRegister(runsTotal)

	mux.Handle(env.MetricsPath, promhttp.Handler())
}
