package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
)

// Metrics represents a metrics collector
type Metrics struct {
	operations map[string]time.Time
	durations  map[string]time.Duration
	counters   map[string]int64
	gauges     map[string]float64
	mu         sync.Mutex
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		operations: make(map[string]time.Time),
		durations:  make(map[string]time.Duration),
		counters:   make(map[string]int64),
		gauges:     make(map[string]float64),
	}
}

// Start starts timing an operation
func (m *Metrics) Start(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[operation] = time.Now()
	log.Debug("Started operation: %s", operation)
}

// Stop stops timing an operation and records the duration
func (m *Metrics) Stop(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	start, ok := m.operations[operation]
	if !ok {
		log.Warn("Attempted to stop timing for unknown operation: %s", operation)
		return
	}

	duration := time.Since(start)
	m.durations[operation] = duration
	delete(m.operations, operation)

	log.Debug("Completed operation: %s in %v", operation, duration)
}

// Increment increments a counter
func (m *Metrics) Increment(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

// Set sets a gauge value
func (m *Metrics) Set(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

// GetDuration gets the duration of an operation
func (m *Metrics) GetDuration(operation string) (time.Duration, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	duration, ok := m.durations[operation]
	return duration, ok
}

// GetCounter gets a counter value
func (m *Metrics) GetCounter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.counters[name]
	return value, ok
}

// GetGauge gets a gauge value
func (m *Metrics) GetGauge(name string) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, ok := m.gauges[name]
	return value, ok
}

// LogMemoryUsage logs the current memory usage
func (m *Metrics) LogMemoryUsage(operation string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.Set(fmt.Sprintf("%s.memory.alloc", operation), float64(memStats.Alloc)/1024/1024)
	m.Set(fmt.Sprintf("%s.memory.total_alloc", operation), float64(memStats.TotalAlloc)/1024/1024)
	m.Set(fmt.Sprintf("%s.memory.sys", operation), float64(memStats.Sys)/1024/1024)
	m.Set(fmt.Sprintf("%s.memory.num_gc", operation), float64(memStats.NumGC))

	log.Info("%s: Memory usage - Alloc: %v MiB, TotalAlloc: %v MiB, Sys: %v MiB, NumGC: %v",
		operation,
		memStats.Alloc/1024/1024,
		memStats.TotalAlloc/1024/1024,
		memStats.Sys/1024/1024,
		memStats.NumGC)
}

// Report generates a report of all metrics
func (m *Metrics) Report() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	report := make(map[string]interface{})

	// Add durations
	durations := make(map[string]string)
	for op, duration := range m.durations {
		durations[op] = duration.String()
	}
	report["durations"] = durations

	// Add counters
	counters := make(map[string]int64)
	for name, value := range m.counters {
		counters[name] = value
	}
	report["counters"] = counters

	// Add gauges
	gauges := make(map[string]float64)
	for name, value := range m.gauges {
		gauges[name] = value
	}
	report["gauges"] = gauges

	return report
}

// DefaultMetrics is the default metrics instance
var DefaultMetrics = NewMetrics()

// Start starts timing an operation using the default metrics
func Start(operation string) {
	DefaultMetrics.Start(operation)
}

// Stop stops timing an operation using the default metrics
func Stop(operation string) {
	DefaultMetrics.Stop(operation)
}

// Increment increments a counter using the default metrics
func Increment(name string, value int64) {
	DefaultMetrics.Increment(name, value)
}

// Set sets a gauge value using the default metrics
func Set(name string, value float64) {
	DefaultMetrics.Set(name, value)
}

// LogMemoryUsage logs the current memory usage using the default metrics
func LogMemoryUsage(operation string) {
	DefaultMetrics.LogMemoryUsage(operation)
}

// Report generates a report of all metrics using the default metrics
func Report() map[string]interface{} {
	return DefaultMetrics.Report()
}
