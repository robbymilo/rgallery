package scanner

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/transcode"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/stretchr/testify/require"
)

// TestNormalScanPreparesPreviewForUnchangedVideo checks scan generation and reuse for indexed videos.
func TestNormalScanPreparesPreviewForUnchangedVideo(t *testing.T) {
	for _, executable := range []string{"ffmpeg", "ffprobe", "exiftool"} {
		if _, err := exec.LookPath(executable); err != nil {
			t.Skipf("%s is required", executable)
		}
	}
	root := t.TempDir()
	c := Conf{Media: filepath.Join(root, "media"), Data: filepath.Join(root, "data"), Cache: filepath.Join(root, "cache"), PreGenerateThumb: true, LocationService: "http://unused.invalid", Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Transcode: types.TranscodeConfig{Encoder: "cpu"}}
	require.NoError(t, os.MkdirAll(c.Media, 0755))
	path := filepath.Join(c.Media, "video.mp4")
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30", "-t", "4.2", "-c:v", "libx264", "-preset", "ultrafast", "-y", path).CombinedOutput()
	require.NoError(t, err, string(out))
	file, err := os.Stat(path)
	require.NoError(t, err)
	database.CreateDB(c)
	db, err := sql.Open("sqlite", database.NewSqlConnectionString(c))
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`INSERT INTO media(hash,path,subject,width,height,ratio,padding,date,modified,folder,mediatype,rating) VALUES(1,'video.mp4','[]',1280,720,1.777,56.25,'2020-01-01T00:00:00.000Z',?,'','video',0)`, file.ModTime().UTC().Format(time.RFC3339))
	require.NoError(t, err)
	m := transcode.For(c)
	defer m.Close()
	SetScanInProgress(false)
	t.Cleanup(func() { SetScanInProgress(false); resetCancelChan(nil) })
	_, err = Scan("default", c, cache.New(-1, -1))
	require.NoError(t, err)
	v, err := m.Open(context.Background(), path, 1)
	require.NoError(t, err)
	preview := filepath.Join(v.Dir, "preview", "preview.mp4")
	info, err := os.Stat(preview)
	require.NoError(t, err, "normal scan must create previews for already-indexed videos")
	source, err := transcode.Probe(context.Background(), preview)
	require.NoError(t, err)
	require.Equal(t, 1280, source.Width)
	require.Equal(t, 720, source.Height)
	require.False(t, source.Audio)
	require.InDelta(t, 4, source.Duration, 0.1)
	sheets, err := filepath.Glob(filepath.Join(v.Dir, "thumbnails-*", "*.jpg"))
	require.NoError(t, err)
	require.Len(t, sheets, 1, "normal scan must create seek thumbnails for already-indexed videos")
	sheet, err := os.Stat(sheets[0])
	require.NoError(t, err)
	require.Positive(t, sheet.Size())
	segments, err := filepath.Glob(filepath.Join(v.Dir, "*", "*.ts"))
	require.NoError(t, err)
	require.Empty(t, segments, "on-demand scan must not pregenerate full playback")
	_, err = Scan("default", c, cache.New(-1, -1))
	require.NoError(t, err)
	cached, err := os.Stat(preview)
	require.NoError(t, err)
	require.Equal(t, info.ModTime(), cached.ModTime(), "unchanged previews should be reused")
	cachedSheet, err := os.Stat(sheets[0])
	require.NoError(t, err)
	require.Equal(t, sheet.ModTime(), cachedSheet.ModTime(), "unchanged seek thumbnails should be reused")
}
