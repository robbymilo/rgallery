package transcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robbymilo/rgallery/pkg/config"
)

var ErrBusy = errors.New("video queue is full")
var managers sync.Map

type job struct {
	key      string
	priority int
	refs     int
	started  bool
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	run      func(context.Context) error
	err      error
	created  time.Time
}

type JobStatus struct {
	ID      string  `json:"id"`
	State   string  `json:"state"`
	Seconds float64 `json:"seconds"`
	Error   string  `json:"error,omitempty"`
}

type Diagnostics struct {
	Capability
	Workers           int         `json:"workers"`
	CacheBytes        int64       `json:"cacheBytes"`
	CacheLimitBytes   int64       `json:"cacheLimitBytes"`
	Jobs              []JobStatus `json:"jobs"`
	Recent            []JobStatus `json:"recent"`
	LastEncodeSeconds float64     `json:"lastEncodeSeconds"`
	LastEncodeSpeed   float64     `json:"lastEncodeSpeed"`
}

type Manager struct {
	conf              Conf
	root              string
	encoder           encoder
	mu                sync.Mutex
	jobs              map[string]*job
	sources           map[string]Source
	background        map[string]struct{}
	backgroundCtx     context.Context
	cancelBackground  context.CancelFunc
	backgroundSlots   chan struct{}
	recent            []JobStatus
	lastEncodeSeconds float64
	lastEncodeSpeed   float64
	wake              chan struct{}
	stop              chan struct{}
	wg                sync.WaitGroup
	cleanupMu         sync.Mutex
	logOnce           sync.Once
}

// LogConfiguration checks the encoder and logs the active settings.
func (m *Manager) LogConfiguration() {
	m.logOnce.Do(func() {
		cap := m.encoder.detect(m.conf)
		s := Settings(m.conf)
		p, _ := FindProfile(m.conf, s.Profile)
		preview, _ := FindProfile(m.conf, "preview")
		m.conf.Logger.Info("video transcoding configured", "requested_encoder", s.Encoder, "encoder", cap.Encoder, "device", cap.Device, "quality", p.ID, "max_long_edge", p.LongEdge, "max_video_kbps", p.MaxRate, "audio_kbps", p.AudioBitrate, "cpu_crf", p.CRF, "mode", s.Mode, "workers", s.Workers)
		m.conf.Logger.Info("video preview configuration", "generate_on_scan", m.conf.PreGenerateThumb, "max_long_edge", preview.LongEdge, "max_video_kbps", preview.MaxRate, "cpu_crf", preview.CRF, "duration_seconds", 4)
	})
}

// NewManager starts the encoding workers and cache cleanup loop.
func NewManager(c Conf) *Manager {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	m := &Manager{
		conf: c, root: filepath.Join(config.CachePath(c), "video", cacheVersion),
		jobs: make(map[string]*job), sources: make(map[string]Source),
		background: make(map[string]struct{}), backgroundCtx: backgroundCtx, cancelBackground: cancelBackground,
		backgroundSlots: make(chan struct{}, Settings(c).Workers),
		wake:            make(chan struct{}, 16), stop: make(chan struct{}),
	}
	for i := 0; i < Settings(c).Workers; i++ {
		m.wg.Add(1)
		go m.worker()
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-ticker.C:
				m.Cleanup()
			}
		}
	}()
	return m
}

// For shares a manager between scanning and playback.
// File locks coordinate cache access across processes.
func For(c Conf) *Manager {
	key := digest([]any{config.CachePath(c), c.Transcode, c.TranscodeResolution})
	if found, ok := managers.Load(key); ok {
		return found.(*Manager)
	}
	m := NewManager(c)
	actual, loaded := managers.LoadOrStore(key, m)
	if loaded {
		m.Close()
	}
	return actual.(*Manager)
}

// Close cancels pending work and waits for workers to stop.
func (m *Manager) Close() {
	m.mu.Lock()
	select {
	case <-m.stop:
		m.mu.Unlock()
		return
	default:
		close(m.stop)
	}
	m.cancelBackground()
	for _, j := range m.jobs {
		j.cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
}

// do shares queued work between callers and waits for its result.
func (m *Manager) do(ctx context.Context, key string, priority int, run func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	select {
	case <-m.stop:
		m.mu.Unlock()
		return context.Canceled
	default:
	}
	j, ok := m.jobs[key]
	if ok && j.ctx.Err() != nil {
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-j.done:
			return m.do(ctx, key, priority, run)
		}
	}
	if !ok {
		if len(m.jobs) >= 128 {
			m.mu.Unlock()
			return ErrBusy
		}
		jobCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		j = &job{key: key, priority: priority, ctx: jobCtx, cancel: cancel, done: make(chan struct{}), run: run, created: time.Now()}
		m.jobs[key] = j
	} else if priority < j.priority {
		j.priority = priority
	}
	j.refs++
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
	defer func() {
		m.mu.Lock()
		j.refs--
		if j.refs == 0 {
			j.cancel()
		}
		m.mu.Unlock()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.stop:
		return context.Canceled
	case <-j.done:
		return j.err
	}
}

// worker runs queued jobs in priority order.
func (m *Manager) worker() {
	defer m.wg.Done()
	for {
		m.mu.Lock()
		var next *job
		for _, j := range m.jobs {
			if !j.started && (next == nil || j.priority < next.priority || (j.priority == next.priority && j.created.Before(next.created))) {
				next = j
			}
		}
		if next != nil {
			next.started = true
		}
		m.mu.Unlock()
		if next == nil {
			select {
			case <-m.stop:
				return
			case <-m.wake:
				continue
			}
		}
		err := next.ctx.Err()
		if err == nil {
			err = next.run(next.ctx)
		}
		next.cancel()
		m.mu.Lock()
		next.err = err
		state := "complete"
		if err != nil {
			state = "failed"
		}
		if errors.Is(err, context.Canceled) {
			state = "canceled"
		}
		m.recent = append([]JobStatus{{ID: filepath.Base(next.key), State: state, Seconds: time.Since(next.created).Seconds(), Error: safeError(err)}}, m.recent...)
		if len(m.recent) > 20 {
			m.recent = m.recent[:20]
		}
		delete(m.jobs, next.key)
		close(next.done)
		m.mu.Unlock()
		if err != nil && !errors.Is(err, context.Canceled) {
			m.conf.Logger.Warn("video job failed", "job", next.key, "error", err)
		}
	}
}

type Video struct {
	Source  Source
	Hash    uint32
	Version string
	Dir     string
}

// Open loads source metadata and prepares its cache directory.
func (m *Manager) Open(ctx context.Context, path string, hash uint32) (*Video, error) {
	version, err := SourceVersion(path)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	source, ok := m.sources[version]
	m.mu.Unlock()
	if !ok {
		err = m.do(ctx, "probe-"+version, 0, func(ctx context.Context) error {
			m.mu.Lock()
			_, exists := m.sources[version]
			m.mu.Unlock()
			if exists {
				return nil
			}
			s, err := Probe(ctx, path)
			if err != nil {
				return err
			}
			m.mu.Lock()
			if len(m.sources) > 512 {
				m.sources = make(map[string]Source)
			}
			m.sources[version] = s
			m.mu.Unlock()
			return nil
		})
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		source, ok = m.sources[version]
		m.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("source changed while probing; retry playback")
		}
	}
	outputVersion := m.outputVersion(source.Version)
	dir := filepath.Join(m.root, fmt.Sprint(hash), outputVersion)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Video{Source: source, Hash: hash, Version: outputVersion, Dir: dir}, nil
}

// outputVersion builds a cache key from the source and encoder settings.
func (m *Manager) outputVersion(sourceVersion string) string {
	s := Settings(m.conf)
	profiles := Profiles(m.conf)
	preview, _ := FindProfile(m.conf, "preview")
	profiles = append(profiles, preview)
	var crfs []int
	for _, p := range profiles {
		crfs = append(crfs, p.CRF)
	}
	return digest([]any{sourceVersion, profiles, crfs, s.Profile, s.CRF, s.Preset, s.Encoder, s.Device, cacheVersion, encoderVersion})
}

type CachedFile struct {
	*os.File
	release func()
}

// Close closes the file and releases its cache lock.
func (f *CachedFile) Close() error { err := f.File.Close(); f.release(); return err }

// File keeps the cache locked until the caller closes the file.
func (m *Manager) File(ctx context.Context, v *Video, p Profile, index, priority int) (*CachedFile, error) {
	if index < 0 || float64(index)*SegmentDuration >= v.Source.Duration {
		return nil, os.ErrNotExist
	}
	if p.ID == "preview" && index != 0 {
		return nil, os.ErrNotExist
	}
	release, err := fileLock(ctx, v.Dir+".lock", false, true)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			release()
		}
	}()
	if err := os.MkdirAll(filepath.Join(v.Dir, p.ID), 0755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%06d.ts", index)
	if p.ID == "preview" {
		name = "preview.mp4"
	}
	path := filepath.Join(v.Dir, p.ID, name)
	if !validOutput(path) {
		// Clear the completion marker if a segment is missing.
		_ = os.Remove(filepath.Join(v.Dir, p.ID, renditionCompleteFile))
		err = m.do(ctx, path, priority, func(ctx context.Context) error {
			// Keep the cache locked while a canceled worker exits.
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
			f, err := os.CreateTemp(filepath.Dir(path), ".encoding-*")
			if err != nil {
				return err
			}
			tmp := f.Name()
			_ = f.Close()
			defer func() { _ = os.Remove(tmp) }()
			if err := m.encode(ctx, v, p, index, tmp); err != nil {
				return err
			}
			version, err := SourceVersion(v.Source.Path)
			if err != nil {
				return err
			}
			if version != v.Source.Version {
				return fmt.Errorf("source changed during encoding")
			}
			if !validOutput(tmp) {
				return fmt.Errorf("encoder produced an empty segment")
			}
			if err := os.Rename(tmp, path); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	v.touch()
	ok = true
	return &CachedFile{File: f, release: release}, nil
}

// touch saves source details and updates the last access time for cleanup.
// Call it while holding the cache lock.
func (v *Video) touch() {
	path := filepath.Join(v.Dir, "source.json")
	if !validOutput(path) {
		metadata, _ := json.Marshal(struct {
			Path    string
			Version string
		}{v.Source.Path, v.Source.Version})
		if f, err := os.CreateTemp(v.Dir, ".source-*"); err == nil {
			tmp := f.Name()
			_, writeErr := f.Write(metadata)
			closeErr := f.Close()
			if writeErr == nil && closeErr == nil {
				_ = os.Rename(tmp, path)
			}
			_ = os.Remove(tmp)
		}
	}
	now := time.Now()
	_ = os.Chtimes(v.Dir, now, now)
}

// Diagnostics reports encoder status, jobs, and cache usage.
func (m *Manager) Diagnostics() Diagnostics {
	d := Diagnostics{Capability: m.encoder.detect(m.conf), Workers: Settings(m.conf).Workers, CacheLimitBytes: int64(Settings(m.conf).CacheMB) * 1024 * 1024, Jobs: []JobStatus{}, Recent: []JobStatus{}}
	m.mu.Lock()
	for _, j := range m.jobs {
		state := "queued"
		if j.started {
			state = "preparing"
		}
		d.Jobs = append(d.Jobs, JobStatus{ID: filepath.Base(j.key), State: state, Seconds: time.Since(j.created).Seconds()})
	}
	d.Recent = append(d.Recent, m.recent...)
	d.LastEncodeSeconds = m.lastEncodeSeconds
	d.LastEncodeSpeed = m.lastEncodeSpeed
	m.mu.Unlock()
	_ = filepath.WalkDir(m.root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			if info, e := entry.Info(); e == nil {
				d.CacheBytes += info.Size()
			}
		}
		return nil
	})
	return d
}

// Cleanup removes old or excess cache entries.
// File locks protect active readers and encoders across processes.
func (m *Manager) Cleanup() {
	if !m.cleanupMu.TryLock() {
		return
	}
	defer m.cleanupMu.Unlock()
	type candidate struct {
		path     string
		size     int64
		modified time.Time
		stale    bool
		partial  bool
	}
	var all []candidate
	var total int64
	hashes, _ := os.ReadDir(m.root)
	for _, hash := range hashes {
		if !hash.IsDir() {
			continue
		}
		versions, _ := os.ReadDir(filepath.Join(m.root, hash.Name()))
		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			path := filepath.Join(m.root, hash.Name(), version.Name())
			info, err := version.Info()
			if err != nil {
				continue
			}
			item := candidate{path: path, modified: info.ModTime()}
			_ = filepath.WalkDir(path, func(_ string, e os.DirEntry, err error) error {
				if err == nil && !e.IsDir() {
					if info, err := e.Info(); err == nil {
						item.size += info.Size()
						if strings.HasPrefix(e.Name(), ".encoding-") || (strings.HasPrefix(e.Name(), ".remux-") || strings.HasPrefix(e.Name(), ".source-")) {
							item.partial = true
						}
					}
				}
				return nil
			})
			var source struct {
				Path    string
				Version string
			}
			b, err := os.ReadFile(filepath.Join(path, "source.json"))
			if err == nil && json.Unmarshal(b, &source) == nil {
				current, err := SourceVersion(source.Path)
				item.stale = errors.Is(err, os.ErrNotExist) || (err == nil && (current != source.Version || m.outputVersion(source.Version) != version.Name()))
			}
			if time.Since(item.modified) > 7*24*time.Hour {
				item.stale = true
			}
			all = append(all, item)
			total += item.size
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].stale != all[j].stale {
			return all[i].stale
		}
		return all[i].modified.Before(all[j].modified)
	})
	limit := int64(Settings(m.conf).CacheMB) * 1024 * 1024
	for _, item := range all {
		if !item.stale && total <= limit && !item.partial {
			continue
		}
		unlock, err := fileLock(context.Background(), item.path+".lock", true, false)
		if err != nil {
			continue
		}
		if item.stale || total > limit {
			if err := os.RemoveAll(item.path); err == nil {
				total -= item.size
			}
		} else if item.partial {
			_ = filepath.WalkDir(item.path, func(path string, e os.DirEntry, err error) error {
				if err == nil && !e.IsDir() && (strings.HasPrefix(e.Name(), ".encoding-") || (strings.HasPrefix(e.Name(), ".remux-") || strings.HasPrefix(e.Name(), ".source-"))) {
					_ = os.Remove(path)
				}
				return nil
			})
		}
		unlock()
	}
}

// Pregenerate creates the previews and playback files enabled for scanning.
func Pregenerate(ctx context.Context, original string, hash uint32, c Conf) error {
	settings := Settings(c)
	if settings.Mode == "ondemand" && !c.PreGenerateThumb {
		return nil
	}
	m := For(c)
	v, err := m.Open(ctx, original, hash)
	if err != nil {
		return err
	}
	if c.PreGenerateThumb {
		preview, _ := FindProfile(c, "preview")
		f, err := m.File(ctx, v, preview, 0, 2)
		if err != nil {
			return fmt.Errorf("generate video preview: %w", err)
		}
		_ = f.Close()
		_, count, _, _ := thumbnailLayout(v)
		for sheet := 0; sheet < (count+thumbnailsPerSheet-1)/thumbnailsPerSheet; sheet++ {
			f, err := m.ThumbnailSheet(ctx, v, sheet, 2)
			if err != nil {
				return fmt.Errorf("generate video seek thumbnails: %w", err)
			}
			_ = f.Close()
		}
	}
	if settings.Mode == "ondemand" {
		return nil
	}
	profiles := Profiles(c)
	if settings.Mode == "hybrid" {
		p, _ := FindProfile(c, settings.Profile)
		profiles = []Profile{p}
	}
	for _, p := range profiles {
		for i := 0; float64(i)*SegmentDuration < v.Source.Duration; i++ {
			f, err := m.File(ctx, v, p, i, 2)
			if err != nil {
				return err
			}
			_ = f.Close()
		}
		m.Cleanup()
	}
	return nil
}
