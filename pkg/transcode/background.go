package transcode

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"time"
)

const renditionCompleteFile = ".complete"

// PlaybackFile prepares a segment, then renders that quality in the background.
// Manifest preparation and scan previews use File instead.
func (m *Manager) PlaybackFile(ctx context.Context, v *Video, p Profile, index int) (*CachedFile, error) {
	if p.ID == "preview" {
		return m.File(ctx, v, p, index, 1)
	}
	f, err := m.File(ctx, v, p, index, 0)
	if err == nil {
		m.startBackground(v, p, index)
	}
	return f, err
}

// startBackground protects the cache until rendering finishes.
// Call it while the requested file is still open.
func (m *Manager) startBackground(v *Video, p Profile, start int) {
	key := filepath.Join(v.Dir, p.ID)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.backgroundCtx.Err() != nil {
		return
	}
	if _, active := m.background[key]; active {
		return
	}
	if _, err := os.Stat(filepath.Join(key, renditionCompleteFile)); err == nil {
		return
	}
	unlock, err := fileLock(m.backgroundCtx, v.Dir+".lock", false, false)
	if err != nil {
		m.conf.Logger.Warn("video background render could not start", "media", v.Hash, "profile", p.ID, "error", err)
		return
	}
	m.background[key] = struct{}{}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer func() {
			unlock()
			m.mu.Lock()
			delete(m.background, key)
			m.mu.Unlock()
		}()
		started := time.Now()
		m.conf.Logger.Info("video background render started", "media", v.Hash, "profile", p.ID, "start_segment", start)
		if err := m.renderRemaining(m.backgroundCtx, v, p, start); err != nil {
			if !errors.Is(err, context.Canceled) {
				m.conf.Logger.Warn("video background render failed", "media", v.Hash, "profile", p.ID, "error", err)
			}
			return
		}
		m.conf.Logger.Info("video background render complete", "media", v.Hash, "profile", p.ID, "seconds", time.Since(started).Seconds())
	}()
}

// renderRemaining fills missing segments ahead of playback, then earlier gaps.
func (m *Manager) renderRemaining(ctx context.Context, v *Video, p Profile, start int) error {
	count := int(math.Ceil(v.Source.Duration / SegmentDuration))
	// Render ahead first, then fill earlier gaps, one segment at a time.
	for offset := 1; offset <= count; offset++ {
		if err := m.backgroundFile(ctx, v, p, (start+offset)%count); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// An empty file marks completion and is evicted with the video.
	// Interrupted renders keep completed segments for reuse.
	f, err := os.OpenFile(filepath.Join(v.Dir, p.ID, renditionCompleteFile), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

// backgroundFile queues one background segment and retries when the queue is full.
func (m *Manager) backgroundFile(ctx context.Context, v *Video, p Profile, index int) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Limit background jobs to leave queue space for playback.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case m.backgroundSlots <- struct{}{}:
		}
		f, err := m.File(ctx, v, p, index, 2)
		if err == nil {
			err = f.Close()
		}
		<-m.backgroundSlots
		if !errors.Is(err, ErrBusy) {
			return err
		}
		// Retry a full queue, but allow shutdown to cancel the wait.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
