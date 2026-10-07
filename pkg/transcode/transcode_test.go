package transcode

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/robbymilo/rgallery/pkg/types"
)

// fixture creates a short test video with optional audio.
func fixture(t *testing.T, audio bool) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is required for video integration tests")
	}
	path := filepath.Join(t.TempDir(), "source.mp4")
	args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30"}
	if audio {
		args = append(args, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=48000")
	}
	args = append(args, "-t", "10.2", "-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-c:a", "aac", "-y", path)
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("fixture: %v: %s", err, output)
	}
	return path
}

// testManager creates a CPU encoder manager and closes it after the test.
func testManager(t *testing.T) *Manager {
	t.Helper()
	m := NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "cpu"}})
	t.Cleanup(m.Close)
	return m
}

// TestDimensionsAndProfileBudgets checks scaling, quality limits, and invalid settings.
func TestDimensionsAndProfileBudgets(t *testing.T) {
	for _, tc := range []struct{ w, h, edge, wantW, wantH int }{{3840, 2160, 1280, 1280, 720}, {2160, 3840, 1280, 720, 1280}, {640, 480, 1280, 640, 480}, {403, 301, 400, 400, 298}} {
		w, h := Dimensions(tc.w, tc.h, tc.edge)
		if w != tc.wantW || h != tc.wantH {
			t.Fatalf("%+v: got %dx%d", tc, w, h)
		}
	}
	c := Conf{TranscodeResolution: 1280, Transcode: types.TranscodeConfig{MaxRate: 500}}
	for _, p := range Profiles(c) {
		if p.LongEdge > 1280 || p.MaxRate > 500 {
			t.Fatalf("profile exceeds cap: %+v", p)
		}
	}
	for _, cfg := range []types.TranscodeConfig{{Profile: "invalid"}, {Mode: "invalid"}, {Encoder: "invalid"}, {CRF: 60, Profile: "small"}, {Workers: 17}, {MaxRate: -1}} {
		if Validate(Conf{Transcode: cfg}) == nil {
			t.Fatalf("accepted invalid configuration: %+v", cfg)
		}
	}
}

// TestSeekGeneratesOnlyRequestedSegmentAndReusesCache checks direct seeking and cached segment reuse.
func TestSeekGeneratesOnlyRequestedSegmentAndReusesCache(t *testing.T) {
	for _, audio := range []bool{false, true} {
		t.Run(fmt.Sprint(audio), func(t *testing.T) {
			m := testManager(t)
			v, err := m.Open(context.Background(), fixture(t, audio), 42)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := FindProfile(m.conf, "small")
			started := time.Now()
			f, err := m.File(context.Background(), v, p, 4, 0)
			if err != nil {
				t.Fatal(err)
			}
			path := f.Name()
			info, _ := f.Stat()
			_ = f.Close()
			t.Logf("uncached seek at 8s: %s, %d bytes", time.Since(started), info.Size())
			if validOutput(filepath.Join(v.Dir, p.ID, "000000.ts")) {
				t.Fatal("seek encoded the beginning of the video")
			}
			probe, err := Probe(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			if probe.Width != 320 || probe.Height != 180 || probe.Audio != audio {
				t.Fatalf("unexpected output: %+v", probe)
			}
			if probe.Duration < 1.95 || probe.Duration > 2.1 {
				t.Fatalf("segment duration: %f", probe.Duration)
			}
			f, err = m.File(context.Background(), v, p, 4, 0)
			if err != nil {
				t.Fatal(err)
			}
			cached, _ := f.Stat()
			_ = f.Close()
			if !cached.ModTime().Equal(info.ModTime()) {
				t.Fatal("cached segment was re-encoded")
			}
			playlist := Playlist(v, p)
			if !strings.Contains(playlist, "#EXTINF:0.200000") || !strings.Contains(playlist, "#EXT-X-ENDLIST") || strings.Contains(playlist, "#EXT-X-DISCONTINUITY") {
				t.Fatalf("invalid VOD timeline: %s", playlist)
			}
			if _, err := m.File(context.Background(), v, p, 6, 0); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid index: %v", err)
			}
		})
	}
}

// TestSourceAndSettingsInvalidateCache checks that source and setting changes use fresh cache entries.
func TestSourceAndSettingsInvalidateCache(t *testing.T) {
	m := testManager(t)
	path := fixture(t, false)
	v, err := m.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	newTime := time.Now().Add(time.Second)
	if err := os.Chtimes(path, newTime, newTime); err != nil {
		t.Fatal(err)
	}
	changed, err := m.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Version == v.Version {
		t.Fatal("source change reused old cache")
	}
	c := m.conf
	c.Transcode.MaxRate = 400
	other := NewManager(c)
	defer other.Close()
	configured, err := other.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if configured.Version == changed.Version {
		t.Fatal("settings change reused old cache")
	}
	crfConfig := m.conf
	crfConfig.Transcode.Profile = "small"
	crfConfig.Transcode.CRF = 35
	crfManager := NewManager(crfConfig)
	defer crfManager.Close()
	crfVideo, err := crfManager.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	if crfVideo.Version == changed.Version {
		t.Fatal("CRF change reused old cache")
	}
}

// TestPortraitAndHDRInput checks portrait orientation and HDR conversion.
func TestPortraitAndHDRInput(t *testing.T) {
	for _, kind := range []string{"portrait", "hdr"} {
		t.Run(kind, func(t *testing.T) {
			m := testManager(t)
			if device := os.Getenv("RGALLERY_TEST_GPU"); device != "" {
				m = NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "vaapi", Device: device}})
				t.Cleanup(m.Close)
			}
			path := filepath.Join(t.TempDir(), "input.mp4")
			var args []string
			if kind == "portrait" {
				args = []string{"-hide_banner", "-loglevel", "error", "-display_rotation", "90", "-i", fixture(t, false), "-c", "copy", "-y", path}
			} else {
				args = []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30", "-t", "2.2", "-pix_fmt", "yuv420p10le", "-c:v", "libx265", "-x265-params", "log-level=error:pools=1:frame-threads=1:colorprim=bt2020:transfer=smpte2084:colormatrix=bt2020nc", "-color_primaries", "bt2020", "-color_trc", "smpte2084", "-colorspace", "bt2020nc", "-y", path}
			}
			out, err := exec.Command("ffmpeg", args...).CombinedOutput()
			if err != nil {
				t.Fatalf("fixture %s: %v: %s", kind, err, out)
			}
			v, err := m.Open(context.Background(), path, 10)
			if err != nil {
				t.Fatal(err)
			}
			p, _ := FindProfile(m.conf, "small")
			f, err := m.File(context.Background(), v, p, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			s, err := Probe(context.Background(), f.Name())
			if err != nil {
				t.Fatal(err)
			}
			if s.HDR || s.PixelFormat != "yuv420p" {
				t.Fatalf("output is not compatible SDR: %+v", s)
			}
			if kind == "portrait" && (s.Width != 180 || s.Height != 320) {
				t.Fatalf("rotation lost: %+v", s)
			}
			if kind == "hdr" && !v.Source.HDR {
				t.Fatal("HDR input was not detected")
			}
			if os.Getenv("RGALLERY_TEST_GPU") != "" && m.Diagnostics().Encoder != "h264_vaapi" {
				t.Fatal("hardware pipeline fell back to CPU")
			}
		})
	}
}

// TestVAAPIIntegration checks real GPU encoding when a render device is available.
func TestVAAPIIntegration(t *testing.T) {
	device := os.Getenv("RGALLERY_TEST_GPU")
	if device == "" {
		t.Skip("set RGALLERY_TEST_GPU to validate actual VA-API hardware")
	}
	m := NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "vaapi", Device: device}})
	defer m.Close()
	if cap := m.encoder.detect(m.conf); cap.Encoder != "h264_vaapi" {
		t.Fatalf("hardware unavailable: %+v", cap)
	}
	v, err := m.Open(context.Background(), fixture(t, true), 55)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(m.conf, "small")
	// Check GPU decoding and scaling as well as encoding.
	hardwareOutput := filepath.Join(t.TempDir(), "hardware.ts")
	args := encodeArgs(v.Source, p, 6, SegmentDuration, hardwareOutput, Settings(m.conf).Preset, m.encoder.detect(m.conf), true, false)
	if _, err := runFFmpeg(context.Background(), args); err != nil {
		t.Fatalf("hardware decode/scale failed: %v", err)
	}
	f, err := m.File(context.Background(), v, p, 3, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s, err := Probe(context.Background(), f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if s.Width != 320 || s.Height != 180 || !s.Audio {
		t.Fatalf("invalid hardware output: %+v", s)
	}
	d := m.Diagnostics()
	if d.Encoder != "h264_vaapi" {
		t.Fatalf("hardware pipeline fell back: %+v", d)
	}
	t.Logf("VA-API: %s, %.2fs encode, %.1fx playback speed", d.Device, d.LastEncodeSeconds, d.LastEncodeSpeed)
}

// TestSharedJobCancellationAndPriority checks that one viewer leaving does not cancel shared work.
func TestSharedJobCancellationAndPriority(t *testing.T) {
	m := testManager(t)
	var calls atomic.Int32
	started := make(chan struct{})
	finish := make(chan struct{})
	firstCtx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	second := make(chan error, 1)
	work := func(ctx context.Context) error {
		calls.Add(1)
		close(started)
		select {
		case <-finish:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	go func() { first <- m.do(firstCtx, "shared", 2, work) }()
	<-started
	go func() { second <- m.do(context.Background(), "shared", 0, work) }()
	deadline := time.Now().Add(time.Second)
	for {
		m.mu.Lock()
		refs := m.jobs["shared"].refs
		m.mu.Unlock()
		if refs == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second viewer did not join")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	close(finish)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate encoding")
	}
}

// TestWorkerLimitAndFailedJobsCanRetry checks worker limits and retries after failure.
func TestWorkerLimitAndFailedJobsCanRetry(t *testing.T) {
	m := testManager(t)
	var active, peak atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = m.do(context.Background(), fmt.Sprint(i), 0, func(context.Context) error {
				n := active.Add(1)
				for {
					old := peak.Load()
					if n <= old || peak.CompareAndSwap(old, n) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				active.Add(-1)
				return nil
			})
		}(i)
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatalf("worker cap exceeded: %d", peak.Load())
	}
	if err := m.do(context.Background(), "retry", 0, func(context.Context) error { return errors.New("expected failure") }); err == nil {
		t.Fatal("missing failure")
	}
	if err := m.do(context.Background(), "retry", 0, func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

// TestPreviewIsShortSilentAndGPUFallsBack checks preview limits and CPU fallback.
func TestPreviewIsShortSilentAndGPUFallsBack(t *testing.T) {
	m := NewManager(Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "vaapi", Device: "/definitely/not/a/render/device"}})
	defer m.Close()
	v, err := m.Open(context.Background(), fixture(t, true), 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(m.conf, "preview")
	f, err := m.File(context.Background(), v, p, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	s, err := Probe(context.Background(), f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if s.Duration > 4.1 || s.Audio || max(s.Width, s.Height) > p.LongEdge {
		t.Fatalf("invalid preview: %+v", s)
	}
	d := m.Diagnostics()
	if d.Encoder != "libx264" || d.Fallback == "" {
		t.Fatalf("missing hardware fallback: %+v", d)
	}
}

// TestEncoderLogsSelectionAndActualCPUFallback checks encoder selection and fallback logs.
func TestEncoderLogsSelectionAndActualCPUFallback(t *testing.T) {
	var logs strings.Builder
	c := Conf{Cache: t.TempDir(), Logger: slog.New(slog.NewTextHandler(&logs, nil)), TranscodeResolution: 1920, Transcode: types.TranscodeConfig{Profile: "high", CRF: -1, Encoder: "vaapi", Device: "/missing/render-device"}}
	m := NewManager(c)
	defer m.Close()
	m.LogConfiguration()
	v, err := m.Open(context.Background(), fixture(t, false), 9)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(c, "high")
	f, err := m.File(context.Background(), v, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	// Stop workers before reading their logs.
	m.Close()
	for _, want := range []string{"video GPU probe failed", "requested_encoder=vaapi", "encoder=libx264", "fallback=", "video transcoding configured", "quality=high", "max_video_kbps=30000", "video encode completed", "hardware_decode=false", "profile=high"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("missing %q in logs:\n%s", want, logs.String())
		}
	}
}

// TestPregenerateCreatesPreviewsAndSeekThumbnailsInOnDemandMode checks scan previews without full playback rendering.
func TestPregenerateCreatesPreviewsAndSeekThumbnailsInOnDemandMode(t *testing.T) {
	c := Conf{Cache: t.TempDir(), PreGenerateThumb: true, Transcode: types.TranscodeConfig{Encoder: "cpu"}}
	m := For(c)
	defer m.Close()
	path := fixture(t, true)
	if err := Pregenerate(context.Background(), path, 5, c); err != nil {
		t.Fatal(err)
	}
	v, err := m.Open(context.Background(), path, 5)
	if err != nil {
		t.Fatal(err)
	}
	preview := filepath.Join(v.Dir, "preview", "preview.mp4")
	if !validOutput(preview) {
		t.Fatal("scan did not prepare a preview")
	}
	sheet := filepath.Join(v.Dir, thumbnailDirectory, "000000.jpg")
	if !validOutput(sheet) {
		t.Fatal("scan did not prepare seek thumbnails")
	}
	sheetBefore, _ := os.Stat(sheet)
	for _, p := range Profiles(c) {
		if validOutput(filepath.Join(v.Dir, p.ID, "000000.ts")) {
			t.Fatalf("on-demand scan encoded full playback profile %s", p.ID)
		}
	}
	before, _ := os.Stat(preview)
	if err := Pregenerate(context.Background(), path, 5, c); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(preview)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("scan regenerated a current preview")
	}
	sheetAfter, _ := os.Stat(sheet)
	if !sheetBefore.ModTime().Equal(sheetAfter.ModTime()) {
		t.Fatal("scan regenerated current seek thumbnails")
	}
	c.PreGenerateThumb = false
	if err := Pregenerate(context.Background(), "/missing/source.mp4", 1, c); err != nil {
		t.Fatal("disabled pregeneration still accessed the source")
	}
}

// TestEvictionProtectsActiveFiles checks that cleanup preserves files still in use.
func TestEvictionProtectsActiveFiles(t *testing.T) {
	m := testManager(t)
	v, err := m.Open(context.Background(), fixture(t, false), 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(m.conf, "small")
	f, err := m.File(context.Background(), v, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(v.Source.Path); err != nil {
		t.Fatal(err)
	}
	m.Cleanup()
	if !validOutput(f.Name()) {
		t.Fatal("evicted an active stream")
	}
	_ = f.Close()
	m.Cleanup()
	if _, err := os.Stat(v.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted source cache survived: %v", err)
	}
}

// TestCacheRetention checks that old and large cache files survive cleanup.
func TestCacheRetention(t *testing.T) {
	c := Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "cpu"}}
	m := NewManager(c)
	path := fixture(t, false)
	v, err := m.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(c, "small")
	f, err := m.File(context.Background(), v, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	output := f.Name()
	_ = f.Close()
	m.Close()
	m = NewManager(c)
	defer m.Close()
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(v.Dir, old, old); err != nil {
		t.Fatal(err)
	}
	v2, err := m.Open(context.Background(), path, 2)
	if err != nil {
		t.Fatal(err)
	}
	active, err := m.File(context.Background(), v2, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = active.Close() }()
	// Use sparse files to test cache sizes without a large video.
	for _, path := range []string{output, active.Name()} {
		if err := os.Truncate(path, 11*1024*1024*1024); err != nil {
			t.Fatal(err)
		}
	}
	m.Cleanup()
	if !validOutput(output) || !validOutput(active.Name()) {
		t.Fatal("cleanup removed unchanged cached output")
	}
	changed := time.Now().Add(time.Hour)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Open(context.Background(), path, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(v.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("opening a changed source retained its old cache: %v", err)
	}
	if !validOutput(active.Name()) {
		t.Fatal("source change removed active output")
	}
}

// TestIndependentChunksKeepSmallBitrate checks that compact segments stay within their bitrate budget.
func TestIndependentChunksKeepSmallBitrate(t *testing.T) {
	m := testManager(t)
	path := filepath.Join(t.TempDir(), "busy.mp4")
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30", "-t", "2", "-c:v", "libx264", "-preset", "ultrafast", "-y", path).CombinedOutput()
	if err != nil {
		t.Fatalf("fixture: %v: %s", err, out)
	}
	v, err := m.Open(context.Background(), path, 1)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(m.conf, "small")
	f, err := m.File(context.Background(), v, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	// Allow transport overhead and small bitrate bursts, but reject large startup bursts.
	kbps := float64(info.Size()) * 8 / SegmentDuration / 1000
	if kbps > float64(p.MaxRate)*1.3 {
		t.Fatalf("short chunk exceeded its bandwidth budget: %.0f kbps", kbps)
	}
}
