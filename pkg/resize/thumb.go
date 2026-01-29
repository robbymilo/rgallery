package resize

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/sizes"
)

// SafeImageOperation executes an image operation function and recovers from panics
func SafeImageOperation(operation func() ([]byte, error)) (file []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			file = nil
			err = fmt.Errorf("panic recovered in image operation: %v", r)
		}
	}()

	return operation()
}

// HandleThumb determines whether to request a thumb from the resize service or check for one on disk.
func HandleThumb(hash uint32, size int, c Conf) ([]byte, error) {
	return SafeImageOperation(func() ([]byte, error) {
		var file []byte
		var err error

		// check that image request is of valid size
		image, err := queries.GetSingleMediaItem(hash, c)
		if err != nil {
			return file, fmt.Errorf("error getting media item: %v", err)
		}
		_, err = sizes.ValidThumbSize(size, image.Width)
		if err != nil {
			return nil, fmt.Errorf("error getting valid thumb size: %v", err)
		}

		// generate a new thumb
		if c.ResizeService != "" {
			file, err = GetThumbFromResizeService(image, size, c)
			if err != nil {
				return nil, fmt.Errorf("error getting thumb from resize service: %v", err)
			}
		} else {
			file, err = CreateThumbFromDisk(hash, size, c)
			if err != nil {
				return nil, fmt.Errorf("error creating thumb from disk: %v", err)
			}
		}

		return file, nil
	})
}

// CreateThumbFromDisk checks for an existing thumbnail in the cache directory. If it does not exist, a new thumbnail is created and saved.
func CreateThumbFromDisk(hash uint32, size int, c Conf) ([]byte, error) {
	var file []byte
	var err error

	// Add panic recovery that returns the error
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in CreateThumbFromDisk: %v\n", r)
			// Convert panic to error
			err = fmt.Errorf("thumbnail generation failed with panic: %v", r)
			file = nil
		}
	}()

	// create a new thumb
	image, err := queries.GetSingleMediaItem(hash, c)
	if err != nil {
		return file, fmt.Errorf("error getting media item: %v", err)
	}
	p := CreateOriginalFilePath(image.Path, c)
	file, err = GenerateSingleThumb(p, image, size, c)
	if err != nil {
		return file, fmt.Errorf("error generating single thumb: %v", err)
	}

	return file, err
}

// GetThumbFromResizeService requests a thumbnail from the resize service, serves the response, and then saves it to disk.
func GetThumbFromResizeService(media Media, size int, c Conf) ([]byte, error) {
	var err error
	var file []byte

	// Add panic recovery that returns the error
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in GetThumbFromResizeService: %v\n", r)
			// Convert panic to error
			err = fmt.Errorf("thumbnail generation failed with panic: %v", r)
			file = nil
		}
	}()

	if media.Path == "" {
		err = errors.New("media path not found")
		return nil, err
	} else {
		path := CreateOriginalFilePath(media.Path, c)
		url := fmt.Sprintf("%s/?size=%d&quality=%d", c.ResizeService, size, c.Quality)

		if strings.ToLower(filepath.Ext(path)) == ".heic" {
			url = fmt.Sprintf("%s&format=%s", url, "heic")
		}

		var res *http.Response

		if media.Type == "image" {
			res, err = uploadFileMultipart(url, path)
			if err != nil {
				return nil, fmt.Errorf("error requesting image from resize service: %v", err)
			}
			defer func() {
				if err := res.Body.Close(); err != nil {
					c.Logger.Error("res.Body.Close error:", "err", err)
				}
			}()
		} else if media.Type == "video" {
			file, err := CreateVideoThumb(path)
			if err != nil {
				return nil, fmt.Errorf("error getting video thumb from video: %v", err)
			}

			res, err = uploadReaderFileMultipart(url, path, file)
			if err != nil {
				return nil, fmt.Errorf("error requesting video thumb from resize service: %v", err)
			}
			defer func() {
				if err := res.Body.Close(); err != nil {
					c.Logger.Error("res.Body.Close error:", "err", err)
				}
			}()
		}

		if res.StatusCode == http.StatusOK {
			file, err = io.ReadAll(res.Body)
			if err != nil {
				return nil, fmt.Errorf("error reading response body: %v", err)
			}

			img, err := imaging.Decode(bytes.NewReader(file))
			if err != nil {
				return nil, fmt.Errorf("error decoding image from resize service: %v", err)
			}

			err = SaveImageToDisk(filepath.Join(config.CachePath(c), fmt.Sprint(size), fmt.Sprint(media.Hash)+".jpg"), img, c)
			if err != nil {
				return nil, fmt.Errorf("error saving image from resize service: %v", err)
			}
		} else {
			responseBodyBytes, readErr := io.ReadAll(res.Body)
			if readErr != nil {
				err = fmt.Errorf("resize service returned status code %d and failed to read response body: %v", res.StatusCode, readErr)
			} else {
				err = fmt.Errorf("resize service returned status code %d: %s", res.StatusCode, string(responseBodyBytes))
			}
			return nil, err
		}
	}

	return file, err
}

// createThumbFilePath creates a string of the path where the thumb will live.
func CreateThumbFilePath(hash uint32, size int, c Conf) string {
	return filepath.Join(config.CachePath(c), fmt.Sprint(size), fmt.Sprint(hash)+".jpg")
}

// CreateOriginalFilePath creates a string of the absolute path of the original media.
func CreateOriginalFilePath(path string, c Conf) string {
	return filepath.Join(config.MediaPath(c), path)
}

// uploadFileMultipart opens a file to send to the resizer.
// source: https://gist.github.com/mattetti/5914158?permalink_comment_id=3422260#gistcomment-3422260
func uploadFileMultipart(url, path string) (*http.Response, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println("f.Close error:", err)
		}
	}()

	resp, err := uploadReaderFileMultipart(url, path, f)
	return resp, err
}

// uploadReaderFileMultipart posts a file to a URL and returns the response.
func uploadReaderFileMultipart(url, path string, f io.Reader) (*http.Response, error) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic in uploadReaderFileMultipart: %v\n", r)
			// We can't modify the return values here since they're not named,
			// but we can at least prevent the panic from crashing the program
		}
	}()

	// Reduce number of syscalls when reading from disk.
	bufferedFileReader := bufio.NewReader(f)

	// Create a pipe for writing from the file and reading to
	// the request concurrently.
	bodyReader, bodyWriter := io.Pipe()
	formWriter := multipart.NewWriter(bodyWriter)

	// Store the first write error in writeErr.
	var (
		writeErr error
		errOnce  sync.Once
	)
	setErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { writeErr = err })
		}
	}
	go func() {
		// Add panic recovery in goroutine
		defer func() {
			if r := recover(); r != nil {
				errOnce.Do(func() {
					writeErr = fmt.Errorf("panic in file upload goroutine: %v", r)
				})
				// Make sure to close writers even in case of panic
				_ = formWriter.Close()
				_ = bodyWriter.Close()
			}
		}()

		partWriter, err := formWriter.CreateFormFile("file", path)
		setErr(err)
		_, err = io.Copy(partWriter, bufferedFileReader)
		setErr(err)
		setErr(formWriter.Close())
		setErr(bodyWriter.Close())
	}()

	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", formWriter.FormDataContentType())

	// This operation will block until both the formWriter
	// and bodyWriter have been closed by the goroutine,
	// or in the event of a HTTP error.
	resp, err := http.DefaultClient.Do(req)

	if writeErr != nil {
		return nil, writeErr
	}

	return resp, err
}

// CreateTempDir creates a temporary directory and returns its path
func CreateTempDir() (string, error) {
	tempDir, err := os.MkdirTemp("", "temp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	return tempDir, nil
}
