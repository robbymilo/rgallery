package transcode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Playlist lists the full video timeline. Seeking does not wait for earlier segments.
func Playlist(v *Video, p Profile) string {
	var b strings.Builder
	fmt.Fprint(&b, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:2\n#EXT-X-MEDIA-SEQUENCE:0\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-INDEPENDENT-SEGMENTS\n")
	for i := 0; float64(i)*SegmentDuration < v.Source.Duration; i++ {
		fmt.Fprintf(&b, "#EXTINF:%.6f,\n/api/transcode/%d/%s/%06d.ts?v=%s\n", min(SegmentDuration, v.Source.Duration-float64(i)*SegmentDuration), v.Hash, p.ID, i, v.Version)
	}
	fmt.Fprint(&b, "#EXT-X-ENDLIST\n")
	return b.String()
}

// Remux caches an MP4 copy without re-encoding its tracks.
func (m *Manager) Remux(ctx context.Context, v *Video) (*CachedFile, error) {
	release, err := fileLock(ctx, v.Dir+".lock", false, true)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(v.Dir, 0755); err != nil {
		release()
		return nil, err
	}
	path := filepath.Join(v.Dir, "remux.mp4")
	if !validOutput(path) {
		err = m.do(ctx, path, 0, func(ctx context.Context) error {
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
			f, err := os.CreateTemp(v.Dir, ".remux-*")
			if err != nil {
				return err
			}
			tmp := f.Name()
			_ = f.Close()
			defer func() { _ = os.Remove(tmp) }()
			args := []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-i", v.Source.Path, "-map", fmt.Sprintf("0:%d", v.Source.VideoIndex)}
			if v.Source.Audio {
				args = append(args, "-map", fmt.Sprintf("0:%d", v.Source.AudioIndex))
			}
			args = append(args, "-c", "copy", "-movflags", "+faststart", "-f", "mp4", tmp)
			if _, err := runFFmpeg(ctx, args); err != nil {
				return err
			}
			version, err := SourceVersion(v.Source.Path)
			if err != nil {
				return err
			}
			if version != v.Source.Version {
				return fmt.Errorf("source changed during remux")
			}
			return os.Rename(tmp, path)
		})
	}
	if err != nil {
		release()
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		release()
		return nil, err
	}
	v.touch()
	return &CachedFile{File: f, release: release}, nil
}
