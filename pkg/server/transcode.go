package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/transcode"
)

// videoError sends the HTTP response for a playback error.
func videoError(w http.ResponseWriter, err error) {
	code, msg := http.StatusInternalServerError, "Unable to prepare this video. Please retry."
	switch {
	case errors.Is(err, os.ErrNotExist):
		code, msg = http.StatusNotFound, "Video not found."
	case errors.Is(err, transcode.ErrBusy):
		code, msg = http.StatusServiceUnavailable, "Video server is busy. Please retry shortly."
		w.Header().Set("Retry-After", "2")
	case errors.Is(err, context.DeadlineExceeded):
		code, msg = http.StatusGatewayTimeout, "Video preparation timed out. Please retry."
	case errors.Is(err, context.Canceled):
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Check access and library membership before serving cached video.
func openVideo(r *http.Request, c Conf) (*transcode.Manager, *transcode.Video, error) {
	hash, err := strconv.ParseUint(chi.URLParam(r, "hash"), 10, 32)
	if err != nil {
		return nil, nil, os.ErrNotExist
	}
	media, err := queries.GetSingleMediaItem(uint32(hash), c)
	if err != nil {
		c.Logger.Warn("video lookup failed", "error", err)
		return nil, nil, err
	}
	if media.Type != "video" {
		return nil, nil, os.ErrNotExist
	}
	root := config.MediaPath(c)
	path, err := filepath.EvalSymlinks(filepath.Join(root, media.Path))
	if err != nil {
		return nil, nil, err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, nil, err
	}
	rel, err := filepath.Rel(resolvedRoot, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, nil, os.ErrNotExist
	}
	m := transcode.For(c)
	v, err := m.Open(r.Context(), path, uint32(hash))
	return m, v, err
}

// selectedProfiles returns the qualities allowed by the player selection.
func selectedProfiles(c Conf, quality string) ([]transcode.Profile, error) {
	if quality != "auto" {
		p, err := transcode.FindProfile(c, quality)
		if err != nil || p.ID == "preview" {
			return nil, fmt.Errorf("invalid quality")
		}
		return []transcode.Profile{p}, nil
	}
	var profiles []transcode.Profile
	for _, p := range transcode.Profiles(c) {
		profiles = append(profiles, p)
		if p.ID == transcode.Settings(c).Profile {
			break
		}
	}
	return profiles, nil
}

// prepareVideo prepares the starting segment of each advertised quality.
func prepareVideo(ctx context.Context, m *transcode.Manager, v *transcode.Video, profiles []transcode.Profile, start float64) error {
	index := int(start / transcode.SegmentDuration)
	var wg sync.WaitGroup
	errs := make(chan error, len(profiles))
	for _, p := range profiles {
		wg.Add(1)
		go func(p transcode.Profile) {
			defer wg.Done()
			f, err := m.File(ctx, v, p, index, 0)
			if err == nil {
				err = f.Close()
			}
			errs <- err
		}(p)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// ServeTranscode serves video metadata, playback files, and seek thumbnails.
func ServeTranscode(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	file := chi.URLParam(r, "file")
	profileID := chi.URLParam(r, "profile")
	quality := r.URL.Query().Get("quality")
	if quality == "" {
		quality = "auto"
	}
	profiles, err := selectedProfiles(c, quality)
	if err != nil {
		http.Error(w, "Invalid video quality", http.StatusBadRequest)
		return
	}
	m, v, err := openVideo(r, c)
	if err != nil {
		videoError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	if version := r.URL.Query().Get("v"); version != "" && version != v.Version {
		http.Error(w, "Video changed; reload playback", http.StatusConflict)
		return
	}
	base := fmt.Sprintf("/api/transcode/%d", v.Hash)
	start := 0.0
	if value := r.URL.Query().Get("start"); value != "" {
		start, err = strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(start) || math.IsInf(start, 0) || start < 0 || start >= v.Source.Duration {
			http.Error(w, "Invalid playback position", http.StatusBadRequest)
			return
		}
	}
	if profileID == "" {
		switch file {
		case "thumbnails.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(transcode.Thumbnails(v))
			return
		case "info":
			var direct any
			p := profiles[len(profiles)-1]
			if v.Source.DirectEligible(p) {
				kind := "source.mp4"
				if !v.Source.IsMP4() {
					kind = "remux.mp4"
				}
				codec := "video/mp4; codecs=\"avc1.640028\""
				if v.Source.Audio {
					codec = "video/mp4; codecs=\"avc1.640028, mp4a.40.2\""
				}
				direct = map[string]string{"url": fmt.Sprintf("%s/%s?v=%s&quality=%s", base, kind, v.Version, p.ID), "type": codec}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"duration": v.Source.Duration, "width": v.Source.Width, "height": v.Source.Height, "frameRate": v.Source.FrameRate, "version": v.Version, "profiles": transcode.Profiles(c), "defaultQuality": transcode.Settings(c).Profile, "direct": direct})
			return
		case "prepare", "master.m3u8":
			if file == "prepare" {
				var allowed []string
				for _, p := range profiles {
					allowed = append(allowed, p.ID)
				}
				c.Logger.Info("video playback requested", "media", v.Hash, "quality", quality, "available_profiles", allowed, "start_seconds", start)
			}
			if err := prepareVideo(r.Context(), m, v, profiles, start); err != nil {
				videoError(w, err)
				return
			}
			if file == "prepare" {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"url": fmt.Sprintf("%s/master.m3u8?v=%s&quality=%s&start=%.6f", base, v.Version, quality, start), "version": v.Version})
				return
			}
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			_, _ = fmt.Fprint(w, "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-INDEPENDENT-SEGMENTS\n")
			for _, p := range profiles {
				width, height := transcode.Dimensions(v.Source.Width, v.Source.Height, p.LongEdge)
				_, _ = fmt.Fprintf(w, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,FRAME-RATE=%.3f\n%s/index.m3u8?v=%s\n", (p.MaxRate*2+p.AudioBitrate)*1100, width, height, p.FrameRate(v.Source), p.ID, v.Version)
			}
			return
		case "source.mp4", "remux.mp4":
			p := profiles[len(profiles)-1]
			if !v.Source.DirectEligible(p) {
				http.Error(w, "Source exceeds playback policy", http.StatusForbidden)
				return
			}
			c.Logger.Info("video direct playback", "media", v.Hash, "quality", p.ID, "source", file, "bitrate_kbps", v.Source.Bitrate/1000, "resolution", fmt.Sprintf("%dx%d", v.Source.Width, v.Source.Height))
			if file == "source.mp4" && v.Source.IsMP4() {
				w.Header().Set("Content-Type", "video/mp4")
				http.ServeFile(w, r, v.Source.Path)
				return
			}
			f, err := m.Remux(r.Context(), v)
			if err != nil {
				videoError(w, err)
				return
			}
			defer func() { _ = f.Close() }()
			w.Header().Set("Content-Type", "video/mp4")
			info, err := f.Stat()
			if err != nil {
				videoError(w, err)
				return
			}
			http.ServeContent(w, r, "video.mp4", info.ModTime(), f)
			return
		case "preview.mp4":
			profileID = "preview"
		case "index.m3u8":
			profileID = transcode.Settings(c).Profile
		default:
			http.NotFound(w, r)
			return
		}
	}
	if profileID == "thumbnails" {
		sheet, err := strconv.Atoi(strings.TrimSuffix(file, ".jpg"))
		if err != nil || fmt.Sprintf("%06d.jpg", sheet) != file {
			http.NotFound(w, r)
			return
		}
		f, err := m.ThumbnailSheet(r.Context(), v, sheet, 1)
		if err != nil {
			videoError(w, err)
			return
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil {
			videoError(w, err)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "private, max-age=3600")
		}
		http.ServeContent(w, r, file, info.ModTime(), f)
		return
	}
	p, err := transcode.FindProfile(c, profileID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if file == "index.m3u8" && p.ID != "preview" {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		_, _ = fmt.Fprint(w, transcode.Playlist(v, p))
		return
	}
	index := 0
	if p.ID != "preview" {
		if !strings.HasSuffix(file, ".ts") {
			http.NotFound(w, r)
			return
		}
		index, err = strconv.Atoi(strings.TrimSuffix(file, ".ts"))
		if err != nil || fmt.Sprintf("%06d.ts", index) != file {
			http.NotFound(w, r)
			return
		}
	} else if file != "preview.mp4" {
		http.NotFound(w, r)
		return
	}
	f, err := m.PlaybackFile(r.Context(), v, p, index)
	if err != nil {
		videoError(w, err)
		return
	}
	defer func() { _ = f.Close() }()
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "private, max-age=3600")
	}
	w.Header().Set("Content-Type", "video/mp2t")
	if p.ID == "preview" {
		w.Header().Set("Content-Type", "video/mp4")
	}
	info, err := f.Stat()
	if err != nil {
		videoError(w, err)
		return
	}
	http.ServeContent(w, r, file, info.ModTime(), f)
}

// ServeVideoDiagnostics returns the current encoder and cache status.
func ServeVideoDiagnostics(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(transcode.For(c).Diagnostics())
}
