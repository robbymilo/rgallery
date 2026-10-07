package rgallery_test

import (
	"encoding/json"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/rgallery"
	"github.com/robbymilo/rgallery/pkg/transcode"
	"github.com/stretchr/testify/require"
)

// Give chi's logging wrapper the ReaderFrom method it expects.
type videoRecorder struct{ *httptest.ResponseRecorder }

// ReadFrom copies response bytes for the test HTTP writer.
func (w videoRecorder) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{w.ResponseRecorder}, r)
}

// videoRequest makes an authenticated video request with an optional byte range.
func videoRequest(f *authFixture, path string, cookie *http.Cookie, rangeHeader string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	r.AddCookie(cookie)
	if rangeHeader != "" {
		r.Header.Set("Range", rangeHeader)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(videoRecorder{w}, r)
	return w
}

// newVideoFixture creates an indexed video for HTTP tests.
func newVideoFixture(t *testing.T) (*authFixture, string) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg required")
	}
	f := newAuthFixture(t)
	f.c.Transcode.Encoder = "cpu"
	f.router = rgallery.SetupRouter(f.c, cache.New(-1, -1), "test", "test")
	require.NoError(t, os.MkdirAll(f.c.Media, 0755))
	path := filepath.Join(f.c.Media, "video.mp4")
	output, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=30", "-t", "6.2", "-c:v", "libx264", "-preset", "ultrafast", "-y", path).CombinedOutput()
	require.NoError(t, err, string(output))
	_, err = f.db.Exec(`INSERT INTO media (hash,path,subject,width,height,ratio,padding,date,modified,folder,mediatype) VALUES (1,'video.mp4','[]',320,180,1.777,56,'2026-01-01T00:00:00.000Z','2026-01-01T00:00:00Z','','video')`)
	require.NoError(t, err)
	return f, path
}

// TestVideoRoutesRequireAuthenticationAndDiagnosticsRequireAdmin checks video route access controls.
func TestVideoRoutesRequireAuthenticationAndDiagnosticsRequireAdmin(t *testing.T) {
	f := newAuthFixture(t)
	for _, path := range []string{"/api/transcode/1/info", "/api/transcode/1/prepare", "/api/transcode/1/index.m3u8", "/api/transcode/1/small/000000.ts", "/api/transcode/1/preview.mp4", "/api/transcode/1/source.mp4", "/api/transcode/1/thumbnails.json", "/api/transcode/1/thumbnails/000000.jpg", "/api/transcode/diagnostics"} {
		require.Equal(t, http.StatusUnauthorized, f.request("GET", path, "", "", nil).Code, path)
	}
	f.seedViewer()
	viewer := f.login("reader", "viewer-password")
	require.Equal(t, http.StatusForbidden, f.request("GET", "/api/transcode/diagnostics", "", "", viewer).Code)
	admin := f.login("admin", "admin")
	require.Equal(t, http.StatusOK, f.request("GET", "/api/transcode/diagnostics", "", "", admin).Code)
}

// TestVideoSeekThumbnailRoutes checks thumbnail responses, access, and invalidation.
func TestVideoSeekThumbnailRoutes(t *testing.T) {
	f, path := newVideoFixture(t)
	t.Cleanup(transcode.For(f.c).Close)
	user := f.login("admin", "admin")
	metadata := f.request("GET", "/api/transcode/1/thumbnails.json", "", "", user)
	require.Equal(t, http.StatusOK, metadata.Code, metadata.Body.String())
	require.Equal(t, "application/json", metadata.Header().Get("Content-Type"))
	var thumbs []transcode.Thumbnail
	require.NoError(t, json.Unmarshal(metadata.Body.Bytes(), &thumbs))
	require.Len(t, thumbs, 2)
	require.Equal(t, float64(0), thumbs[0].StartTime)
	require.Equal(t, float64(5), thumbs[1].StartTime)
	sheet := videoRequest(f, thumbs[0].URL, user, "")
	require.Equal(t, http.StatusOK, sheet.Code, sheet.Body.String())
	require.Equal(t, "image/jpeg", sheet.Header().Get("Content-Type"))
	require.Contains(t, sheet.Header().Get("Cache-Control"), "private")
	img, err := jpeg.Decode(sheet.Body)
	require.NoError(t, err)
	require.Equal(t, 640, img.Bounds().Dx())
	require.Equal(t, 180, img.Bounds().Dy())
	partial := videoRequest(f, thumbs[0].URL, user, "bytes=0-63")
	require.Equal(t, http.StatusPartialContent, partial.Code)
	require.Equal(t, 64, partial.Body.Len())
	for _, name := range []string{"000099.jpg", "-00001.jpg", "0.jpg", "000000.ts"} {
		require.Equal(t, http.StatusNotFound, videoRequest(f, "/api/transcode/1/thumbnails/"+name, user, "").Code, name)
	}
	now := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(path, now, now))
	require.Equal(t, http.StatusConflict, videoRequest(f, thumbs[0].URL, user, "").Code)
	_, err = f.db.Exec("DELETE FROM media WHERE hash=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, f.request("GET", "/api/transcode/1/thumbnails.json", "", "", user).Code)
	require.Equal(t, http.StatusNotFound, videoRequest(f, thumbs[0].URL, user, "").Code)
}

// TestVideoVODPrepareSeekAndInvalidation checks playback preparation, seeking, and source changes.
func TestVideoVODPrepareSeekAndInvalidation(t *testing.T) {
	f, path := newVideoFixture(t)
	user := f.login("admin", "admin")
	info := f.request("GET", "/api/transcode/1/info", "", "", user)
	require.Equal(t, http.StatusOK, info.Code, info.Body.String())
	var data struct {
		Version  string  `json:"version"`
		Duration float64 `json:"duration"`
	}
	require.NoError(t, json.Unmarshal(info.Body.Bytes(), &data))
	require.InDelta(t, 6.2, data.Duration, 0.05)
	prepared := f.request("GET", "/api/transcode/1/prepare?quality=saver&start=4", "", "", user)
	require.Equal(t, http.StatusOK, prepared.Code, prepared.Body.String())
	var result struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.Unmarshal(prepared.Body.Bytes(), &result))
	master := f.request("GET", result.URL, "", "", user)
	require.Equal(t, http.StatusOK, master.Code, master.Body.String())
	require.Contains(t, master.Body.String(), "saver/index.m3u8")
	require.NotContains(t, master.Body.String(), "small/index.m3u8")
	playlist := f.request("GET", "/api/transcode/1/saver/index.m3u8?v="+data.Version, "", "", user)
	require.Contains(t, playlist.Body.String(), "#EXT-X-ENDLIST")
	require.Contains(t, playlist.Header().Get("Content-Type"), "mpegurl")
	segmentURL := "/api/transcode/1/saver/000002.ts?v=" + data.Version
	segment := videoRequest(f, segmentURL, user, "")
	require.Equal(t, http.StatusOK, segment.Code, segment.Body.String())
	require.Equal(t, "video/mp2t", segment.Header().Get("Content-Type"))
	require.Greater(t, segment.Body.Len(), 100)
	partial := videoRequest(f, segmentURL, user, "bytes=0-63")
	require.Equal(t, http.StatusPartialContent, partial.Code)
	require.Equal(t, 64, partial.Body.Len())
	for _, tail := range []string{"prepare?start=NaN", "prepare?start=-1", "prepare?start=Infinity", "prepare?quality=bogus"} {
		require.Equal(t, http.StatusBadRequest, f.request("GET", "/api/transcode/1/"+tail, "", "", user).Code, tail)
	}
	require.Equal(t, http.StatusNotFound, f.request("GET", "/api/transcode/1/saver/000099.ts", "", "", user).Code)
	now := time.Now().Add(time.Second)
	require.NoError(t, os.Chtimes(path, now, now))
	require.Equal(t, http.StatusConflict, f.request("GET", segmentURL, "", "", user).Code)
	_, err := f.db.Exec("DELETE FROM media WHERE hash=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, f.request("GET", segmentURL, "", "", user).Code)
}

// TestVideoSourceRespectsBandwidthCapAndRangeRequests checks direct playback limits and byte ranges.
func TestVideoSourceRespectsBandwidthCapAndRangeRequests(t *testing.T) {
	f, _ := newVideoFixture(t)
	user := f.login("admin", "admin")
	for _, file := range []string{"source.mp4", "remux.mp4"} {
		partial := videoRequest(f, "/api/transcode/1/"+file+"?quality=high", user, "bytes=0-127")
		require.Equal(t, http.StatusPartialContent, partial.Code, partial.Body.String())
		require.Equal(t, 128, partial.Body.Len())
		require.Equal(t, "video/mp4", partial.Header().Get("Content-Type"))
	}
	f.c.Transcode.MaxRate = 1
	f.router = rgallery.SetupRouter(f.c, cache.New(-1, -1), "test", "test")
	response := f.request("GET", "/api/transcode/1/source.mp4", "", "", user)
	require.Equal(t, http.StatusForbidden, response.Code)
	info := f.request("GET", "/api/transcode/1/info", "", "", user)
	require.True(t, strings.Contains(info.Body.String(), `"direct":null`), info.Body.String())
}
