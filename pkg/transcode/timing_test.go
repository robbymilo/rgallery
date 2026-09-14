package transcode

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/robbymilo/rgallery/pkg/types"
)

// TestSegmentPresentationClock checks that timestamps stay continuous across segments.
func TestSegmentPresentationClock(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	for _, audioRate := range []int{0, 48000, 44100} {
		t.Run(fmt.Sprintf("audio=%d", audioRate), func(t *testing.T) {
			audio := audioRate > 0
			path := filepath.Join(t.TempDir(), "motion.mp4")
			args := []string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=60000/1001"}
			if audio {
				args = append(args, "-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=440:sample_rate=%d", audioRate))
			}
			args = append(args, "-t", "8.2", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-y", path)
			if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
				t.Fatalf("fixture: %v: %s", err, out)
			}
			c := Conf{Cache: t.TempDir(), Transcode: types.TranscodeConfig{Encoder: "cpu"}}
			if device := os.Getenv("RGALLERY_TEST_GPU"); device != "" {
				c.Transcode.Encoder, c.Transcode.Device = "vaapi", device
			}
			m := NewManager(c)
			defer m.Close()
			v, err := m.Open(context.Background(), path, 123)
			if err != nil {
				t.Fatal(err)
			}
			for _, id := range []string{"saver", "high"} {
				t.Run(id, func(t *testing.T) {
					p, _ := FindProfile(c, id)
					paths := make([]string, 5)
					// Seeking can generate later intervals before earlier ones.
					for _, index := range []int{2, 0, 1, 4, 3} {
						f, err := m.File(context.Background(), v, p, index, 0)
						if err != nil {
							t.Fatal(err)
						}
						paths[index] = f.Name()
						packets := firstPacketTimes(t, f.Name())
						_ = f.Close()
						start := 1 + float64(index)*SegmentDuration
						if got, ok := packets[0]; !ok || math.Abs(got-start) > 1/p.FrameRate(v.Source)+.0001 {
							t.Fatalf("segment %d video clock: got %f, want %f within one frame", index, got, start)
						}
						if audio {
							// The first whole AAC frame on the shared sample grid.
							want := 1 + math.Ceil(float64(index)*SegmentDuration*48000/1024)*1024/48000
							if got, ok := packets[1]; !ok || math.Abs(got-want) > 1.0/48000 {
								t.Fatalf("segment %d audio clock: got %f, want %f", index, got, want)
							}
						}
					}
					if audio {
						assertAudioJoins(t, paths)
					}
				})
			}
			if c.Transcode.Encoder == "vaapi" && m.Diagnostics().Encoder != "h264_vaapi" {
				t.Fatal("hardware timing checks fell back to CPU")
			}
		})
	}
}

// assertAudioJoins checks for gaps and audio glitches between segments.
func assertAudioJoins(t *testing.T, paths []string) {
	t.Helper()
	input := "concat:" + strings.Join(paths, "|")
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "a:0", "-show_packets", "-show_entries", "packet=pts_time", "-of", "json", input).Output()
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Packets []struct {
			PTS string `json:"pts_time"`
		}
	}
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Packets) == 0 {
		t.Fatal("missing audio packets")
	}
	var previous float64
	for i, packet := range data.Packets {
		pts, err := strconv.ParseFloat(packet.PTS, 64)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && math.Abs(pts-previous-1024.0/48000) > 2.0/90000 {
			t.Fatalf("audio overlap or gap at packet %d: %f -> %f", i, previous, pts)
		}
		previous = pts
	}
	// Decode joined segments as a browser would, checking for silence or clicks.
	out, err = exec.Command("ffmpeg", "-v", "error", "-i", input, "-vn", "-ac", "1", "-ar", "48000", "-f", "f32le", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	pcm := make([]float64, len(out)/4)
	for i := range pcm {
		pcm[i] = float64(math.Float32frombits(binary.LittleEndian.Uint32(out[i*4:])))
	}
	if len(pcm) < (len(paths)-1)*2*48000+2400 {
		t.Fatalf("truncated joined audio: %d samples", len(pcm))
	}
	rms := func(samples []float64) float64 {
		var power float64
		for _, sample := range samples {
			power += sample * sample
		}
		return math.Sqrt(power / float64(len(samples)))
	}
	baseline := rms(pcm[48000:72000])
	if baseline < .01 {
		t.Fatal("tone fixture is silent")
	}
	for segment := 1; segment < len(paths); segment++ {
		second := segment * 2
		boundary := second * 48000
		for start := boundary - 2400; start < boundary+2400; start += 240 {
			if level := rms(pcm[start : start+240]); level < baseline*.6 {
				t.Fatalf("audio dip near %ds: 5ms RMS %.5f, baseline %.5f", second, level, baseline)
			}
		}
		for i := boundary - 2400; i < boundary+2400; i++ {
			if jump := math.Abs(pcm[i] - pcm[i-1]); jump > baseline*.6 {
				t.Fatalf("audio click near %ds: sample jump %.5f, baseline RMS %.5f", second, jump, baseline)
			}
		}
	}
}

// firstPacketTimes reads the first packet timestamp for each stream.
func firstPacketTimes(t *testing.T, path string) map[int]float64 {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-show_packets", "-show_entries", "packet=stream_index,pts_time,flags", "-of", "json", path).Output()
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Packets []struct {
			Stream int    `json:"stream_index"`
			PTS    string `json:"pts_time"`
			Flags  string `json:"flags"`
		}
	}
	if err := json.Unmarshal(out, &data); err != nil {
		t.Fatal(err)
	}
	first := make(map[int]float64)
	for _, packet := range data.Packets {
		if _, ok := first[packet.Stream]; ok {
			continue
		}
		if packet.Stream == 0 && (len(packet.Flags) == 0 || packet.Flags[0] != 'K') {
			t.Fatal("segment does not start with an independently decodable video frame")
		}
		pts, err := strconv.ParseFloat(packet.PTS, 64)
		if err != nil {
			t.Fatal(err)
		}
		first[packet.Stream] = pts
	}
	return first
}
