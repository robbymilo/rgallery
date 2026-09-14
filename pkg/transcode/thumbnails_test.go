package transcode

import (
	"context"
	"errors"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestThumbnailTimelineAndSheetBoundaries checks thumbnail timing, coordinates, and size limits.
func TestThumbnailTimelineAndSheetBoundaries(t *testing.T) {
	v := &Video{Hash: 42, Version: "test", Source: Source{Width: 1080, Height: 1920, Duration: 130.2}}
	thumbs := Thumbnails(v)
	if len(thumbs) != 27 || thumbs[0].StartTime != 0 || thumbs[26].EndTime != v.Source.Duration {
		t.Fatalf("incomplete thumbnail timeline: %+v", thumbs)
	}
	for i, thumb := range thumbs {
		if thumb.Width != 180 || thumb.Height != 320 || thumb.StartTime != float64(i)*5 {
			t.Fatalf("incorrect portrait thumbnail %d: %+v", i, thumb)
		}
		if i > 0 && thumbs[i-1].EndTime != thumb.StartTime {
			t.Fatal("gap between thumbnail intervals")
		}
	}
	if thumbs[24].Coords.X != 720 || thumbs[24].Coords.Y != 1280 || !strings.Contains(thumbs[24].URL, "/000000.jpg?") || !strings.Contains(thumbs[25].URL, "/000001.jpg?") || thumbs[25].Coords.X != 0 || thumbs[25].Coords.Y != 0 {
		t.Fatal("incorrect sprite sheet boundary")
	}
	v.Source.Duration = 24 * 60 * 60
	if len(Thumbnails(v)) > 2000 {
		t.Fatal("long recordings produce unbounded thumbnail metadata")
	}
	v.Source.Duration = 0.1
	if thumbs := Thumbnails(v); len(thumbs) != 1 || thumbs[0].EndTime != 0.1 {
		t.Fatal("short video has no thumbnail")
	}
}

// TestThumbnailSheetSamplesOriginalAndReusesCache checks sampled scenes and shared cache reuse.
func TestThumbnailSheetSamplesOriginalAndReusesCache(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	m := testManager(t)
	path := filepath.Join(t.TempDir(), "colors.mp4")
	args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "color=c=red:s=320x180:r=10:d=5", "-f", "lavfi", "-i", "color=c=blue:s=320x180:r=10:d=5", "-f", "lavfi", "-i", "color=c=lime:s=320x180:r=10:d=0.2", "-filter_complex", "[0:v][1:v][2:v]concat=n=3:v=1:a=0", "-c:v", "libx264", "-preset", "ultrafast", "-y", path}
	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("color fixture: %v: %s", err, output)
	}
	v, err := m.Open(context.Background(), path, 42)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	// Concurrent requests should share the same completed sheet.
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := m.ThumbnailSheet(context.Background(), v, 0, 1)
			if err != nil {
				t.Error(err)
				return
			}
			_ = f.Close()
		}()
	}
	wg.Wait()
	f, err := m.ThumbnailSheet(context.Background(), v, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 960 || img.Bounds().Dy() != 180 {
		t.Fatalf("incorrect sprite dimensions: %v", img.Bounds())
	}
	for i, dominant := range []int{0, 2, 1} {
		r, g, b, _ := img.At(i*320+160, 90).RGBA()
		channels := []uint32{r, g, b}
		if channels[dominant] < 50000 || channels[(dominant+1)%3] > 10000 || channels[(dominant+2)%3] > 10000 {
			t.Fatalf("thumbnail at %ds has the wrong scene: %v", i*5, channels)
		}
	}
	f, err = m.ThumbnailSheet(context.Background(), v, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	cached, _ := f.Stat()
	_ = f.Close()
	if !info.ModTime().Equal(cached.ModTime()) {
		t.Fatal("cached sheet was regenerated")
	}
	for _, index := range []int{-1, 1, 999999} {
		if _, err := m.ThumbnailSheet(context.Background(), v, index, 1); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid sheet %d: %v", index, err)
		}
	}
	for _, p := range Profiles(m.conf) {
		if _, err := os.Stat(filepath.Join(v.Dir, p.ID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("thumbnail request started playback rendering for %s", p.ID)
		}
	}
}

// TestLateThumbnailSheetDoesNotGenerateEarlierSheets checks that a late seek only builds its requested sheet.
func TestLateThumbnailSheetDoesNotGenerateEarlierSheets(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	m := testManager(t)
	path := filepath.Join(t.TempDir(), "long-portrait.mp4")
	output, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=180x320:rate=1", "-t", "131", "-c:v", "libx264", "-preset", "ultrafast", "-y", path).CombinedOutput()
	if err != nil {
		t.Fatalf("portrait fixture: %v: %s", err, output)
	}
	v, err := m.Open(context.Background(), path, 42)
	if err != nil {
		t.Fatal(err)
	}
	f, err := m.ThumbnailSheet(context.Background(), v, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	_ = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 360 || img.Bounds().Dy() != 320 {
		t.Fatalf("incorrect last portrait sheet: %v", img.Bounds())
	}
	if validOutput(filepath.Join(v.Dir, thumbnailDirectory, "000000.jpg")) {
		t.Fatal("late seek generated the earlier sheet")
	}
	if thumbs := Thumbnails(v); len(thumbs) != 27 || thumbs[26].Coords.X != 180 || thumbs[26].Coords.Y != 0 || thumbs[26].EndTime != 131 {
		t.Fatal("last-sheet coordinates do not match the image")
	}
}
