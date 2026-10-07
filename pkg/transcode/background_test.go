package transcode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/types"
)

// awaitBackground waits for a background test condition or fails on timeout.
func awaitBackground(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for background rendering")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// backgroundIdle reports whether all background passes have finished.
func backgroundIdle(m *Manager) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.background) == 0
}

// TestPlaybackFinishesRenditionAfterRequestEnds checks that rendering continues after a viewer leaves.
func TestPlaybackFinishesRenditionAfterRequestEnds(t *testing.T) {
	source := fixture(t, true)
	for _, tc := range []struct {
		name   string
		start  int
		cached bool
	}{{"cold", 0, false}, {"cached resume", 3, true}} {
		t.Run(tc.name, func(t *testing.T) {
			var logs strings.Builder
			c := Conf{Cache: t.TempDir(), Logger: slog.New(slog.NewTextHandler(&logs, nil)), Transcode: types.TranscodeConfig{Encoder: "cpu", Workers: 1}}
			m := NewManager(c)
			t.Cleanup(m.Close)
			v, err := m.Open(context.Background(), source, 42)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := FindProfile(c, "small")
			if tc.cached {
				f, err := m.File(context.Background(), v, p, tc.start, 0)
				if err != nil {
					t.Fatal(err)
				}
				_ = f.Close()
			}
			if err := os.MkdirAll(filepath.Join(v.Dir, p.ID), 0755); err != nil {
				t.Fatal(err)
			}
			// Block the next segment to check that the first request can finish independently.
			next := filepath.Join(v.Dir, p.ID, fmt.Sprintf("%06d.ts", tc.start+1))
			unlock, err := fileLock(context.Background(), next+".lock", true, false)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { unlock() }()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			f, err := m.PlaybackFile(ctx, v, p, tc.start)
			if err != nil {
				t.Fatal(err)
			}
			first, err := f.Stat()
			_ = f.Close()
			if err != nil {
				t.Fatal(err)
			}
			cancel()
			awaitBackground(t, func() bool {
				m.mu.Lock()
				defer m.mu.Unlock()
				return m.jobs[next] != nil && m.jobs[next].started
			})
			if validOutput(next) {
				t.Fatal("blocked segment was published")
			}
			if release, err := fileLock(context.Background(), v.Dir+".lock", true, false); err == nil {
				release()
				t.Fatal("background rendition was not protected from eviction")
			}
			// A second viewer must share the background render.
			f, err = m.PlaybackFile(context.Background(), v, p, tc.start)
			if err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
			unlock()
			unlock = func() {}
			awaitBackground(t, func() bool { return backgroundIdle(m) })
			marker := filepath.Join(v.Dir, p.ID, renditionCompleteFile)
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("rendition did not finish: %v", err)
			}
			for i := 0; i < 6; i++ {
				path := filepath.Join(v.Dir, p.ID, fmt.Sprintf("%06d.ts", i))
				if !validOutput(path) {
					t.Fatalf("missing unrequested segment %d", i)
				}
			}
			// Only the requested quality should be rendered.
			for _, profile := range []string{"saver", "high", "preview"} {
				if _, err := os.Stat(filepath.Join(v.Dir, profile)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("unexpected output for %s: %v", profile, err)
				}
			}
			f, err = m.PlaybackFile(context.Background(), v, p, tc.start)
			if err != nil {
				t.Fatal(err)
			}
			cached, _ := f.Stat()
			_ = f.Close()
			if !cached.ModTime().Equal(first.ModTime()) || !backgroundIdle(m) {
				t.Fatal("complete rendition restarted work or rewrote a cached segment")
			}
			m.Close()
			// After restart, the completion marker should prevent another cache scan.
			restarted := NewManager(c)
			defer restarted.Close()
			f, err = restarted.PlaybackFile(context.Background(), v, p, tc.start)
			if err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
			restarted.Close()
			if strings.Count(logs.String(), "video background render started") != 1 || strings.Count(logs.String(), "video background render complete") != 1 {
				t.Fatalf("duplicate or incomplete background pass: %s", logs.String())
			}
			// Check that rendering follows the saved position, then fills earlier gaps.
			last := -1
			for offset := 1; offset < 6; offset++ {
				i := (tc.start + offset) % 6
				pos := strings.Index(logs.String(), fmt.Sprintf("segment=%d resolution=", i))
				if pos <= last {
					t.Fatalf("segment %d encoded out of playback order: %s", i, logs.String())
				}
				last = pos
			}
		})
	}
}

// backgroundTestVideo creates cached data for tests that do not need FFmpeg.
func backgroundTestVideo(t *testing.T, m *Manager) (*Video, Profile) {
	t.Helper()
	dir, err := os.MkdirTemp(t.TempDir(), "video-")
	if err != nil {
		t.Fatal(err)
	}
	v := &Video{Dir: dir, Hash: 42, Source: Source{Duration: 6}}
	p, _ := FindProfile(m.conf, "small")
	if err := os.MkdirAll(filepath.Join(v.Dir, p.ID), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Dir, p.ID, "000000.ts"), []byte("cached"), 0600); err != nil {
		t.Fatal(err)
	}
	return v, p
}

// TestBackgroundShutdownAndPlaybackPriority checks foreground priority and clean shutdown.
func TestBackgroundShutdownAndPlaybackPriority(t *testing.T) {
	m := NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "cpu", Workers: 1}})
	t.Cleanup(m.Close)
	v, p := backgroundTestVideo(t, m)
	started, release := make(chan struct{}), make(chan struct{})
	blocker := make(chan error, 1)
	go func() {
		blocker <- m.do(context.Background(), "occupy-worker", 0, func(ctx context.Context) error {
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	<-started
	f, err := m.PlaybackFile(context.Background(), v, p, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	next := filepath.Join(v.Dir, p.ID, "000001.ts")
	awaitBackground(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.jobs[next] != nil && m.jobs[next].priority == 2
	})
	foregroundStarted := make(chan struct{})
	foreground := make(chan error, 1)
	go func() {
		foreground <- m.do(context.Background(), "foreground-seek", 0, func(ctx context.Context) error {
			close(foregroundStarted)
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	awaitBackground(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.jobs["foreground-seek"] != nil
	})
	close(release)
	if err := <-blocker; err != nil {
		t.Fatal(err)
	}
	select {
	case <-foregroundStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("foreground playback did not run")
	}
	m.mu.Lock()
	nextJob := m.jobs[next]
	queued := nextJob != nil && !nextJob.started
	m.mu.Unlock()
	if !queued {
		t.Fatal("background encode ran ahead of a queued playback request")
	}
	closed := make(chan struct{})
	go func() { m.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not cancel background rendering")
	}
	if err := <-foreground; !errors.Is(err, context.Canceled) {
		t.Fatalf("foreground shutdown: %v", err)
	}
	if !backgroundIdle(m) {
		t.Fatal("background pass survived shutdown")
	}
	if _, err := os.Stat(filepath.Join(v.Dir, p.ID, renditionCompleteFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("interrupted pass marked complete: %v", err)
	}
	unlock, err := fileLock(context.Background(), v.Dir+".lock", true, false)
	if err != nil {
		t.Fatalf("shutdown leaked the asset lock: %v", err)
	}
	unlock()
}

// TestPreviewAndInvalidPlaybackDoNotStartBackground checks that only valid playback starts background work.
func TestPreviewAndInvalidPlaybackDoNotStartBackground(t *testing.T) {
	m := testManager(t)
	v, p := backgroundTestVideo(t, m)
	if _, err := m.PlaybackFile(context.Background(), v, p, 3); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid segment: %v", err)
	}
	preview, _ := FindProfile(m.conf, "preview")
	if err := os.MkdirAll(filepath.Join(v.Dir, preview.ID), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Dir, preview.ID, "preview.mp4"), []byte("cached preview"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := m.PlaybackFile(context.Background(), v, preview, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if !backgroundIdle(m) {
		t.Fatal("preview or invalid segment started rendering a full rendition")
	}
}

// TestBackgroundFailureCanRetry checks that failed background work can be retried.
func TestBackgroundFailureCanRetry(t *testing.T) {
	m := testManager(t)
	v, p := backgroundTestVideo(t, m)
	next := filepath.Join(v.Dir, p.ID, "000001.ts")
	started, fail := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- m.do(context.Background(), next, 2, func(ctx context.Context) error {
			close(started)
			select {
			case <-fail:
				return errors.New("test encode failure")
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	<-started
	f, err := m.PlaybackFile(context.Background(), v, p, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	awaitBackground(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.jobs[next].refs == 2
	})
	close(fail)
	if err := <-done; err == nil {
		t.Fatal("expected encode failure")
	}
	awaitBackground(t, func() bool { return backgroundIdle(m) })
	marker := filepath.Join(v.Dir, p.ID, renditionCompleteFile)
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed pass marked complete: %v", err)
	}
	// Fill the cache as another process would, then check that playback reuses it.
	for i := 1; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(v.Dir, p.ID, fmt.Sprintf("%06d.ts", i)), []byte("cached"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	f, err = m.PlaybackFile(context.Background(), v, p, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	awaitBackground(t, func() bool { return backgroundIdle(m) })
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("retry did not complete: %v", err)
	}
}

// TestBackgroundSubmissionsAreBoundedAcrossVideos checks the shared limit on background jobs.
func TestBackgroundSubmissionsAreBoundedAcrossVideos(t *testing.T) {
	m := NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "cpu", Workers: 1}})
	t.Cleanup(m.Close)
	started, done := make(chan struct{}), make(chan error, 1)
	go func() {
		done <- m.do(context.Background(), "occupy-worker", 0, func(ctx context.Context) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	<-started
	for i := 0; i < 8; i++ {
		v, p := backgroundTestVideo(t, m)
		f, err := m.PlaybackFile(context.Background(), v, p, 0)
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	awaitBackground(t, func() bool {
		m.mu.Lock()
		defer m.mu.Unlock()
		return len(m.jobs) == 2
	})
	m.mu.Lock()
	active, queued := len(m.background), len(m.jobs)
	m.mu.Unlock()
	if active != 8 || queued != 2 {
		t.Fatalf("background submissions were not bounded: %d active renditions, %d jobs", active, queued)
	}
	closed := make(chan struct{})
	go func() { m.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown left background passes waiting for submission slots")
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("worker shutdown: %v", err)
	}
	if !backgroundIdle(m) {
		t.Fatal("background passes survived shutdown")
	}
}
