package resize

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
)

// ResizeImageUpload accepts an image via POST, resizes it based on the size query param, and serves the resulting image.
func ResizeImageUpload(w http.ResponseWriter, r *http.Request, c Conf) error {

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		return fmt.Errorf("error parsing multipart form: %v", err)
	}
	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		return fmt.Errorf("error getting size: %v", err)
	}

	quality, err := strconv.Atoi(r.URL.Query().Get("quality"))
	if err != nil {
		return fmt.Errorf("error getting quality: %v", err)
	}

	format := r.URL.Query().Get("format")

	// get the uploaded file
	f := r.MultipartForm.File["file"]
	file, err := f[0].Open()
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			c.Logger.Error("file.Close error:", "err", err)
		}
	}()

	imgByte, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error parsing image: %v", err)
	}

	var temp_heic string
	var temp_jpg string
	if format == "heic" {

		// make temp dir
		tmpDir, err := os.MkdirTemp("", "rgallery_temp_*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				c.Logger.Error("os.RemoveAll error:", "err", err)
			}
		}()

		// write temp heic file
		temp_id := uuid.NewString()
		temp_heic = tmpDir + temp_id + ".heic"
		err = os.WriteFile(temp_heic, imgByte, 0744)
		if err != nil {
			return fmt.Errorf("error writing temp heic file: %v", err)
		}

		// convert temp heic file to temp jpg file
		temp_jpg = tmpDir + temp_id + ".jpg"
		cmd := exec.Command("vips", "resize", temp_heic, temp_jpg, "1")
		_, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error calling vips command: %v", err)
		}

		// open the new temp jpg
		file, err = os.Open(temp_jpg)
		if err != nil {
			return fmt.Errorf("error opening temp jpg: %v", err)
		}
		defer func() {
			if err := file.Close(); err != nil {
				c.Logger.Error("file.Close error:", "err", err)
			}
		}()

		// read the temp jpg
		imgByte, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("error parsing image: %v", err)
		}

	}

	// decode the image
	var img image.Image
	img, err = imaging.Decode(bytes.NewReader(imgByte), imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	// resize the original image
	resized := imaging.Resize(img, size, 0, imaging.Lanczos)

	// serve the resized image
	w.Header().Set("Content-Type", "image/jpeg")
	err = jpeg.Encode(w, resized, &jpeg.Options{Quality: quality})
	if err != nil {
		return fmt.Errorf("error encoding jpeg: %v", err)
	}

	if temp_jpg != "" {
		err := os.Remove(temp_jpg)
		if err != nil {
			return fmt.Errorf("error removing temp jpeg: %v", err)
		}
	}

	if temp_heic != "" {
		err := os.Remove(temp_heic)
		if err != nil {
			return fmt.Errorf("error removing temp heic: %v", err)
		}
	}

	return nil
}
