package scanner

import (
	"sync"
	"time"

	"github.com/robbymilo/rgallery/pkg/queries"
)

const progressInterval = 250 * time.Millisecond

// progressNotifier coalesces per-file updates. Start, cancellation, and
// completion notifications remain explicit calls and are never delayed.
type progressNotifier struct {
	mu       sync.Mutex
	last     time.Time
	now      func() time.Time
	interval time.Duration
	notify   func(string, string) error
}

func newProgressNotifier(c Conf) *progressNotifier {
	return &progressNotifier{
		now:      time.Now,
		interval: progressInterval,
		notify: func(message, state string) error {
			return queries.Notify(c, message, state)
		},
	}
}

func (p *progressNotifier) Notify(message, state string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	if state == "scanning" && !p.last.IsZero() && now.Sub(p.last) < p.interval {
		return
	}
	p.last = now
	if err := p.notify(message, state); err != nil {
		// Notification failures must not abort ingestion.
		return
	}
}

func (p *progressNotifier) Force(message, state string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.last = p.now()
	_ = p.notify(message, state)
}
