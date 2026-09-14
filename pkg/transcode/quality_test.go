package transcode

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/robbymilo/rgallery/pkg/types"
)

// TestHighResolutionAndFrameRate checks High quality resolution and frame rate settings.
func TestHighResolutionAndFrameRate(t *testing.T) {
	for _, edge := range []int{1280, 1920, 3840, 7680} {
		c := Conf{TranscodeResolution: edge}
		p, err := FindProfile(c, "high")
		if err != nil || p.LongEdge != edge {
			t.Fatalf("High ignored resolution %d: %+v, %v", edge, p, err)
		}
		w, h := Dimensions(3840, 2160, p.LongEdge)
		if edge >= 3840 && (w != 3840 || h != 2160) {
			t.Fatalf("High downscaled a permitted 4K source: %dx%d", w, h)
		}
		c.Transcode.MaxRate = 5000
		capped, _ := FindProfile(c, "high")
		if capped.MaxRate > 5000 {
			t.Fatal("High bypassed the explicit bandwidth cap")
		}
	}
	for _, value := range []string{"24/1", "30000/1001", "60/1", "120/1"} {
		rate := parseFrameRate(value)
		high, _ := FindProfile(Conf{}, "high")
		if rate == 0 || high.FrameRate(Source{FrameRate: rate}) != rate {
			t.Fatalf("High lost source frame rate %s", value)
		}
		for _, id := range []string{"small", "saver", "preview"} {
			p, _ := FindProfile(Conf{}, id)
			if p.FrameRate(Source{FrameRate: rate}) != 30 {
				t.Fatalf("%s no longer limits frame rate", id)
			}
		}
	}
	for _, value := range []string{"0/0", "NaN/1", "1/0", "-1/1", "60000/0", "90000/1", "broken"} {
		if parseFrameRate(value) != 0 {
			t.Fatalf("accepted invalid frame rate %q", value)
		}
	}
}

// TestHighQualityPreservesDetailAndMotion checks detail and motion in High quality output.
func TestHighQualityPreservesDetailAndMotion(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	c := Conf{Cache: t.TempDir(), TranscodeResolution: 1920, Transcode: types.TranscodeConfig{Profile: "high", CRF: -1, Encoder: "cpu"}}
	if device := os.Getenv("RGALLERY_TEST_GPU"); device != "" {
		c.Transcode.Encoder, c.Transcode.Device = "vaapi", device
	}
	m := NewManager(c)
	defer m.Close()
	path := filepath.Join(t.TempDir(), "motion.mp4")
	args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=1920x1080:rate=60,noise=alls=6:allf=t+u", "-t", "2.2", "-c:v", "libx264", "-preset", "ultrafast", "-crf", "10", "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709", "-color_range", "tv", "-threads", "2", "-y", path}
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("motion fixture: %v: %s", err, output)
	}
	v, err := m.Open(context.Background(), path, 123)
	if err != nil {
		t.Fatal(err)
	}
	p, _ := FindProfile(c, "high")
	f, err := m.File(context.Background(), v, p, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	output, err := Probe(context.Background(), f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if output.Width != 1920 || output.Height != 1080 || math.Abs(output.FrameRate-60) > .01 || math.Abs(output.Duration-2) > .05 {
		t.Fatalf("High lost resolution, motion or segment timing: %+v", output)
	}
	if c.Transcode.Encoder == "vaapi" && m.Diagnostics().Encoder != "h264_vaapi" {
		t.Fatal("High hardware encoding fell back to CPU")
	}
	score := qualitySSIM(t, path, f.Name())
	// Compare old quality settings at the same resolution and frame rate.
	previous := p
	previous.MaxRate, previous.CRF = 8000, 18
	oldPath := filepath.Join(t.TempDir(), "old-budget.ts")
	if err := m.encode(context.Background(), v, previous, 0, oldPath); err != nil {
		t.Fatal(err)
	}
	oldScore := qualitySSIM(t, path, oldPath)
	t.Logf("1080p60 source comparison: old budget SSIM %.6f, High %.6f; encoder %s", oldScore, score, m.Diagnostics().Encoder)
	if score < .98 {
		t.Fatalf("High lost too much detail compared with source: SSIM %.6f", score)
	}
	if score <= oldScore {
		t.Fatalf("High did not improve detail: old %.6f, new %.6f", oldScore, score)
	}
}

// qualitySSIM measures how closely the encoded video matches its source.
func qualitySSIM(t *testing.T, source, encoded string) float64 {
	t.Helper()
	args := []string{"-hide_banner", "-i", source, "-i", encoded, "-filter_complex", "[0:v]trim=duration=2,setpts=PTS-STARTPTS[ref];[1:v]setpts=PTS-STARTPTS[out];[out][ref]ssim", "-an", "-f", "null", "-"}
	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("compare source: %v: %s", err, out)
	}
	match := regexp.MustCompile(`All:([0-9.]+)`).FindStringSubmatch(string(out))
	if len(match) != 2 {
		t.Fatalf("SSIM score missing: %s", out)
	}
	score, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		t.Fatal(err)
	}
	return score
}
