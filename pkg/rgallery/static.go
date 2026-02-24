package rgallery

import (
	"bytes"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/dist"
	"github.com/robbymilo/rgallery/pkg/fonts"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/static"
)

func Dist(h http.Handler, c Conf) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/dist/")

		content, err := dist.DistDir.ReadFile(path)
		if err != nil {
			c.Logger.Error("error reading file", "error", err)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		e := render.GenerateEtag(string(content))
		w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Static(h http.Handler, c Conf) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		content, err := static.StaticDir.ReadFile(strings.TrimPrefix(r.URL.Path, "/static/"))
		if err != nil {
			c.Logger.Error("error reading file", "error", err)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		e := render.GenerateEtag(string(content))
		w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Fonts(h http.Handler, c Conf) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		content, err := fonts.FontDir.ReadFile(strings.TrimPrefix(r.URL.Path, "/fonts/"))
		if err != nil {
			c.Logger.Error("error reading file", "error", err)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		e := render.GenerateEtag(string(content))
		w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// SpaHandler serves the SPA, handling assets, specific index files, and the root fallback.
func SpaHandler(fileSystem fs.FS, c Conf) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := chi.URLParam(r, "*")
		if path == "" {
			path = "index.html"
		}

		// Try to read the exact file requested, fallback to index.html for SPA routing
		content, err := fs.ReadFile(fileSystem, path)
		if err != nil {
			path = "index.html"
			content, err = fs.ReadFile(fileSystem, path)
			if err != nil {
				c.Logger.Error("critical: index.html missing from embed", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		// Set content type
		if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
			w.Header().Set("Content-Type", ct)
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		// Set ETag and cache headers
		e := render.GenerateEtag(string(content))
		w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")

		// Handle 304 Not Modified
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, e) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(content))
	}
}
