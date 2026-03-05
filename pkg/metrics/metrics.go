// Package metrics provides Prometheus instrumentation for all PVP services.
// It exposes a shared metrics registry and HTTP middleware for request tracing.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all shared Prometheus metrics.
type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsInFlight *prometheus.GaugeVec

	// NATS metrics
	NATSPublishedTotal *prometheus.CounterVec
	NATSConsumedTotal  *prometheus.CounterVec
	NATSErrorsTotal    *prometheus.CounterVec

	// Business metrics
	OrdersTotal        *prometheus.CounterVec // proxy + vps
	WalletTransactions *prometheus.CounterVec
	ActiveInstances    *prometheus.GaugeVec

	// Circuit breaker state
	CircuitBreakerState *prometheus.GaugeVec // 0=closed, 1=open, 2=half_open
}

// New creates and registers all Prometheus metrics for a service.
func New(serviceName string) *Metrics {
	labels := prometheus.Labels{"service": serviceName}
	_ = labels // used in metric help strings

	m := &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests by method, path, and status code.",
		}, []string{"method", "path", "status"}),

		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets, // .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		}, []string{"method", "path"}),

		HTTPRequestsInFlight: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "http_requests_in_flight",
			Help:      "Current number of in-flight HTTP requests.",
		}, []string{"method"}),

		NATSPublishedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "nats_published_total",
			Help:      "Total NATS messages published by subject.",
		}, []string{"subject"}),

		NATSConsumedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "nats_consumed_total",
			Help:      "Total NATS messages consumed by subject.",
		}, []string{"subject", "status"}), // status: ok|error|nack

		NATSErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "nats_errors_total",
			Help:      "Total NATS errors.",
		}, []string{"subject"}),

		OrdersTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "orders_total",
			Help:      "Total orders by type and status.",
		}, []string{"type", "status"}), // type: proxy|vps

		WalletTransactions: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "wallet_transactions_total",
			Help:      "Wallet transactions by type.",
		}, []string{"type"}), // type: deduct|credit|hold|release

		ActiveInstances: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "active_instances",
			Help:      "Currently active VPS/proxy instances.",
		}, []string{"type"}),

		CircuitBreakerState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "pvp",
			Subsystem: serviceName,
			Name:      "circuit_breaker_state",
			Help:      "Circuit breaker state: 0=closed, 1=open, 2=half_open.",
		}, []string{"name"}),
	}
	return m
}

// HTTPMiddleware instruments HTTP handlers with request count, latency, and in-flight metrics.
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := sanitizePath(r.URL.Path)
		method := r.Method

		m.HTTPRequestsInFlight.WithLabelValues(method).Inc()
		defer m.HTTPRequestsInFlight.WithLabelValues(method).Dec()

		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rw, r)
		dur := time.Since(start).Seconds()

		statusStr := strconv.Itoa(rw.status)
		m.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
		m.HTTPRequestDuration.WithLabelValues(method, path).Observe(dur)
	})
}

// Handler returns the Prometheus HTTP handler for /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}

// statusWriter wraps ResponseWriter to capture the HTTP status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// sanitizePath reduces high-cardinality path params (UUIDs → :id).
func sanitizePath(path string) string {
	// Very simple: replace UUID-like segments with :id
	// In production: use a proper router pattern extractor
	result := make([]byte, 0, len(path))
	i := 0
	for i < len(path) {
		if i+36 <= len(path) && looksLikeUUID(path[i:i+36]) {
			result = append(result, []byte(":id")...)
			i += 36
		} else {
			result = append(result, path[i])
			i++
		}
	}
	return string(result)
}

func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !isHex(byte(c)) {
			return false
		}
	}
	return true
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
