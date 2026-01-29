package scanner

import (
	"sync"
)

var (
	// ScanInProgress tracks whether a scan is currently in progress
	scanInProgress bool
	// mutex to protect access to scanInProgress
	scanMutex sync.RWMutex
	// scanCancel is used to signal cancellation of a running scan
	scanCancel chan struct{}
	// mutex to protect access to scanCancel
	scanCancelMutex sync.RWMutex
)

// IsScanInProgress returns true if a scan is currently in progress
func IsScanInProgress() bool {
	scanMutex.RLock()
	defer scanMutex.RUnlock()
	return scanInProgress
}

// SetScanInProgress sets the scan in progress status
func SetScanInProgress(status bool) {
	scanMutex.Lock()
	defer scanMutex.Unlock()
	scanInProgress = status
}

// CancelScan requests cancellation of a running scan.
func CancelScan() bool {
	scanCancelMutex.RLock()
	ch := scanCancel
	scanCancelMutex.RUnlock()

	if ch == nil {
		return false
	}

	// Protect write to avoid races with resetCancelChan
	scanCancelMutex.Lock()
	if scanCancel != nil {
		close(scanCancel)
	}
	scanCancelMutex.Unlock()

	// Immediately set scan in progress to false so UI updates right away
	SetScanInProgress(false)

	return true
}

// resetCancelChan sets the package cancel channel
func resetCancelChan(ch chan struct{}) {
	scanCancelMutex.Lock()
	defer scanCancelMutex.Unlock()
	scanCancel = ch
}

// isCanceled returns true if the current cancel channel is closed.
func isCanceled() bool {
	scanCancelMutex.RLock()
	ch := scanCancel
	scanCancelMutex.RUnlock()

	if ch == nil {
		return false
	}

	select {
	case <-ch:
		return true
	default:
		return false
	}
}
