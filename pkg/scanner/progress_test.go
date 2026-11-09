package scanner

import (
	"testing"
	"time"
)

func TestProgressNotifierRateLimitsScanningUpdates(t *testing.T) {
	now := time.Unix(100, 0)
	var messages []string
	n := &progressNotifier{
		now:      func() time.Time { return now },
		interval: progressInterval,
		notify: func(message, _ string) error {
			messages = append(messages, message)
			return nil
		},
	}

	n.Notify("one", "scanning")
	n.Notify("two", "scanning")
	if len(messages) != 1 {
		t.Fatalf("got %d notifications before interval, want 1", len(messages))
	}
	now = now.Add(progressInterval)
	n.Notify("three", "scanning")
	if len(messages) != 2 {
		t.Fatalf("got %d notifications after interval, want 2", len(messages))
	}
	n.Force("complete", "complete")
	if len(messages) != 3 {
		t.Fatalf("got %d notifications after Force, want 3", len(messages))
	}
}
