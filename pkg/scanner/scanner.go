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
func Scan(scanType string, c Conf, cache *cache.Cache) (string, error) {
	// Defer a panic recovery function
	defer func() {
		if r := recover(); r != nil {
			c.Logger.Error("recovered from panic in scanner", "panic", r)
			queries.Notify(c, "Scan encountered an error but will continue.", "scanning")
			// Ensure scan lock is released even after panic
			SetScanInProgress(false)

		}
	}()

	start := time.Now()
	total := 0
	var status string

	// Check if a scan is already in progress using our global variable
	if IsScanInProgress() {
		c.Logger.Info("scan already in progress")
		queries.Notify(c, "Scan already in progress.", "scanning")

		return "scan already in progress", nil
	} else {
		if scanType == "thumb" {
			return "", nil
		}
		// Set scan in progress to true
		SetScanInProgress(true)

		// create a cancel channel for this scan
		resetCancelChan(make(chan struct{}))

		var unsupportedPaths []string

		c.Logger.Info("scanning media at " + config.MediaPath(c))
		queries.Notify(c, "Scan started at "+config.MediaPath(c)+".", "scanning")

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
		defer et.Close()

		// get all current media items
		items, err := queries.GetMediaItems(0, "ASC", -1, c)
		if err != nil {
			return "", fmt.Errorf("error getting media items %v", err)
		}

		scanErrors, err := GetScanErrors(c)
		if err != nil {
			c.Logger.Error("error getting scan errors", "error", err)
		}

		c.Logger.Info("checking for modified and deleted items...")
		for _, item := range items {

			// check for cancellation
			if isCanceled() {
				c.Logger.Info("scan canceled by user")
				queries.Notify(c, "Scan canceled while checking for modified and deleted items.", "canceled")
				SetScanInProgress(false)
				// reset cancel channel
				resetCancelChan(nil)
				return "scan canceled", nil
			}

			// if deleted image exists in db
			if _, err := os.Stat(filepath.Join(config.MediaPath(c), item.Path)); errors.Is(err, os.ErrNotExist) {

				// remove if deleted
				err := deleteMediaItem(item.Path, true, item, c, cache)
				if err != nil {
					c.Logger.Error("error removing item", "error", err)
				}

				c.Logger.Info("removed item " + item.Path)
				queries.Notify(c, "Removed item: "+item.Path, "scanning")

			} else {

				// check existing image for modifications
				// recreate thumbnails
				if mediaModified(item, c) {

					err = updateMediaItem(item.Path, true, et, h, c, item, cache)
					if err != nil {
						c.Logger.Error("error updating media item", "error", err)
					} else {
						queries.Notify(c, "Updated media: "+item.Path, "scanning")
					}

				} else if scanType == "deep" {

					// remove and add all items
					// recreate thumbnails
					c.Logger.Info("deepScanning: " + item.Path)
					err = updateMediaItem(item.Path, true, et, h, c, item, cache)
					if err != nil {
						c.Logger.Error("error updating media item during deep scan", "error", err)
					} else {
						queries.Notify(c, "Updated media: "+item.Path, "scanning")
					}

				} else if scanType == "metadata" {

					// remove and add all items
					// do not recreate thumbnails
					c.Logger.Info("metadataScanning: " + item.Path)
					err = updateMediaItem(item.Path, false, et, h, c, item, cache)
					if err != nil {
						c.Logger.Error("error updating media item during metadata scan", "error", err)
					} else {
						queries.Notify(c, "Updated media: "+item.Path, "scanning")
					}

				}

			}

		}

		c.Logger.Info("checking for new items...")

		err = filepath.WalkDir(config.MediaPath(c), func(p string, info fs.DirEntry, err error) error {
			// Add panic recovery for each file processing
			defer func() {
				if r := recover(); r != nil {
					c.Logger.Error("recovered from panic while processing file", "file", p, "panic", r)
					// Continue with next file instead of aborting the whole scan
					err = nil
				}
			}()

			if err != nil {
				return fmt.Errorf("error walking media directory %v", err)

			}
			if !info.IsDir() {

				// check for cancellation inside file walk
				if isCanceled() {
					c.Logger.Info("scan canceled by user (during walk)")
					return errors.New("scan canceled")
				}

				// remove working dir from path to store a relative ref in db
				relative_path := strings.Replace(p, config.MediaPath(c)+"/", "", 1)
				absolute_path := filepath.Join(config.MediaPath(c), relative_path)

				// check if image exists in db
				file, err := os.Stat(absolute_path)
				if err != nil {
					c.Logger.Error("error stating file:", "error", err)
				}

				// skip previously scanned items that had an error
				erroredImage := false
				if lastErrorTime, ok := scanErrors[relative_path]; ok {
					if file.ModTime().Before(lastErrorTime) {
						c.Logger.Info("skipping image " + relative_path + " as it was previously scanned and had an error.")
						erroredImage = true
					}
				}

				if !mediaExists(items, relative_path) && !erroredImage {

					// check if file is an image
					if isImage(p) {
						err = addImage(relative_path, absolute_path, false, c.PreGenerateThumb, et, h, c, cache)
						if err != nil {
							c.Logger.Error("error adding image "+absolute_path, "error", err)
							err := TrackScanError(relative_path, time.Now(), err, c)
							if err != nil {
								c.Logger.Error("error tracking scan error", "error", err)
							}
						} else {
							queries.Notify(c, "Added image: "+relative_path, "scanning")
						}

					} else if isVideo(p) {
						err = addVideo(relative_path, absolute_path, false, c.PreGenerateThumb, et, h, c, cache)
						if err != nil {
							c.Logger.Error("error adding video "+absolute_path, "error", err)
						} else {
							queries.Notify(c, "Added video: "+relative_path, "scanning")
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

		if err != nil {
			if err.Error() == "scan canceled" {
				SetScanInProgress(false)
				resetCancelChan(nil)
				queries.Notify(c, "Scan canceled while checking for new items.", "canceled")
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

		// Reset our global scan in progress flag
		SetScanInProgress(false)
		// cleanup cancel channel
		resetCancelChan(nil)

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
	queries.Notify(c, status, "complete")

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
		queries.Notify(c, "Thumbscan already in progress.", "scanning")

		status = "thumbscan already in progress"
	} else {
		start := time.Now()
		SetScanInProgress(true)

		// create a cancel channel for this scan
		resetCancelChan(make(chan struct{}))

		c.Logger.Info("scanning thumbs at " + config.CachePath(c))
		// Notify clients immediately that a thumbscan has started
		queries.Notify(c, "Thumbscan started.", "scanning")

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
				queries.Notify(c, "Thumbscan canceled.", "canceled")
				SetScanInProgress(false)
				resetCancelChan(nil)
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
		queries.Notify(c, status, "scanning")

		var totalErrors int
		for idx, missing := range missingItems {
			// periodically notify progress so clients receive updates
			if idx%10 == 0 {
				queries.Notify(c, fmt.Sprintf("Generating thumbnails: %d/%d", idx, len(missingItems)), "scanning")
			}
			// check for cancellation
			if isCanceled() {
				c.Logger.Info("thumbscan canceled by user")
				queries.Notify(c, "Thumbscan canceled.", "canceled")
				SetScanInProgress(false)
				resetCancelChan(nil)
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
		}

		status = fmt.Sprintf("Scan complete. %d thumbnails checked in %s. %d missing. %d errors occurred.", totalItems, time.Since(start).Truncate(time.Second).String(), len(missingItems), totalErrors)
		c.Logger.Info(status)
		time.Sleep(100 * time.Millisecond) // needed for long polling
		queries.Notify(c, status, "complete")

		SetScanInProgress(false)
		// cleanup cancel channel
		resetCancelChan(nil)

	}

	return status, nil

}

// mediaExists tests if a media item exists in the db.
func mediaExists(media []Media, path string) bool {
	for _, item := range media {
		if item.Path == path {
			return true
		}
	}
	return false
}

// mediaModified tests if a media item has been modified after its addition to the db.
func mediaModified(media Media, c Conf) bool {

	media_path := config.MediaPath(c)
	file, err := os.Stat(filepath.Join(media_path, media.Path))
	if err != nil {
		fmt.Println("error stating file:", err)
	}

	db_modified := media.Modified.Format("2006-01-02T15:04:05")
	file_modified := file.ModTime().UTC().Format("2006-01-02T15:04:05")

	return db_modified != file_modified
}

func isImage(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	accepted := []string{".jpg", ".jpeg", ".heic", ".gif", ".png"}
	for _, ext := range accepted {
		if strings.HasSuffix(path, ext) || strings.HasSuffix(path, strings.ToUpper(ext)) && !strings.HasPrefix(path, ".") {
			return true
		}
	}
	return false
}

func isVideo(path string) bool {
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	accepted := []string{".mp4", ".mov"}
	for _, ext := range accepted {
		if strings.HasSuffix(path, ext) || strings.HasSuffix(path, strings.ToUpper(ext)) && !strings.HasPrefix(path, ".") {
			return true
		}
	}
	return false

}
