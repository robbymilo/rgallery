package scanner

import (
	"context"
	"sync"
)

var (
	// ScanInProgress tracks whether a scan is currently in progress.
	scanInProgress bool
	scanStateMutex sync.Mutex
	scanCancel     context.CancelFunc
	scanContext    context.Context
)

// IsScanInProgress returns true if a scan is currently in progress.
func IsScanInProgress() bool {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	return scanInProgress
}

// SetScanInProgress preserves the legacy status API. New scans should use
// beginScan/endScan so claiming and cancellation are atomic.
func SetScanInProgress(status bool) {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	scanInProgress = status
	if !status {
		if scanCancel != nil {
			scanCancel()
		}
		scanCancel = nil
		scanContext = nil
	}
}

// beginScan claims the single-process scan slot and returns a context owned by
// that scan. Keeping the cancel function with the slot prevents an old scan
// from canceling a later scan after cleanup.
func beginScan() (context.Context, bool) {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	if scanInProgress {
		return nil, false
	}
	scanContext, scanCancel = context.WithCancel(context.Background())
	scanInProgress = true
	return scanContext, true
}

func endScan() {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	scanInProgress = false
	if scanCancel != nil {
		scanCancel()
	}
	scanCancel = nil
	scanContext = nil
}

// CancelScan requests cancellation of a running scan. The scan remains in
// progress until its worker reaches cleanup, so a replacement scan cannot race
// with resources owned by the canceled worker.
func CancelScan() bool {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	if !scanInProgress || scanCancel == nil {
		return false
	}
	scanCancel()
	return true
}

func currentScanContext() context.Context {
	scanStateMutex.Lock()
	defer scanStateMutex.Unlock()
	return scanContext
}

// isCanceled is retained for package-local compatibility while callers are
// migrated to context checks at operation boundaries.
func isCanceled() bool {
	ctx := currentScanContext()
	return ctx != nil && ctx.Err() != nil
}
