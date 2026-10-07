package transcode

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Capability struct {
	Encoder     string `json:"encoder"`
	Device      string `json:"device,omitempty"`
	RateControl string `json:"rateControl,omitempty"`
	Fallback    string `json:"fallback,omitempty"`
}

type encoder struct {
	once sync.Once
	mu   sync.Mutex
	cap  Capability
}

// detect tests VA-API support with a real VBR encode.
func (e *encoder) detect(c Conf) Capability {
	e.once.Do(func() {
		s := Settings(c)
		logger := c.Logger
		if logger == nil {
			logger = slog.Default()
		}
		cap := Capability{Encoder: "libx264"}
		if s.Encoder != "cpu" {
			devices := []string{s.Device}
			if s.Device == "" {
				devices, _ = filepath.Glob("/dev/dri/renderD*")
			}
			cap.Fallback = "No accessible VA-API render device; using CPU encoding."
			for _, device := range devices {
				logger.Info("probing video GPU", "device", device, "encoder", "h264_vaapi", "rate_control", "VBR")
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-vaapi_device", device,
					"-f", "lavfi", "-i", "color=size=128x128:rate=30", "-t", "0.2", "-vf", "format=nv12,hwupload",
					"-c:v", "h264_vaapi", "-rc_mode", "VBR", "-b:v", "300k", "-maxrate", "600k", "-bufsize", "1200k", "-f", "null", "-"}
				_, err := runFFmpeg(ctx, args)
				cancel()
				if err == nil {
					cap = Capability{Encoder: "h264_vaapi", Device: device, RateControl: "VBR"}
					break
				}
				cap.Fallback = "VA-API VBR encode check failed; using CPU encoding."
				logger.Warn("video GPU probe failed", "device", device, "error", err)
			}
		}
		if cap.Fallback != "" {
			logger.Warn("video encoder selected", "requested_encoder", s.Encoder, "encoder", cap.Encoder, "fallback", cap.Fallback)
		} else {
			logger.Info("video encoder selected", "requested_encoder", s.Encoder, "encoder", cap.Encoder, "device", cap.Device, "rate_control", cap.RateControl)
		}
		e.mu.Lock()
		e.cap = cap
		e.mu.Unlock()
	})
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cap
}

// fallback switches future encoding to the CPU and records the reason.
func (e *encoder) fallback(reason string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cap = Capability{Encoder: "libx264", Fallback: reason}
}

// Limit encoder error output to avoid excessive memory use.
type tailWriter struct{ data []byte }

// Write keeps only the last 8 KiB of encoder output.
func (w *tailWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.data = append(w.data, p...)
	if len(w.data) > 8192 {
		w.data = append([]byte(nil), w.data[len(w.data)-8192:]...)
	}
	return n, nil
}

// runFFmpeg runs FFmpeg with cancellation and bounded error output.
func runFFmpeg(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr tailWriter
	cmd.Stderr = &stderr
	cmd.WaitDelay = 3 * time.Second
	err := cmd.Run()
	if ctx.Err() != nil {
		return string(stderr.data), ctx.Err()
	}
	if err != nil {
		return string(stderr.data), fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.data))
	}
	return string(stderr.data), nil
}

// encodeArgs builds FFmpeg arguments for a segment or preview.
func encodeArgs(s Source, p Profile, start, duration float64, output, preset string, cr Capability, hardwareDecode bool, preview bool) []string {
	seekStart, encodeDuration := start, duration
	var audioPrerollSamples int64
	if s.Audio && !preview {
		// Warm up AAC before the segment. Align seeks to 1024-sample boundaries
		// to avoid overlapping audio frames.
		seekSample := max(int64(0), (int64(math.Floor(start*48000/1024))-12)*1024)
		audioPrerollSamples = int64(math.Round(start*48000)) - seekSample
		seekStart = float64(seekSample) / 48000
		// Encode past the join, then drop extra packets to avoid silence padding.
		// The overlap adds less than 0.4 seconds of work.
		encodeDuration += float64(audioPrerollSamples)/48000 + 0.1
	}
	a := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-threads", "2", "-filter_threads", "1"}
	if cr.Encoder == "h264_vaapi" {
		a = append(a, "-vaapi_device", cr.Device)
		if hardwareDecode {
			a = append(a, "-hwaccel", "vaapi", "-hwaccel_device", cr.Device, "-hwaccel_output_format", "vaapi")
		}
	}
	a = append(a, "-ss", fmt.Sprintf("%.9f", seekStart), "-i", s.Path, "-t", fmt.Sprintf("%.9f", encodeDuration), "-map", fmt.Sprintf("0:%d", s.VideoIndex))
	if s.Audio && !preview {
		a = append(a, "-map", fmt.Sprintf("0:%d", s.AudioIndex))
	} else {
		a = append(a, "-an")
	}
	w, h := Dimensions(s.Width, s.Height, p.LongEdge)
	fps := p.FrameRate(s)
	scaling := ""
	if p.ID == "high" {
		scaling = ":flags=lanczos"
	}
	filter := fmt.Sprintf("scale=%d:%d%s,setsar=1,fps=%.6f,format=yuv420p", w, h, scaling, fps)
	if s.HDR {
		filter = "zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=hable:desat=0,zscale=t=bt709:m=bt709:r=tv," + filter
	}
	if cr.Encoder == "h264_vaapi" {
		if hardwareDecode {
			// Convert color without copying frames off the GPU.
			filter = fmt.Sprintf("scale_vaapi=w=%d:h=%d:format=nv12:out_color_matrix=bt709:out_color_primaries=bt709:out_color_transfer=bt709:out_range=tv,setsar=1,fps=%.6f", w, h, fps)
		} else {
			filter += ",format=nv12,hwupload"
		}
	}
	if s.Audio && !preview {
		// Trim audio warm-up time from the video without shifting its timestamps.
		// These filters also work with GPU frames.
		preroll := float64(audioPrerollSamples) / 48000
		filter = fmt.Sprintf("trim=start=%.9f:duration=%.9f,setpts=PTS-%.9f/TB,", preroll, duration, preroll) + filter
	}
	a = append(a, "-vf", filter, "-c:v", cr.Encoder, "-threads", "2", "-g", strconv.Itoa(int(math.Ceil(fps*SegmentDuration))), "-bf", "0", "-profile:v", "high")
	if cr.Encoder == "libx264" {
		a = append(a, "-preset", preset, "-crf", strconv.Itoa(p.CRF), "-sc_threshold", "0")
	} else {
		a = append(a, "-rc_mode", "VBR", "-b:v", fmt.Sprintf("%dk", p.MaxRate*3/4))
	}
	// Keep startup bursts small, but give High more buffer space
	// to avoid over-compressing the first frames.
	initialBuffer := p.MaxRate * 200 // 10% of the two-second buffer, in bits
	if p.ID == "high" {
		initialBuffer = p.MaxRate * 1000 // 50%
	}
	a = append(a, "-maxrate", fmt.Sprintf("%dk", p.MaxRate), "-bufsize", fmt.Sprintf("%dk", p.MaxRate*2), "-rc_init_occupancy", strconv.Itoa(initialBuffer))
	if hardwareDecode {
		// Set H.264 color metadata after GPU conversion. Using -colorspace here
		// causes an unsupported software conversion in FFmpeg 7.
		a = append(a, "-bsf:v", "h264_metadata=colour_primaries=1:transfer_characteristics=1:matrix_coefficients=1:video_full_range_flag=0")
	} else {
		a = append(a, "-color_primaries", "bt709", "-color_trc", "bt709", "-colorspace", "bt709")
	}
	if s.Audio && !preview {
		a = append(a, "-c:a", "aac", "-b:a", fmt.Sprintf("%dk", p.AudioBitrate), "-ac", "2", "-ar", "48000",
			"-af", fmt.Sprintf("aresample=48000:async=1:first_pts=0,apad,asetpts=PTS-%d", audioPrerollSamples),
			"-bsf:a", fmt.Sprintf("noise=amount=0:drop='lt(pts,0)+gte(pts,%d)'", int64(math.Ceil(duration*48000))))
	}
	if preview {
		a = append(a, "-movflags", "+faststart", "-f", "mp4", output)
	} else {
		// Keep segment timestamps continuous to avoid audio gaps and frozen frames.
		// Use the same one-second offset with or without audio, on CPU and GPU.
		// Do not shift each segment to its first packet.
		a = append(a,
			"-output_ts_offset", fmt.Sprintf("%.6f", start+1),
			"-avoid_negative_ts", "disabled", "-mpegts_copyts", "1",
			"-muxdelay", "0", "-muxpreload", "0", "-f", "mpegts", output)
	}
	return a
}

// encode renders a segment and retries with CPU encoding if needed.
func (m *Manager) encode(ctx context.Context, v *Video, p Profile, index int, out string) error {
	started := time.Now()
	duration := min(SegmentDuration, v.Source.Duration-float64(index)*SegmentDuration)
	preview := p.ID == "preview"
	if preview {
		duration = min(4, v.Source.Duration)
	}
	cap := m.encoder.detect(m.conf)
	// Tone-map HDR in software. Retry GPU failures with software decoding, then CPU encoding.
	hwDecode := cap.Encoder == "h264_vaapi" && !v.Source.HDR
	w, h := Dimensions(v.Source.Width, v.Source.Height, p.LongEdge)
	m.conf.Logger.Info("video encode started", "media", v.Hash, "profile", p.ID, "segment", index, "resolution", fmt.Sprintf("%dx%d", w, h), "fps", p.FrameRate(v.Source), "encoder", cap.Encoder, "device", cap.Device, "hardware_decode", hwDecode, "hdr_tonemap", v.Source.HDR, "max_video_kbps", p.MaxRate, "cpu_crf", p.CRF)
	args := encodeArgs(v.Source, p, float64(index)*SegmentDuration, duration, out, Settings(m.conf).Preset, cap, hwDecode, preview)
	_, err := runFFmpeg(ctx, args)
	if err != nil && ctx.Err() == nil && hwDecode {
		m.conf.Logger.Warn("video hardware decode failed; retrying software decode with GPU encoding", "media", v.Hash, "profile", p.ID, "segment", index, "device", cap.Device, "error", err)
		hwDecode = false
		args = encodeArgs(v.Source, p, float64(index)*SegmentDuration, duration, out, Settings(m.conf).Preset, cap, false, preview)
		_, err = runFFmpeg(ctx, args)
	}
	if err != nil && ctx.Err() == nil && cap.Encoder != "libx264" {
		m.conf.Logger.Warn("hardware transcode failed; retrying on CPU", "media", v.Hash, "profile", p.ID, "segment", index, "device", cap.Device, "error", err)
		m.encoder.fallback("Hardware encoding failed for this input; using CPU encoding.")
		cap = Capability{Encoder: "libx264"}
		hwDecode = false
		args = encodeArgs(v.Source, p, float64(index)*SegmentDuration, duration, out, Settings(m.conf).Preset, cap, false, preview)
		_, err = runFFmpeg(ctx, args)
	}
	if err == nil {
		seconds := time.Since(started).Seconds()
		m.mu.Lock()
		m.lastEncodeSeconds = seconds
		m.lastEncodeSpeed = duration / seconds
		m.mu.Unlock()
		var size int64
		if info, err := os.Stat(out); err == nil {
			size = info.Size()
		}
		m.conf.Logger.Info("video encode completed", "media", v.Hash, "profile", p.ID, "segment", index, "resolution", fmt.Sprintf("%dx%d", w, h), "encoder", cap.Encoder, "device", cap.Device, "hardware_decode", hwDecode, "seconds", seconds, "speed", duration/seconds, "bytes", size)
	}
	return err
}

// validOutput checks that a cached file is nonempty.
func validOutput(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

// safeError returns an error message or an empty string.
func safeError(err error) string {
	if err == nil {
		return ""
	}
	if strings.Contains(err.Error(), "context canceled") {
		return "Canceled"
	}
	return "Encoding failed; inspect server logs for details."
}
