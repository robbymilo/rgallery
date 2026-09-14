package transcode

import (
	"fmt"
	"math"

	"github.com/robbymilo/rgallery/pkg/types"
)

type Conf = types.Conf

const SegmentDuration = 2.0
const cacheVersion = "v2"
const encoderVersion = 3

type Profile struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	LongEdge     int    `json:"longEdge"`
	MaxRate      int    `json:"maxRate"` // kilobits per second, excluding audio
	AudioBitrate int    `json:"audioBitrate"`
	CRF          int    `json:"-"`
}

// Settings fills in default transcoding settings.
func Settings(c Conf) types.TranscodeConfig {
	s := c.Transcode
	if s.Profile == "" {
		s.Profile = "small"
		s.CRF = -1
	}
	if s.Mode == "" {
		s.Mode = "ondemand"
	}
	if s.Encoder == "" {
		s.Encoder = "auto"
	}
	if s.Preset == "" {
		s.Preset = "veryfast"
	}
	if s.Workers == 0 {
		s.Workers = 2
	}
	if s.CacheMB == 0 {
		s.CacheMB = 10240
	}
	return s
}

// Validate checks that transcoding settings are supported.
func Validate(c Conf) error {
	s := Settings(c)
	if s.Profile != "saver" && s.Profile != "small" && s.Profile != "high" {
		return fmt.Errorf("transcode-quality must be saver, small, or high")
	}
	if s.Mode != "ondemand" && s.Mode != "pregenerate" && s.Mode != "hybrid" {
		return fmt.Errorf("transcode-mode must be ondemand, pregenerate, or hybrid")
	}
	if s.Encoder != "auto" && s.Encoder != "cpu" && s.Encoder != "vaapi" {
		return fmt.Errorf("transcode-encoder must be auto, cpu, or vaapi")
	}
	if c.TranscodeResolution < 0 || c.TranscodeResolution > 7680 || c.TranscodeResolution == 1 {
		return fmt.Errorf("transcode-resolution must be between 2 and 7680")
	}
	if s.CRF < -1 || s.CRF > 51 {
		return fmt.Errorf("transcode-crf must be -1 (profile default) or 0–51")
	}
	if s.MaxRate < 0 || s.MaxRate > 100000 {
		return fmt.Errorf("transcode-maxrate must be 0 (profile default) or 1–100000 kbps")
	}
	if s.AudioBitrate != 0 && (s.AudioBitrate < 32 || s.AudioBitrate > 320) {
		return fmt.Errorf("transcode-audio-bitrate must be 0 (profile default) or 32–320 kbps")
	}
	if s.Workers < 1 || s.Workers > 16 {
		return fmt.Errorf("transcode-workers must be between 1 and 16")
	}
	if s.CacheMB < 64 {
		return fmt.Errorf("transcode-cache-mb must be at least 64")
	}
	switch s.Preset {
	case "ultrafast", "superfast", "veryfast", "faster", "fast", "medium", "slow", "slower", "veryslow":
	default:
		return fmt.Errorf("invalid transcode-preset")
	}
	return nil
}

// Profiles returns playback qualities with the configured limits.
func Profiles(c Conf) []Profile {
	s := Settings(c)
	limit := c.TranscodeResolution
	if limit == 0 {
		limit = 1280
	}
	// Give higher resolutions more bitrate to preserve detail.
	highRate := min(100000, max(30000, int(30000*math.Pow(float64(limit)/1920, 2))))
	profiles := []Profile{{"saver", "Data saver", 854, 600, 64, 30}, {"small", "Small", 1280, 1000, 64, 28}, {"high", "High", limit, highRate, 256, 14}}
	for i := range profiles {
		p := &profiles[i]
		p.LongEdge = min(p.LongEdge, limit)
		if s.MaxRate > 0 {
			p.MaxRate = min(p.MaxRate, s.MaxRate)
		}
		if s.AudioBitrate > 0 {
			p.AudioBitrate = s.AudioBitrate
		}
		if s.CRF >= 0 && p.ID == s.Profile {
			p.CRF = s.CRF
		}
	}
	return profiles
}

// FrameRate keeps the source rate for High and uses 30 fps for other qualities.
func (p Profile) FrameRate(s Source) float64 {
	if p.ID == "high" && s.FrameRate > 0 {
		return s.FrameRate
	}
	return 30
}

// FindProfile looks up a playback or preview quality.
func FindProfile(c Conf, id string) (Profile, error) {
	if id == "" {
		id = Settings(c).Profile
	}
	if id == "preview" {
		p := Profile{"preview", "Preview", 1280, 2000, 0, 22}
		if c.TranscodeResolution > 0 {
			p.LongEdge = min(p.LongEdge, c.TranscodeResolution)
		}
		if rate := Settings(c).MaxRate; rate > 0 {
			p.MaxRate = min(p.MaxRate, rate)
		}
		return p, nil
	}
	for _, p := range Profiles(c) {
		if p.ID == id {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("unknown video quality %q", id)
}

// Dimensions scales to even pixel dimensions without enlarging the source.
func Dimensions(width, height, edge int) (int, int) {
	scale := math.Min(1, float64(edge)/float64(max(width, height)))
	return max(2, int(float64(width)*scale)/2*2), max(2, int(float64(height)*scale)/2*2)
}
