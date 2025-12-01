package transcode

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/types"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Conf = types.Conf

var transcodeLock sync.Mutex

// TranscodeVideo splits a video into .ts files and saves the output.
func TranscodeVideo(original, cache string, hash uint32, c Conf) error {
	c.Logger.Info("starting transcoding", "file", original)

	err := os.MkdirAll(filepath.Dir(cache), os.ModePerm)
	if err != nil {
		return err
	}

	err = ffmpeg.Input(original).
		Output(cache, ffmpeg.KwArgs{
			"c:v":           "libx264",
			"crf":           "23",
			"preset":        "veryfast",
			"maxrate":       "1500k",
			"bufsize":       "3000k",
			"vf":            fmt.Sprintf("scale=-2:%d", c.TranscodeResolution),
			"c:a":           "aac",
			"b:a":           "96k",
			"movflags":      "+faststart",
			"start_number":  0,
			"hls_time":      1,
			"hls_list_size": 0,
			"f":             "hls",
		}).
		Run()

	c.Logger.Info("finished transcoding", "file", original)

	return err
}

// TranscodeWithLock ensures only one video is transcoded at a time.
func TranscodeWithLock(original, cache string, hash uint32, c Conf) error {
	// Acquire the lock
	transcodeLock.Lock()
	defer transcodeLock.Unlock()

	// Perform the transcoding
	return TranscodeVideo(original, cache, hash, c)
}

// CreateHLSIndexFilePath creates a string of the path where the HLS index file lives.
func CreateHLSIndexFilePath(hash uint32, c Conf) string {
	return filepath.Join(config.CachePath(c), "video", fmt.Sprint(hash), "index.m3u8")
}

// CreateTSFilePath creates a string of the path where the TS files live.
func CreateTSFilePath(hash uint32, file string, c Conf) string {
	return filepath.Join(config.CachePath(c), "video", fmt.Sprint(hash), file)
}
