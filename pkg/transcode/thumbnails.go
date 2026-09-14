package transcode

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"
)

const thumbnailDirectory = "thumbnails-v1"
const thumbnailColumns = 5
const thumbnailsPerSheet = 25

// Thumbnail describes a frame's time and pixel position in a sprite sheet.
type Thumbnail struct {
	URL       string  `json:"url"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Coords    struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"coords"`
}

// thumbnailLayout chooses the thumbnail interval, count, and frame size.
func thumbnailLayout(v *Video) (interval float64, count, width, height int) {
	// Sample every five seconds, less often for long recordings.
	interval = max(5, math.Ceil(v.Source.Duration/2000/5)*5)
	count = int(math.Ceil(v.Source.Duration / interval))
	width, height = Dimensions(v.Source.Width, v.Source.Height, 320)
	return
}

// Thumbnails lists frames from the original video without encoding them.
func Thumbnails(v *Video) []Thumbnail {
	interval, count, width, height := thumbnailLayout(v)
	result := make([]Thumbnail, count)
	for i := range result {
		result[i] = Thumbnail{
			URL:       fmt.Sprintf("/api/transcode/%d/thumbnails/%06d.jpg?v=%s&revision=1", v.Hash, i/thumbnailsPerSheet, v.Version),
			StartTime: float64(i) * interval,
			EndTime:   min(float64(i+1)*interval, v.Source.Duration),
			Width:     width,
			Height:    height,
		}
		result[i].Coords.X = (i % thumbnailsPerSheet % thumbnailColumns) * width
		result[i].Coords.Y = (i % thumbnailsPerSheet / thumbnailColumns) * height
	}
	return result
}

// ThumbnailSheet uses the shared workers to cache a sprite sheet, not a playback stream.
func (m *Manager) ThumbnailSheet(ctx context.Context, v *Video, sheet, priority int) (*CachedFile, error) {
	_, count, _, _ := thumbnailLayout(v)
	if sheet < 0 || sheet >= (count+thumbnailsPerSheet-1)/thumbnailsPerSheet {
		return nil, os.ErrNotExist
	}
	release, err := fileLock(ctx, v.Dir+".lock", false, true)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			release()
		}
	}()
	dir := filepath.Join(v.Dir, thumbnailDirectory)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, fmt.Sprintf("%06d.jpg", sheet))
	if !validOutput(path) {
		err = m.do(ctx, path, priority, func(ctx context.Context) error {
			assetUnlock, err := fileLock(ctx, v.Dir+".lock", false, true)
			if err != nil {
				return err
			}
			defer assetUnlock()
			unlock, err := fileLock(ctx, path+".lock", true, true)
			if err != nil {
				return err
			}
			defer unlock()
			if validOutput(path) {
				return nil
			}
			started := time.Now()
			m.conf.Logger.Info("video seek thumbnails started", "media", v.Hash, "sheet", sheet)
			if err := m.encodeThumbnailSheet(ctx, v, sheet, path); err != nil {
				return err
			}
			m.conf.Logger.Info("video seek thumbnails complete", "media", v.Hash, "sheet", sheet, "seconds", time.Since(started).Seconds())
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	v.touch()
	ok = true
	return &CachedFile{File: f, release: release}, nil
}

// encodeThumbnailSheet samples the source and saves a JPEG sprite sheet.
func (m *Manager) encodeThumbnailSheet(ctx context.Context, v *Video, sheet int, path string) error {
	interval, total, width, height := thumbnailLayout(v)
	first := sheet * thumbnailsPerSheet
	count := min(thumbnailsPerSheet, total-first)
	sprite := image.NewRGBA(image.Rect(0, 0, min(count, thumbnailColumns)*width, ((count+thumbnailColumns-1)/thumbnailColumns)*height))
	frame, err := os.CreateTemp(filepath.Dir(path), ".encoding-*")
	if err != nil {
		return err
	}
	framePath := frame.Name()
	_ = frame.Close()
	defer func() { _ = os.Remove(framePath) }()
	filter := fmt.Sprintf("scale=%d:%d:flags=lanczos,setsar=1", width, height)
	if v.Source.HDR {
		filter = "zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=hable:desat=0,zscale=t=bt709:m=bt709:r=tv," + filter
	}
	for i := 0; i < count; i++ {
		// Seek to each sample instead of decoding the whole video.
		start := min(float64(first+i)*interval, max(0, v.Source.Duration-0.05))
		if err := os.Truncate(framePath, 0); err != nil {
			return err
		}
		args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-threads", "2", "-filter_threads", "1", "-ss", fmt.Sprintf("%.6f", start), "-i", v.Source.Path, "-map", fmt.Sprintf("0:%d", v.Source.VideoIndex), "-an", "-frames:v", "1", "-vf", filter, "-c:v", "png", "-threads", "1", "-f", "image2", "-update", "1", framePath}
		if _, err := runFFmpeg(ctx, args); err != nil {
			return err
		}
		f, err := os.Open(framePath)
		if err != nil {
			return err
		}
		img, err := png.Decode(f)
		_ = f.Close()
		if err != nil {
			return fmt.Errorf("decode seek thumbnail: %w", err)
		}
		x, y := (i%thumbnailColumns)*width, (i/thumbnailColumns)*height
		draw.Draw(sprite, image.Rect(x, y, x+width, y+height), img, img.Bounds().Min, draw.Src)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".encoding-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() { _ = os.Remove(tmp) }()
	writeErr := jpeg.Encode(f, sprite, &jpeg.Options{Quality: 80})
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	version, err := SourceVersion(v.Source.Path)
	if err != nil {
		return err
	}
	if version != v.Source.Version {
		return fmt.Errorf("source changed during thumbnail generation")
	}
	return os.Rename(tmp, path)
}
