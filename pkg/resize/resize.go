package resize

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/disintegration/imageorient"
	"github.com/disintegration/imaging"
	"github.com/robbymilo/rgallery/pkg/config"
	"github.com/robbymilo/rgallery/pkg/sizes"

	"github.com/robbymilo/rgallery/pkg/types"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Conf = types.Conf
type Media = types.Media

// GenerateSingleThumb serves a generated thumbnail from the original media.
func GenerateSingleThumb(path string, media Media, size int, c Conf) ([]byte, error) {
	var file []byte
	var err error

	// do not create if request is larger than the original image
	validSize, err := sizes.ValidThumbSize(size, media.Width)
	if err != nil {
		return nil, err
	}

	if validSize {
		switch media.Type {
		case "video":
			file, err = CreateSaveVideoThumb(path, media, size, c)
			if err != nil {
				return nil, fmt.Errorf("error creating and saving video thumb: %v", err)
			}
		case "image":
			file, err = CreateSaveImageThumb(path, media, size, c)
			if err != nil {
				return nil, fmt.Errorf("error creating and saving image thumb: %v", err)
			}
		default:
			return nil, fmt.Errorf("file is not a video or image")
		}
	} else {
		return nil, fmt.Errorf("thumb request is out of range")
	}

	return file, err
}

// CreateSaveImageThumb opens the original file for image media types.
func CreateSaveImageThumb(path string, media Media, size int, c Conf) ([]byte, error) {
	var file []byte

	path, err := saveTempHeic(path)
	if err != nil {
		return nil, fmt.Errorf("error converting heic to temp jpg file: %v", err)
	}

	img, err := imaging.Open(path, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("error decoding original image: %v", err)
	}

	// if media.Rotation != 0 {
	// 	img = imaging.Rotate(img, media.Rotation, color.Transparent)
	// }

	// resize and save image
	if c.ResizeService != "" {
		file, err = GetThumbFromResizeService(media, size, c)
		if err != nil {
			return nil, fmt.Errorf("error getting thumb from resizer: %v", err)
		}

		// remove the temp file
		if strings.Contains(strings.ToLower(path), ".heic") {

			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("file does not exist: %v", err)
			}

			err := os.Remove(path)
			if err != nil {
				return nil, fmt.Errorf("error removing temp heic: %v", err)
			}
		}

	} else {
		err := imageToThumb(img, media, size, c)
		if err != nil {
			return nil, fmt.Errorf("error generating thumb from image: %v", err)
		}

		// load saved file
		file, err = os.ReadFile(filepath.Join(config.CachePath(c), fmt.Sprint(size), fmt.Sprint(media.Hash)+".jpg"))
		if err != nil {
			return nil, fmt.Errorf("error loading generated thumb: %v", err)
		}

		// remove the temp file
		if strings.Contains(strings.ToLower(path), ".heic") {

			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("file does not exist: %v", err)
			}

			err := os.Remove(path)
			if err != nil {
				return nil, fmt.Errorf("error removing temp heic: %v", err)
			}
		}

	}

	c.Logger.Info("resized image", "path", path, "size", size, "hash", media.Hash)

	return file, nil
}

// CreateVideoThumb creates a thumbnail for video media types.
func CreateVideoThumb(path string) (io.Reader, error) {
	// get frame from the video to use as thumbnail
	video, err := GetFFMPEGThumb(path, 1)
	if err != nil {
		return nil, err
	}

	return video, nil
}

// CreateSaveVideoThumb saves and returns a thumbnail for video media types.
func CreateSaveVideoThumb(path string, media Media, size int, c Conf) ([]byte, error) {

	video, err := CreateVideoThumb(path)
	if err != nil {
		return nil, fmt.Errorf("error getting thumb from video: %v", err)
	}

	// convert frame to image
	img, err := imaging.Decode(video)
	if err != nil {
		return nil, fmt.Errorf("error decoding video thumb: %v", err)
	}

	// resize and save image
	err = imageToThumb(img, media, size, c)
	if err != nil {
		return nil, fmt.Errorf("error converting thumb from image: %v", err)
	}

	// load saved file
	file, err := os.ReadFile(filepath.Join(config.CachePath(c), fmt.Sprint(size), fmt.Sprint(media.Hash)+".jpg"))
	if err != nil {
		return nil, fmt.Errorf("error loading generated thumb: %v", err)
	}

	return file, nil
}

// GetFFMPEGThumb serves a frame from a video.
func GetFFMPEGThumb(video string, frameNum int) (io.Reader, error) {

	buf := bytes.NewBuffer(nil)
	err := ffmpeg.Input(video).
		Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", frameNum)}).
		Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2", "vcodec": "mjpeg"}).
		WithOutput(buf).
		Run()
	return buf, err
}

// convert image to file
func imageToThumb(img image.Image, media Media, size int, c Conf) error {

	// resize the image
	img = imaging.Resize(img, size, 0, imaging.Lanczos)

	// AutoOrientation does not work with heic https://github.com/disintegration/imaging/issues/157
	// if media.Rotation != 0 {
	// 	img = imaging.Rotate(img, media.Rotation, color.Transparent)
	// }

	// build dir string
	dir := filepath.Join(config.CachePath(c), fmt.Sprint(size))

	// create the thumb dir
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error making cache dir: %v", err)
	}

	// save the image
	err = imaging.Save(img, filepath.Join(dir, fmt.Sprint(media.Hash)+".jpg"), imaging.JPEGQuality(c.Quality))
	if err != nil {
		return fmt.Errorf("error saving resized image: %v", err)
	}

	return nil

}

// SaveImageToDisk saves an image to disk at a specified location after confirming the dir exists.
func SaveImageToDisk(path string, image image.Image, c Conf) error {

	// create dir if it does not exist
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return fmt.Errorf("error making dir: %v", err)
	}

	// save the image to disk
	err = imaging.Save(image, filepath.Join(path), imaging.JPEGQuality(c.Quality))
	if err != nil {
		return fmt.Errorf("failed to save thumbnail: %v", err)
	}
	return nil
}

// saveTempHeic converts a .heic file to .jpg and saves it to /tmp/rgallery for usage.
func saveTempHeic(path string) (string, error) {
	if strings.ToLower(filepath.Ext(path)) == ".heic" {

		// make temp dir
		tmpDir, err := os.MkdirTemp("", "rgallery_temp_*")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				fmt.Printf("os.RemoveAll error: %v\n", err)
			}
		}()

		tmpFile := tmpDir + filepath.Base(path) + ".jpg"
		cmd := exec.Command("vips", "resize", path, tmpFile, "1")
		_, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("error calling vips command: %v", err)
		}

		path = tmpFile
	}
	return path, nil
}

func DecodeImage(path string) (image.Image, error) {

	var img image.Image
	path, err := saveTempHeic(path)
	if err != nil {
		return nil, fmt.Errorf("error converting heic to temp jpg file: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("os.Open failed: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("file.Close error: %v", err)
		}
	}()

	img, _, err = imageorient.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("imageorient.Decode failed: %v", err)
	}

	// remove the temp file
	if strings.Contains(strings.ToLower(path), ".heic") {

		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file does not exist: %v", err)
		}

		err := os.Remove(path)
		if err != nil {
			return nil, fmt.Errorf("error removing temp heic: %v", err)
		}
	}

	return img, nil
}
