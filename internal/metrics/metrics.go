package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agency_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agency_http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	BatchOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agency_batch_ops_total",
			Help: "Total batch operations",
		},
		[]string{"action", "result"},
	)
)

func Register() {
	prometheus.MustRegister(HTTPRequestsTotal, HTTPRequestDuration, BatchOpsTotal)
}

func Handler() http.Handler {
	return promhttp.Handler()
}
