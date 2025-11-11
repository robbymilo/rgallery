package transcode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/types"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Conf = types.Conf

// TranscodeVideo splits a video into .ts files and saves the output.
func TranscodeVideo(original, cache string, hash uint32) error {
	err := os.MkdirAll(filepath.Dir(cache), os.ModePerm)
	if err != nil {
		return err
	}

	err = ffmpeg.Input(original).
		Output(cache, ffmpeg.KwArgs{
			"c:v":           "libx264",   // Encode video with H.264
			"crf":           "23",        // Quality control (lower = better)
			"preset":        "ultrafast", // Fast encoding
			"c:a":           "aac",       // Re-encode audio to AAC
			"b:a":           "128k",      // Audio bitrate
			"movflags":      "+faststart",
			"start_number":  0,
			"hls_time":      2,
			"hls_list_size": 0,
			"f":             "hls",
		}).
		Run()

	return err
}

// CreateHLSIndexFilePath creates a string of the path where the HLS index file lives.
func CreateHLSIndexFilePath(hash uint32, c Conf) string {
	return filepath.Join(config.CachePath(c), "video", fmt.Sprint(hash), "index.m3u8")
}

// CreateTSFilePath creates a string of the path where the TS files live.
func CreateTSFilePath(hash uint32, file string, c Conf) string {
	return filepath.Join(config.CachePath(c), "video", fmt.Sprint(hash), file)
}
