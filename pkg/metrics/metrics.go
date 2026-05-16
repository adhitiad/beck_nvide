package metrics

import (
	"sync"
)

// Metrics handles application-wide observability counters
type Metrics struct {
	mu                sync.RWMutex
	wsConnections     int64
	streamDuration    float64
	giftTotalValue    int64
}

var defaultMetrics = &Metrics{}

// GetDefault returns the singleton metrics instance
func GetDefault() *Metrics {
	return defaultMetrics
}

// IncWSConnections increments active websocket connections
func (m *Metrics) IncWSConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnections++
}

// DecWSConnections decrements active websocket connections
func (m *Metrics) DecWSConnections() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wsConnections--
}

// AddStreamDuration adds seconds to total stream duration
func (m *Metrics) AddStreamDuration(seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamDuration += seconds
}

// AddGiftValue adds IDR value to total gift volume
func (m *Metrics) AddGiftValue(value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.giftTotalValue += value
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
