package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	exiftool "github.com/barasher/go-exiftool"
	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/geo"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/resize"
	"github.com/robbymilo/rgallery/pkg/sizes"
	"github.com/robbymilo/rgallery/pkg/types"
)

type Media = types.Media
type Conf = types.Conf

// BackgroundScan allows us to call Scan as a goroutine and still report a status/error.
func BackgroundScan(scanType string, c Conf, cache *cache.Cache) {
	_, err := Scan(scanType, c, cache)
	if err != nil {
		c.Logger.Error("error background scanning", "error", err)
		return
	}
}

// Scan coordinates the addition, updating, and removal of all media items.
func Scan(scanType string, c Conf, cache *cache.Cache) (status string, returnErr error) {

	start := time.Now()
	total := 0

	// Check if a scan is already in progress using our global variable
	if scanType == "thumb" {
		return "", nil
	}
	ctx, claimed := beginScan()
	if !claimed {
		c.Logger.Info("scan already in progress")
		if err := queries.Notify(c, "Scan already in progress.", "scanning"); err != nil {
			c.Logger.Error("failed to notify", "err", err)
		}

		return "scan already in progress", nil
	}
	defer func() {
		endScan()
		resize.SetExternalProcessObserver(nil)
		if r := recover(); r != nil {
			c.Logger.Error("recovered from panic in scanner", "panic", r)
			status = "scan failed"
			returnErr = fmt.Errorf("scanner panic: %v", r)
		}
	}()
	{
		metrics := newScanMetrics(scanType)
		resize.SetExternalProcessObserver(metrics.ExternalProcess)
		defer func() {
			metrics.finish()
			setLastScanMetrics(metrics.snapshot())
		}()

		var unsupportedPaths []string
		progress := newProgressNotifier(c)

		c.Logger.Info("scanning media at " + config.MediaPath(c))
		if err := queries.Notify(c, "Scan started at "+config.MediaPath(c)+".", "scanning"); err != nil {
			c.Logger.Error("failed to notify", "err", err)
		}

		var h *geo.Handlers
		if c.LocationService == "" {
			var err error
			h, err = geo.NewGeoHandler(c)
			if err != nil {
				return "", fmt.Errorf("error getting geo handler %v", err)
			}
		}

		if scanType == "deep" {
			c.Logger.Info("deep scan started")
		}

		if scanType == "metadata" {
			c.Logger.Info("metadata scan started")
		}

		buf := make([]byte, 1024*1024)
		et, err := exiftool.NewExiftool(exiftool.NoPrintConversion(), exiftool.Buffer(buf, 256*1024))
		if err != nil {
			return "", fmt.Errorf("error starting exiftool %v", err)
		}
		metrics.ExternalProcess()
		defer func() {
			if err := et.Close(); err != nil {
				c.Logger.Error("et.Close error", "err", err)
			}
		}()

		// Raw numeric fields and converted lens names require different
		// ExifTool modes. Keep one instance of each alive for this scan so
		// lens extraction does not launch a process for every media item.
		convertedEt, err := exiftool.NewExiftool(exiftool.Buffer(buf, 256*1024))
		if err != nil {
			return "", fmt.Errorf("error starting converted exiftool %v", err)
		}
		metrics.ExternalProcess()
		defer func() {
			if err := convertedEt.Close(); err != nil {
				c.Logger.Error("converted et.Close error", "err", err)
			}
		}()

		// get all current media items. Discovery uses a path map so each
		// filesystem entry is checked in constant time.
		items, err := queries.GetMediaItems(0, "ASC", -1, c)
		if err != nil {
			return "", fmt.Errorf("error getting media items %v", err)
		}

		scanErrors, err := GetScanErrors(c)
		if err != nil {
			c.Logger.Error("error getting scan errors", "error", err)
			scanErrors = make(map[string]time.Time)
		}
		recordScanError := func(path string, scanErr error) {
			scanTime := time.Now()
			scanErrors[path] = scanTime
			if err := TrackScanError(path, scanTime, scanErr, c); err != nil {
				c.Logger.Error("error tracking scan error", "error", err)
			}
		}
		clearScanError := func(path string) {
			if _, ok := scanErrors[path]; !ok {
				return
			}
			if err := ClearScanError(path, c); err != nil {
				c.Logger.Error("error clearing scan error", "error", err)
				return
			}
			delete(scanErrors, path)
		}
		itemsByPath := make(map[string]Media, len(items))
		for _, item := range items {
			itemsByPath[item.Path] = item
		}

		c.Logger.Info("checking for modified and deleted items...")
		for _, item := range items {

			// check for cancellation
			if isCanceled() {
				c.Logger.Info("scan canceled by user")
				if err := queries.Notify(c, "Scan canceled while checking for modified and deleted items.", "canceled"); err != nil {
					c.Logger.Error("Notify error", "err", err)
				}
				return "scan canceled", nil
			}

			// if deleted image exists in db
			statStarted := time.Now()
			file, statErr := os.Stat(filepath.Join(config.MediaPath(c), item.Path))
			metrics.addDuration(&metrics.data.FilesystemStat, time.Since(statStarted))
			metrics.mu.Lock()
			metrics.data.StatCalls++
			metrics.mu.Unlock()
			if statErr != nil && classifyStatError(statErr) == statNotFound {

				// remove if deleted
				err := deleteMediaItem(item.Path, true, item, c, cache)
				if err != nil {
					c.Logger.Error("error removing item", "error", err)
				}

				c.Logger.Info("removed item " + item.Path)
				progress.Notify("Removed item: "+item.Path, "scanning")

			} else if statErr != nil {
				c.Logger.Error("error stating existing media", "path", item.Path, "kind", classifyStatError(statErr), "error", statErr)
			} else {

				// check existing image for modifications
				// recreate thumbnails
				if mediaModified(item, file, c) {

					err = updateMediaItem(ctx, item.Path, true, et, convertedEt, h, c, item, cache, metrics, signatureFromInfo(file))
					if err != nil {
						c.Logger.Error("error updating media item", "error", err)
						if !isCanceled() {
							recordScanError(item.Path, err)
						}
					} else {
						clearScanError(item.Path)
						progress.Notify("Updated media: "+item.Path, "scanning")
					}

				} else if scanType == "deep" {

					// remove and add all items
					// recreate thumbnails
					c.Logger.Info("deepScanning: " + item.Path)
					err = updateMediaItem(ctx, item.Path, true, et, convertedEt, h, c, item, cache, metrics, signatureFromInfo(file))
					if err != nil {
						c.Logger.Error("error updating media item during deep scan", "error", err)
						if !isCanceled() {
							recordScanError(item.Path, err)
						}
					} else {
						clearScanError(item.Path)
						progress.Notify("Updated media: "+item.Path, "scanning")
					}

				} else if scanType == "metadata" {

					// remove and add all items
					// do not recreate thumbnails
					c.Logger.Info("metadataScanning: " + item.Path)
					err = updateMediaItem(ctx, item.Path, false, et, convertedEt, h, c, item, cache, metrics, signatureFromInfo(file))
					if err != nil {
						c.Logger.Error("error updating media item during metadata scan", "error", err)
						if !isCanceled() {
							recordScanError(item.Path, err)
						}
					} else {
						clearScanError(item.Path)
						progress.Notify("Updated media: "+item.Path, "scanning")
					}

				}

			}

		}

		c.Logger.Info("checking for new items...")

		discoveryStarted := time.Now()
		err = filepath.WalkDir(config.MediaPath(c), func(p string, info fs.DirEntry, walkErr error) (callbackErr error) {
			// Add panic recovery for each file processing
			defer func() {
				if r := recover(); r != nil {
					c.Logger.Error("recovered from panic while processing file", "file", p, "panic", r)
					if relativePath, pathErr := filepath.Rel(config.MediaPath(c), p); pathErr == nil && relativePath != "." {
						recordScanError(filepath.ToSlash(relativePath), fmt.Errorf("panic while processing file: %v", r))
					}
					// Continue with next file instead of aborting the whole scan.
					callbackErr = nil
				}
			}()

			if walkErr != nil {
				c.Logger.Error("error walking media directory", "path", p, "kind", classifyStatError(walkErr), "error", walkErr)
				if p == config.MediaPath(c) {
					return fmt.Errorf("error walking media directory %v", walkErr)
				}
				return nil

			}
			if info == nil {
				return nil
			}
			if !info.IsDir() {
				metrics.mu.Lock()
				metrics.data.FilesDiscovered++
				metrics.mu.Unlock()

				// check for cancellation inside file walk
				if isCanceled() {
					c.Logger.Info("scan canceled by user (during walk)")
					return errors.New("scan canceled")
				}

				// remove working dir from path to store a relative ref in db
				relative_path, err := filepath.Rel(config.MediaPath(c), p)
				if err != nil {
					return fmt.Errorf("error making relative path for %s: %w", p, err)
				}
				absolute_path := filepath.Join(config.MediaPath(c), relative_path)

				statStarted := time.Now()
				fileSignature, err := signatureFromDirEntry(info)
				metrics.addDuration(&metrics.data.FilesystemStat, time.Since(statStarted))
				metrics.mu.Lock()
				metrics.data.StatCalls++
				metrics.mu.Unlock()
				if err != nil {
					c.Logger.Error("error getting file information", "path", relative_path, "kind", classifyStatError(err), "error", err)
					return nil
				}

				// skip previously scanned items that had an error
				erroredImage := false
				if lastErrorTime, ok := scanErrors[relative_path]; ok {
					if fileSignature.modTime.Before(lastErrorTime) {
						c.Logger.Info("skipping image " + relative_path + " as it was previously scanned and had an error.")
						erroredImage = true
					}
				}

				if _, exists := itemsByPath[relative_path]; !exists && !erroredImage {

					// check if file is an image
					if isImage(p) {
						err = addImage(ctx, relative_path, absolute_path, fileSignature, false, c.PreGenerateThumb, et, convertedEt, h, c, cache, metrics)
						if err != nil {
							c.Logger.Error("error adding image "+absolute_path, "error", err)
							if !isCanceled() {
								recordScanError(relative_path, err)
							}
						} else {
							clearScanError(relative_path)
							progress.Notify("Added image: "+relative_path, "scanning")
						}

					} else if isVideo(p) {
						err = addVideo(ctx, relative_path, absolute_path, fileSignature, false, c.PreGenerateThumb, et, convertedEt, h, c, cache, metrics)
						if err != nil {
							c.Logger.Error("error adding video "+absolute_path, "error", err)
							if !isCanceled() {
								recordScanError(relative_path, err)
							}
						} else {
							clearScanError(relative_path)
							progress.Notify("Added video: "+relative_path, "scanning")
						}

					} else {
						c.Logger.Info("skipping unsupported file " + relative_path)
						unsupportedPaths = append(unsupportedPaths, relative_path)
						// queries.Notify(c, "Skipping unsupported file: " + relative_path)
					}

				}
			}

			return nil

		})
		metrics.addDuration(&metrics.data.FilesystemDiscovery, time.Since(discoveryStarted))
		metrics.sampleMemory()

		if err != nil {
			if err.Error() == "scan canceled" {
				if err := queries.Notify(c, "Scan canceled while checking for new items.", "canceled"); err != nil {
					c.Logger.Error("Notify error", "err", err)
				}
				return "scan canceled", nil
			}

			return "", fmt.Errorf("error scanning %v", err)
		}

		from := time.Unix(0, 0)
		to := time.Now()
		total, err = queries.GetTotalMediaItems(0, from.Format(time.RFC3339), to.Format(time.RFC3339), "", "", c)
		if err != nil {
			c.Logger.Error("error getting total media items", "error", err)
		}

		unsupportedStatus := ""
		if len(unsupportedPaths) > 0 {
			unsupportedStatus = fmt.Sprintf("%d unsupported items skipped.", len(unsupportedPaths))
		}

		errorsStatus := ""
		if len(scanErrors) > 0 {
			errorsStatus = fmt.Sprintf("%d items with errors occurred during scan.", len(scanErrors))
		}

		status = fmt.Sprintf("Scan complete. %d media items scanned in %s. %s %s", total, time.Since(start).Truncate(time.Second).String(), unsupportedStatus, errorsStatus)

	}

	c.Logger.Info(status)
	time.Sleep(100 * time.Millisecond) // needed for long polling
	if err := queries.Notify(c, status, "complete"); err != nil {
		c.Logger.Error("Notify error", "err", err)
	}

	return status, nil

}

type missingThumb struct {
	Size  int
	Media Media
}

// ThumbScan checks all media items for missing thumbnails and generates any missing ones.
func ThumbScan(c Conf) (string, error) {
	var status string
	if IsScanInProgress() {
		c.Logger.Info("thumbscan already in progress")
		if err := queries.Notify(c, "Thumbscan already in progress.", "scanning"); err != nil {
			c.Logger.Warn("Notify error", "err", err)
		}

		status = "thumbscan already in progress"
	} else {
		start := time.Now()
		if _, claimed := beginScan(); !claimed {
			c.Logger.Info("thumbscan already in progress")
			return "thumbscan already in progress", nil
		}
		defer endScan()
		progress := newProgressNotifier(c)

		c.Logger.Info("scanning thumbs at " + config.CachePath(c))
		// Notify clients immediately that a thumbscan has started
		if err := queries.Notify(c, "Thumbscan started.", "scanning"); err != nil {
			c.Logger.Warn("Notify error", "err", err)
		}

		items, err := queries.GetMediaItems(0, "ASC", -1, c)
		if err != nil {
			return "", fmt.Errorf("error getting media items %v", err)
		}

		var totalItems int
		var missingItems []missingThumb

		for _, item := range items {

			// check for cancellation
			if isCanceled() {
				c.Logger.Info("thumbscan canceled by user")
				if err := queries.Notify(c, "Thumbscan canceled.", "canceled"); err != nil {
					c.Logger.Error("Notify error", "err", err)
				}
				return "thumbscan canceled", nil
			}

			// build a map of sizes for the thumbnail
			var s []int
			final := false

			for _, size := range sizes.GetSizes() {
				if size <= item.Width {

					s = append(s, size)

				} else if !final {

					final = true
					s = append(s, item.Width)

				}
			}

			if len(s) > 0 {
				for _, size := range s {
					thumbPath := filepath.Join(config.CachePath(c), strconv.Itoa(size), strconv.Itoa(int(item.Hash))+".jpg")

					if _, err := os.Stat(thumbPath); errors.Is(err, os.ErrNotExist) {
						missingItems = append(missingItems, missingThumb{Size: int(size), Media: item})
					}

					totalItems++
				}
			}
		}

		status = fmt.Sprintf("Generating %d missing thumbnails...", len(missingItems))
		c.Logger.Info(status)
		if err := queries.Notify(c, status, "scanning"); err != nil {
			c.Logger.Warn("Notify error", "err", err)
		}

		var totalErrors int
		for idx, missing := range missingItems {
			// Rate-limit progress updates so notification work does not dominate
			// large thumbnail scans.
			progress.Notify(fmt.Sprintf("Generating thumbnails: %d/%d", idx, len(missingItems)), "scanning")
			// check for cancellation
			if isCanceled() {
				c.Logger.Info("thumbscan canceled by user")
				if err := queries.Notify(c, "Thumbscan canceled.", "canceled"); err != nil {
					c.Logger.Warn("Notify error", "err", err)
				}
				return "thumbscan canceled", nil
			}

			var wg sync.WaitGroup
			errChan := make(chan error, 1)

			wg.Add(1)
			go resize.AddImageThumb(missing.Media, missing.Size, c, &wg, errChan)

			go func() {
				wg.Wait()
				c.Logger.Info("Thumbnail generated for " + missing.Media.Path + " with size " + strconv.Itoa(missing.Size))
				close(errChan)
			}()

			for err := range errChan {
				if err != nil {
					c.Logger.Error("error generating thumbnail", "error", err)
					totalErrors++
				}
			}
			if isCanceled() {
				c.Logger.Info("thumbscan canceled by user")
				if err := queries.Notify(c, "Thumbscan canceled.", "canceled"); err != nil {
					c.Logger.Warn("Notify error", "err", err)
				}
				return "thumbscan canceled", nil
			}
		}

		status = fmt.Sprintf("Scan complete. %d thumbnails checked in %s. %d missing. %d errors occurred.", totalItems, time.Since(start).Truncate(time.Second).String(), len(missingItems), totalErrors)
		c.Logger.Info(status)
		time.Sleep(100 * time.Millisecond) // needed for long polling
		if err := queries.Notify(c, status, "complete"); err != nil {
			c.Logger.Error("Notify error", "err", err)
		}

	}

	return status, nil

}

// mediaModified tests if a media item has been modified after its addition to the db.
// The caller supplies the successful stat result from the existing-item pass so
// discovery does not stat the same path twice.
func mediaModified(media Media, file os.FileInfo, c Conf) bool {
	if file == nil {
		c.Logger.Error("cannot compare media modification: missing file info", "path", media.Path)
		return false
	}

	return !media.Modified.Equal(file.ModTime())
}

func isImage(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	accepted := []string{".jpg", ".jpeg", ".heic", ".gif", ".png"}
	for _, acceptedExt := range accepted {
		if acceptedExt == ext {
			return true
		}
	}
	return false
}

func isVideo(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	ext := strings.ToLower(filepath.Ext(path))
	accepted := []string{".mp4", ".mov"}
	for _, acceptedExt := range accepted {
		if acceptedExt == ext {
			return true
		}
	}
	return false

}
