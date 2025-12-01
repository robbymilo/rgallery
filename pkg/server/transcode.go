package server

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/resize"
	"github.com/robbymilo/rgallery/pkg/transcode"
)

// ServeTranscode serves cached transcode files, otherwise serves generated transcode files on demand.
func ServeTranscode(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	var file []byte
	var err error

	h, err := DecodeURL(chi.URLParam(r, "hash"))
	if err != nil {
		c.Logger.Error("error decoding hash", "error", err)
	}
	hash := GetHash(h)
	index_file := transcode.CreateHLSIndexFilePath(hash, c)

	if chi.URLParam(r, "file") == "index.m3u8" {

		// check if index.m3u8 files exist in cache
		if fileExists(index_file) {
			// serve file
			file, err = os.ReadFile(index_file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("503\n"))
				fmt.Println("error reading saved index.m3u8 file:", err)
			}

		} else {

			// get original file path from db
			video, err := queries.GetSingleMediaItem(hash, c)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("503\n"))
				fmt.Println("error getting single media item:", err)
				return
			}

			// Use TranscodeWithLock to ensure only one transcoding process runs at a time
			original := resize.CreateOriginalFilePath(video.Path, c)
			err = transcode.TranscodeWithLock(original, index_file, hash, c)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("503\n"))
				fmt.Println("error transcoding video:", err)
				return
			}

			file, err = os.ReadFile(index_file)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte("503\n"))
				fmt.Println("error reading saved index.m3u8 file:", err)
			}
			w.Header().Set("Content-Type", "audio/mpegurl")
		}

	} else {

		// serve cached file
		file, err = os.ReadFile(transcode.CreateTSFilePath(hash, chi.URLParam(r, "file"), c))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("503\n"))
			fmt.Println("error reading saved TS file:", err)
		}

		w.Header().Set("Content-Type", "application/octet-stream")

	}

	// send to browser
	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")
	http.ServeContent(w, r, "thumbnail", time.Now(), bytes.NewReader(file))
}
