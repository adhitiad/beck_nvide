package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics handles application-wide observability counters
type Metrics struct {
	mu             sync.RWMutex
	wsConnections  int64
	streamDuration float64
	giftTotalValue int64

	// Prometheus Metrics
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	WSConnectionsGauge    prometheus.Gauge
	StreamDurationCounter prometheus.Counter
	GiftTotalValueCounter prometheus.Counter
}

var defaultMetrics *Metrics
var once sync.Once

// GetDefault returns the singleton metrics instance
func GetDefault() *Metrics {
	once.Do(func() {
		defaultMetrics = &Metrics{
			HTTPRequestsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "http_requests_total",
					Help: "Total number of HTTP requests processed",
				},
				[]string{"method", "path", "status"},
			),
			HTTPRequestDuration: prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "http_request_duration_seconds",
					Help:    "Latency of HTTP requests in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"method", "path"},
			),
			WSConnectionsGauge: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: "ws_active_connections",
					Help: "Number of active WebSocket connections",
				},
			),
			StreamDurationCounter: prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: "stream_duration_seconds_total",
					Help: "Total duration of live streams in seconds",
				},
			),
			GiftTotalValueCounter: prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: "gift_total_value_idr_total",
					Help: "Total IDR value of gifts sent",
				},
			),
		}

		// Register to standard prometheus registry
		prometheus.MustRegister(
			defaultMetrics.HTTPRequestsTotal,
			defaultMetrics.HTTPRequestDuration,
			defaultMetrics.WSConnectionsGauge,
			defaultMetrics.StreamDurationCounter,
			defaultMetrics.GiftTotalValueCounter,
		)
	})
	return defaultMetrics
}

// IncWSConnections increments active websocket connections
func (m *Metrics) IncWSConnections() {
	m.mu.Lock()
	m.wsConnections++
	m.mu.Unlock()
	m.WSConnectionsGauge.Inc()
}

// DecWSConnections decrements active websocket connections
func (m *Metrics) DecWSConnections() {
	m.mu.Lock()
	m.wsConnections--
	m.mu.Unlock()
	m.WSConnectionsGauge.Dec()
}

// AddStreamDuration adds seconds to total stream duration
func (m *Metrics) AddStreamDuration(seconds float64) {
	m.mu.Lock()
	m.streamDuration += seconds
	m.mu.Unlock()
	m.StreamDurationCounter.Add(seconds)
}

// AddGiftValue adds IDR value to total gift volume
func (m *Metrics) AddGiftValue(value int64) {
	m.mu.Lock()
	m.giftTotalValue += value
	m.mu.Unlock()
	m.GiftTotalValueCounter.Add(float64(value))
}

// GetSnapshot returns current metrics values
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"ws_active_connections":   m.wsConnections,
		"stream_duration_seconds": m.streamDuration,
		"gift_total_value_idr":    m.giftTotalValue,
	}
}
