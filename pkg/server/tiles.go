package server

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/tilesets"
	mbtiles "github.com/twpayne/go-mbtiles"
)

func ServeTiles(w http.ResponseWriter, r *http.Request, c Conf) {

	z, err := strconv.Atoi(chi.URLParam(r, "z"))
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("invalid zoom level", "error", err)
		return
	}
	x, err := strconv.Atoi(chi.URLParam(r, "x"))
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("invalid X coordinate", "error", err)
		return
	}
	y, err := strconv.Atoi(chi.URLParam(r, "y"))
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("error getting embedded file content", "error", err)
		return
	}

	// get embedded mbtiles file contents
	content, err := tilesets.ExposeEmbeddedFile()
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("error getting embedded file content", "error", err)
		return

	}

	// save mbtiles content to a temp file
	tmpFile, err := os.CreateTemp("", "world-*.mbtiles")
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("error creating temporary file", "error", err)
		return

	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			c.Logger.Error("tmpFile.Close error", "err", err)
		}
	}()

	if _, err := tmpFile.Write(content); err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("error writing to temporary file", "error", err)
		return

	}

	// pass the temp file name to mbtiles
	reader, err := mbtiles.NewReader("sqlite", tmpFile.Name())
	if err != nil {
		http.Error(w, "server error", http.StatusBadRequest)
		c.Logger.Error("error opening tile db", "error", err)
	}
	tile, err := reader.SelectTile(z, x, y)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "server error", http.StatusBadRequest)
			c.Logger.Error("tile does not exist", "error", err)
			return
		} else {
			http.Error(w, "server error", http.StatusBadRequest)
			c.Logger.Error("error getting tile", "error", err)

		}
	}

	// Set the appropriate headers for PNG
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(tile)
	if err != nil {
		c.Logger.Error("error writing tile", "error", err)
	}
}
