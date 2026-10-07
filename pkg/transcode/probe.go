package transcode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	Path        string  `json:"-"`
	Version     string  `json:"version"`
	Duration    float64 `json:"duration"`
	FrameRate   float64 `json:"frameRate"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Audio       bool    `json:"audio"`
	HDR         bool    `json:"hdr"`
	VideoCodec  string  `json:"videoCodec"`
	AudioCodec  string  `json:"audioCodec"`
	PixelFormat string  `json:"pixelFormat"`
	Bitrate     int64   `json:"bitrate"`
	Format      string  `json:"format"`
	VideoIndex  int     `json:"-"`
	AudioIndex  int     `json:"-"`
}

// digest hashes a value for use in cache keys.
func digest(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

// SourceVersion identifies a source by its path, size, and modification time.
func SourceVersion(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("video source is not a regular file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return digest([]any{abs, info.Size(), info.ModTime().UnixNano(), cacheVersion}), nil
}

// Probe reads video metadata and checks that the source is usable.
func Probe(ctx context.Context, path string) (Source, error) {
	version, err := SourceVersion(path)
	if err != nil {
		return Source{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	b, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_format", "-show_streams", "-of", "json", path).Output()
	if err != nil {
		return Source{}, fmt.Errorf("probe video: %w", err)
	}
	var data struct {
		Streams []struct {
			Index       int    `json:"index"`
			Type        string `json:"codec_type"`
			Codec       string `json:"codec_name"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			SAR         string `json:"sample_aspect_ratio"`
			PixelFormat string `json:"pix_fmt"`
			Transfer    string `json:"color_transfer"`
			Duration    string `json:"duration"`
			FrameRate   string `json:"avg_frame_rate"`
			RealRate    string `json:"r_frame_rate"`
			Disposition struct {
				AttachedPic int `json:"attached_pic"`
			} `json:"disposition"`
			SideData []struct {
				Rotation float64 `json:"rotation"`
			} `json:"side_data_list"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Bitrate  string `json:"bit_rate"`
			Name     string `json:"format_name"`
		} `json:"format"`
	}
	if err := json.Unmarshal(b, &data); err != nil {
		return Source{}, err
	}
	s := Source{Path: path, Version: version, VideoIndex: -1, AudioIndex: -1, Format: data.Format.Name}
	s.Duration, _ = strconv.ParseFloat(data.Format.Duration, 64)
	s.Bitrate, _ = strconv.ParseInt(data.Format.Bitrate, 10, 64)
	for _, stream := range data.Streams {
		if stream.Type == "audio" && !s.Audio {
			s.Audio = true
			s.AudioCodec = stream.Codec
			s.AudioIndex = stream.Index
		}
		if stream.Type != "video" || stream.Disposition.AttachedPic != 0 || s.VideoIndex >= 0 {
			continue
		}
		s.VideoIndex = stream.Index
		s.VideoCodec, s.PixelFormat = stream.Codec, stream.PixelFormat
		s.Width, s.Height = stream.Width, stream.Height
		s.FrameRate = parseFrameRate(stream.FrameRate)
		if s.FrameRate == 0 {
			s.FrameRate = parseFrameRate(stream.RealRate)
		}
		var num, den float64
		if _, err := fmt.Sscanf(stream.SAR, "%f:%f", &num, &den); err == nil && num > 0 && den > 0 {
			s.Width = int(math.Round(float64(s.Width) * num / den))
		}
		for _, side := range stream.SideData {
			if int(math.Round(math.Abs(side.Rotation)))%180 == 90 {
				s.Width, s.Height = s.Height, s.Width
			}
		}
		s.HDR = stream.Transfer == "smpte2084" || stream.Transfer == "arib-std-b67"
		// Prefer the video track's duration to avoid an audio-only tail.
		if d, err := strconv.ParseFloat(stream.Duration, 64); err == nil && d > 0 {
			s.Duration = d
		}
	}
	if s.VideoIndex < 0 || s.Width < 2 || s.Height < 2 || math.IsNaN(s.Duration) || math.IsInf(s.Duration, 0) || s.Duration <= 0 || s.Duration > 7*24*3600 {
		return Source{}, fmt.Errorf("unsupported video dimensions or duration")
	}
	return s, nil
}

// parseFrameRate parses a frame rate and rejects invalid values.
func parseFrameRate(value string) float64 {
	var num, den float64
	if _, err := fmt.Sscanf(value, "%f/%f", &num, &den); err != nil || den <= 0 {
		return 0
	}
	rate := num / den
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 || rate > 240 {
		return 0
	}
	return rate
}

// DirectEligible checks source formats and quality limits.
// The browser checks playback support separately.
func (s Source) DirectEligible(p Profile) bool {
	return s.VideoCodec == "h264" && (s.PixelFormat == "yuv420p" || s.PixelFormat == "yuvj420p") && !s.HDR &&
		(!s.Audio || s.AudioCodec == "aac") && s.Bitrate > 0 && s.Bitrate <= int64(p.MaxRate+p.AudioBitrate)*1000 && max(s.Width, s.Height) <= p.LongEdge
}

// IsMP4 reports whether the source uses an MP4 or MOV container.
func (s Source) IsMP4() bool {
	return strings.Contains(s.Format, "mp4") || strings.Contains(s.Format, "mov")
}
