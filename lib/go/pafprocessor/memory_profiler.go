package pafprocessor

import (
	"runtime"
	"sync"
	"time"
)

// MemoryProfiler tracks memory usage during processing
type MemoryProfiler struct {
	initialAlloc      uint64
	maxAlloc          uint64
	lastAlloc         uint64
	mutex             sync.Mutex
	snapshots         map[string]uint64
	startTime         time.Time
	memoryLimit       uint64 // Memory limit in bytes (0 for no limit)
	allocHistory      []uint64
	allocTimestamps   []time.Time
	historyInterval   time.Duration
	lastHistoryUpdate time.Time
	totalAllocated    uint64 // Total bytes allocated since start
	totalFreed        uint64 // Total bytes freed since start
	gcCount           uint32 // Number of GC cycles
}

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler() *MemoryProfiler {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &MemoryProfiler{
		initialAlloc:      memStats.Alloc,
		maxAlloc:          memStats.Alloc,
		lastAlloc:         memStats.Alloc,
		snapshots:         make(map[string]uint64),
		startTime:         time.Now(),
		allocHistory:      make([]uint64, 0, 1000),
		allocTimestamps:   make([]time.Time, 0, 1000),
		historyInterval:   5 * time.Second, // Record history every 5 seconds
		lastHistoryUpdate: time.Now(),
		totalAllocated:    memStats.TotalAlloc,
		totalFreed:        memStats.TotalAlloc - memStats.Alloc,
		gcCount:           memStats.NumGC,
	}
}

// TakeSnapshot takes a snapshot of the current memory usage with a label
func (mp *MemoryProfiler) TakeSnapshot(label string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.snapshots[label] = memStats.Alloc
	mp.lastAlloc = memStats.Alloc
	if memStats.Alloc > mp.maxAlloc {
		mp.maxAlloc = memStats.Alloc
	}

	Logger.Infof("Memory snapshot '%s': %d MB", label, memStats.Alloc/1024/1024)
}

// LogCurrentUsage logs the current memory usage
func (mp *MemoryProfiler) LogCurrentUsage(context string) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.lastAlloc = memStats.Alloc
	if memStats.Alloc > mp.maxAlloc {
		mp.maxAlloc = memStats.Alloc
	}

	Logger.Infof("Memory usage (%s): %d MB", context, memStats.Alloc/1024/1024)
}

// LogMemoryStats logs detailed memory statistics
func (mp *MemoryProfiler) LogMemoryStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Update history if interval has passed
	if time.Since(mp.lastHistoryUpdate) >= mp.historyInterval {
		mp.allocHistory = append(mp.allocHistory, memStats.Alloc)
		mp.allocTimestamps = append(mp.allocTimestamps, time.Now())
		mp.lastHistoryUpdate = time.Now()
	}

	// Update tracking stats
	newTotalAlloc := memStats.TotalAlloc
	newGCCount := memStats.NumGC

	// Calculate deltas
	allocatedSinceLastCheck := newTotalAlloc - mp.totalAllocated
	gcCountDelta := newGCCount - mp.gcCount

	// Update stored values
	mp.totalAllocated = newTotalAlloc
	mp.totalFreed = newTotalAlloc - memStats.Alloc
	mp.gcCount = newGCCount

	// Log detailed stats
	Logger.Infof("Memory Statistics:")
	Logger.Infof("  Alloc: %d MB", memStats.Alloc/1024/1024)
	Logger.Infof("  TotalAlloc: %d MB", memStats.TotalAlloc/1024/1024)
	Logger.Infof("  Sys: %d MB", memStats.Sys/1024/1024)
	Logger.Infof("  NumGC: %d (delta: +%d)", memStats.NumGC, gcCountDelta)
	Logger.Infof("  GCCPUFraction: %f", memStats.GCCPUFraction)
	Logger.Infof("  Allocated since last check: %d MB", allocatedSinceLastCheck/1024/1024)
	Logger.Infof("  Total freed: %d MB", mp.totalFreed/1024/1024)
	Logger.Infof("  HeapObjects: %d", memStats.HeapObjects)
}

// GetSummary returns a summary of memory usage
func (mp *MemoryProfiler) GetSummary() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	// Update last allocation
	mp.lastAlloc = memStats.Alloc
	if memStats.Alloc > mp.maxAlloc {
		mp.maxAlloc = memStats.Alloc
	}

	// Calculate allocation rate (bytes per second)
	var allocRate float64
	if len(mp.allocHistory) >= 2 {
		firstIdx := 0
		lastIdx := len(mp.allocHistory) - 1
		allocDiff := int64(mp.allocHistory[lastIdx] - mp.allocHistory[firstIdx])
		timeDiff := mp.allocTimestamps[lastIdx].Sub(mp.allocTimestamps[firstIdx]).Seconds()
		if timeDiff > 0 {
			allocRate = float64(allocDiff) / timeDiff
		}
	}

	return map[string]interface{}{
		"initial_alloc_mb":   mp.initialAlloc / 1024 / 1024,
		"max_alloc_mb":       mp.maxAlloc / 1024 / 1024,
		"current_alloc_mb":   mp.lastAlloc / 1024 / 1024,
		"increase_mb":        (mp.lastAlloc - mp.initialAlloc) / 1024 / 1024,
		"duration_seconds":   time.Since(mp.startTime).Seconds(),
		"snapshots":          mp.snapshots,
		"total_allocated_mb": mp.totalAllocated / 1024 / 1024,
		"total_freed_mb":     mp.totalFreed / 1024 / 1024,
		"gc_count":           mp.gcCount,
		"alloc_rate_mb_sec":  allocRate / 1024 / 1024,
		"heap_objects":       memStats.HeapObjects,
	}
}

// ForceGC forces garbage collection and logs memory usage before and after
func (mp *MemoryProfiler) ForceGC() {
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	Logger.Infof("Memory before GC: %d MB", memStatsBefore.Alloc/1024/1024)

	runtime.GC()

	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	Logger.Infof("Memory after GC: %d MB", memStatsAfter.Alloc/1024/1024)
	Logger.Infof("Memory freed by GC: %d MB", (memStatsBefore.Alloc-memStatsAfter.Alloc)/1024/1024)
}

// SetMemoryLimit sets the memory limit in bytes
func (mp *MemoryProfiler) SetMemoryLimit(limit uint64) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.memoryLimit = limit
	Logger.Infof("Memory limit set to %d MB", limit/1024/1024)
}

// CheckMemoryLimit checks if the current memory usage exceeds the limit
// Returns true if the limit is exceeded, false otherwise
func (mp *MemoryProfiler) CheckMemoryLimit() bool {
	if mp.memoryLimit == 0 {
		return false // No limit set
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.lastAlloc = memStats.Alloc
	if memStats.Alloc > mp.maxAlloc {
		mp.maxAlloc = memStats.Alloc
	}

	if memStats.Alloc > mp.memoryLimit {
		Logger.Warnf("Memory limit exceeded: %d MB used, limit is %d MB",
			memStats.Alloc/1024/1024, mp.memoryLimit/1024/1024)
		return true
	}

	return false
}

// PrintSummary prints a summary of memory usage
func (mp *MemoryProfiler) PrintSummary() {
	summary := mp.GetSummary()

	Logger.Infof("Memory Usage Summary:")
	Logger.Infof("  Initial allocation: %d MB", summary["initial_alloc_mb"])
	Logger.Infof("  Maximum allocation: %d MB", summary["max_alloc_mb"])
	Logger.Infof("  Current allocation: %d MB", summary["current_alloc_mb"])
	Logger.Infof("  Increase: %d MB", summary["increase_mb"])
	Logger.Infof("  Duration: %.2f seconds", summary["duration_seconds"])
	Logger.Infof("  Total allocated: %d MB", summary["total_allocated_mb"])
	Logger.Infof("  Total freed: %d MB", summary["total_freed_mb"])
	Logger.Infof("  GC cycles: %d", summary["gc_count"])
	Logger.Infof("  Allocation rate: %.2f MB/sec", summary["alloc_rate_mb_sec"])
	Logger.Infof("  Heap objects: %d", summary["heap_objects"])
}

// GlobalMemoryProfiler is a global instance of MemoryProfiler
var GlobalMemoryProfiler *MemoryProfiler

// InitMemoryProfiler initializes the global memory profiler
func InitMemoryProfiler() {
	GlobalMemoryProfiler = NewMemoryProfiler()
	Logger.Infof("Memory profiler initialized. Initial allocation: %d MB",
		GlobalMemoryProfiler.initialAlloc/1024/1024)
}

// StartPeriodicMemoryTracking starts periodic memory tracking
func (mp *MemoryProfiler) StartPeriodicMemoryTracking(interval time.Duration, callback func(stats map[string]interface{})) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			mp.mutex.Lock()
			// Update tracking stats
			mp.lastAlloc = memStats.Alloc
			if memStats.Alloc > mp.maxAlloc {
				mp.maxAlloc = memStats.Alloc
			}

			// Update history
			mp.allocHistory = append(mp.allocHistory, memStats.Alloc)
			mp.allocTimestamps = append(mp.allocTimestamps, time.Now())
			mp.lastHistoryUpdate = time.Now()

			// Limit history size to prevent memory growth
			if len(mp.allocHistory) > 1000 {
				mp.allocHistory = mp.allocHistory[len(mp.allocHistory)-1000:]
				mp.allocTimestamps = mp.allocTimestamps[len(mp.allocTimestamps)-1000:]
			}

			// Update totals
			mp.totalAllocated = memStats.TotalAlloc
			mp.totalFreed = memStats.TotalAlloc - memStats.Alloc
			mp.gcCount = memStats.NumGC
			mp.mutex.Unlock()

			// Call the callback with current stats
			if callback != nil {
				stats := mp.GetSummary()
				callback(stats)
			}
		}
	}()
}
