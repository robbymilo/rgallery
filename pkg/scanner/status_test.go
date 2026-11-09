package scanner

import (
	"testing"
	"time"
)

func TestCancelScanKeepsSlotOwnedUntilCleanup(t *testing.T) {
	SetScanInProgress(false)

	ctx, ok := beginScan()
	if !ok {
		t.Fatal("beginScan did not claim an idle scanner")
	}
	if !CancelScan() {
		t.Fatal("CancelScan did not cancel the active scan")
	}
	if !isCanceled() {
		t.Fatal("scan context was not canceled")
	}
	if !IsScanInProgress() {
		t.Fatal("cancellation released the scan slot before cleanup")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("scan context did not finish")
	}

	endScan()
	if IsScanInProgress() {
		t.Fatal("endScan did not release the scan slot")
	}
	if CancelScan() {
		t.Fatal("CancelScan succeeded after cleanup")
	}
}

func TestCanceledScanCannotCancelNextScan(t *testing.T) {
	SetScanInProgress(false)
	first, ok := beginScan()
	if !ok {
		t.Fatal("first scan did not start")
	}
	if !CancelScan() {
		t.Fatal("first scan did not cancel")
	}
	endScan()

	second, ok := beginScan()
	if !ok {
		t.Fatal("second scan did not start")
	}
	defer endScan()
	if second.Err() != nil {
		t.Fatal("new scan inherited cancellation from old scan")
	}
	if first.Err() == nil {
		t.Fatal("old scan was not canceled")
	}
}
