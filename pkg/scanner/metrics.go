package scanner

import (
	"runtime"
	"sync"
	"time"
)

// ScanMetrics is a performance snapshot for one scan. Durations are reported
// as time.Duration values so callers can use them directly in benchmarks.
// Counts intentionally include work done by failed files; that work still
// affects scan throughput and is useful when comparing retries.
type ScanMetrics struct {
	ScanType string

	Duration            time.Duration
	FilesystemDiscovery time.Duration
	FilesystemStat      time.Duration
	RawMetadata         time.Duration
	ConvertedLens       time.Duration
	ImageProcessing     time.Duration
	ThumbnailGeneration time.Duration
	ReverseGeocoding    time.Duration
	DatabaseWrites      time.Duration

	FilesDiscovered          int
	StatCalls                int
	RawMetadataFiles         int
	ConvertedLensFiles       int
	ImageProcessingFiles     int
	ThumbnailCount           int
	ReverseGeocodeCalls      int
	DatabaseTransactions     int
	DatabaseWriteCount       int
	CacheWrites              int
	ExternalProcessLaunches  int
	PeakMemoryBytes          uint64
	MediaProcessed           int
	ThroughputMediaPerSecond float64
	ThroughputFilesPerSecond float64
}

type scanMetrics struct {
	mu      sync.Mutex
	started time.Time
	data    ScanMetrics
}

func newScanMetrics(scanType string) *scanMetrics {
	m := &scanMetrics{started: time.Now()}
	m.data.ScanType = scanType
	m.sampleMemory()
	return m
}

func (m *scanMetrics) addDuration(field *time.Duration, elapsed time.Duration) {
	m.mu.Lock()
	*field += elapsed
	m.mu.Unlock()
}

func (m *scanMetrics) sampleMemory() {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	m.mu.Lock()
	if stats.Alloc > m.data.PeakMemoryBytes {
		m.data.PeakMemoryBytes = stats.Alloc
	}
	m.mu.Unlock()
}

func (m *scanMetrics) snapshot() ScanMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data
}

func (m *scanMetrics) finish() {
	m.sampleMemory()
	m.mu.Lock()
	m.data.Duration = time.Since(m.started)
	if m.data.Duration > 0 {
		m.data.ThroughputMediaPerSecond = float64(m.data.MediaProcessed) / m.data.Duration.Seconds()
		m.data.ThroughputFilesPerSecond = float64(m.data.FilesDiscovered) / m.data.Duration.Seconds()
	}
	m.mu.Unlock()
}

var lastScanMetrics struct {
	sync.RWMutex
	value ScanMetrics
}

func setLastScanMetrics(m ScanMetrics) {
	lastScanMetrics.Lock()
	lastScanMetrics.value = m
	lastScanMetrics.Unlock()
}

// LastScanMetrics returns a copy of the most recent scan metrics snapshot.
// It is primarily intended for diagnostics and benchmark assertions.
func LastScanMetrics() ScanMetrics {
	lastScanMetrics.RLock()
	defer lastScanMetrics.RUnlock()
	return lastScanMetrics.value
}

// The following methods form the small instrumentation interface consumed by
// the exif package. Keeping the interface there avoids an import cycle.
func (m *scanMetrics) Metadata(elapsed time.Duration) {
	m.addDuration(&m.data.RawMetadata, elapsed)
	m.mu.Lock()
	m.data.RawMetadataFiles++
	m.mu.Unlock()
	m.sampleMemory()
}

func (m *scanMetrics) Lens(elapsed time.Duration) {
	m.addDuration(&m.data.ConvertedLens, elapsed)
	m.mu.Lock()
	m.data.ConvertedLensFiles++
	m.mu.Unlock()
	m.sampleMemory()
}

func (m *scanMetrics) ImageProcessing(elapsed time.Duration) {
	m.addDuration(&m.data.ImageProcessing, elapsed)
	m.mu.Lock()
	m.data.ImageProcessingFiles++
	m.mu.Unlock()
	m.sampleMemory()
}

func (m *scanMetrics) ReverseGeocode(elapsed time.Duration) {
	m.addDuration(&m.data.ReverseGeocoding, elapsed)
	m.mu.Lock()
	m.data.ReverseGeocodeCalls++
	m.mu.Unlock()
	m.sampleMemory()
}

func (m *scanMetrics) ExternalProcess() {
	m.mu.Lock()
	m.data.ExternalProcessLaunches++
	m.mu.Unlock()
}
