package server

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/resize"
)

// ServeThumbnail passes the thumbnail request from the router to the handler.
func ServeThumbnail(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	var file []byte
	var err error

	// get image hash and size from request
	h, err := DecodeURL(chi.URLParam(r, "hash"))
	if err != nil {
		c.Logger.Error("error decoding hash", "error", err)
	}
	hash := GetHash(h)
	size, err := strconv.Atoi(chi.URLParam(r, "size"))
	if err != nil {
		c.Logger.Error("error reading thumb size", "err", err)
	}

	// get image from the handler
	p := resize.CreateThumbFilePath(hash, size, c)
	modTime := time.Now()
	if fileExists(p) {
		// send saved thumb if it exists
		file, err = os.ReadFile(p)
		if err != nil {
			c.Logger.Error("error reading saved thumb:", "err", err)
		}

		info, err := os.Stat(p)
		if err != nil {
			c.Logger.Error("error stating saved thumb:", "err", err)
		}
		modTime = info.ModTime()
	} else {
		file, err = resize.HandleThumb(hash, size, c)
	}

	// serve a 404 if not found
	if err != nil {
		c.Logger.Error("error loading thumb", "hash", hash, "size", size, "error", err)
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("404\n"))
		if err != nil {
			c.Logger.Error("error writing thumbnail 404 response:", "err", err)
		}
		return
	}

	// serve the image
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	http.ServeContent(w, r, "thumbnail", modTime, bytes.NewReader(file))

}

func fileExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}
