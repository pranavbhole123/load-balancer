package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Recorder struct {
	requestTotal      *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeConnections *prometheus.GaugeVec
}

// we also need a constructor

func NewRecorder() *Recorder {
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests proxied",
		},
		[]string{"backend", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets, // default: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"backend"},
	)

	activeConnections := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections per backend",
		},
		[]string{"backend"},
	)

	prometheus.MustRegister(requestsTotal, requestDuration, activeConnections)

	return &Recorder{
		requestTotal:      requestsTotal,
		requestDuration:   requestDuration,
		activeConnections: activeConnections,
	}
}

func (r *Recorder) Record(backend string, status int, duration time.Duration) {
    r.requestTotal.WithLabelValues(backend, strconv.Itoa(status)).Inc()
    r.requestDuration.WithLabelValues(backend).Observe(duration.Seconds())
}

func (r *Recorder) TrackActive(backend string) func() {
    r.activeConnections.WithLabelValues(backend).Inc()
    return func() {
        r.activeConnections.WithLabelValues(backend).Dec()
    }
}
