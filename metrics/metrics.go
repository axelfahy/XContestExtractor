package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DocumentsTotal             prometheus.Counter
	DuplicatesTotal            prometheus.Counter
	ErrorsTotal                prometheus.Counter
	HttpRequestDurationSeconds prometheus.Summary
	HttpRequestsTotal          prometheus.Counter
	RunsTotal                  prometheus.Counter
)

type Config struct {
	Namespace string
	Subsystem string
	Path      string
}

func InitPrometheus(config Config, mux *http.ServeMux) {
	DocumentsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "documents_total",
		Help:      "Number of documents inserted.",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(DocumentsTotal)

	DuplicatesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "duplicates_total",
		Help:      "Number of duplicates documents.",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(DuplicatesTotal)

	ErrorsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "errors_total",
		Help:      "Number of errors.",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(ErrorsTotal)

	HttpRequestDurationSeconds = prometheus.NewSummary(prometheus.SummaryOpts{
		Name:      "http_request_duration_seconds",
		Help:      "Duration of http requests",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(HttpRequestDurationSeconds)

	HttpRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "http_requests_total",
		Help:      "Number of http requests.",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(HttpRequestsTotal)

	RunsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "runs_total",
		Help:      "Number of runs.",
		Namespace: config.Namespace,
		Subsystem: config.Subsystem,
	})
	prometheus.MustRegister(RunsTotal)

	mux.Handle(config.Path, promhttp.Handler())
}
