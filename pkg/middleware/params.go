package middleware

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/robbymilo/rgallery/pkg/types"
)

type ParamsKey = types.ParamsKey

func Params(c Conf) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			json := false
			if r.URL.Query().Get("format") == "json" || r.Header.Get("Content-Type") == "application/json" {
				json = true
				w.Header().Set("Content-Type", "application/json")
			}

			// check for page
			var page = 1
			if r.URL.Query().Get("page") != "" {
				p, err := strconv.Atoi(r.URL.Query().Get("page"))
				if err != nil {
					c.Logger.Error("error parsing page param", "error", err)
				}
				page = p
			}

			// check for rating
			var rating = 0
			if r.URL.Query().Get("rating") != "" {
				q, err := strconv.Atoi(r.URL.Query().Get("rating"))
				if err != nil {
					c.Logger.Error("error parsing rating param", "error", err)

				}
				rating = q
			}

			// check for proper direction
			direction := "desc"
			if r.URL.Query().Get("direction") != "" {
				direction = strings.ToLower(r.URL.Query().Get("direction"))
			}

			if direction != "asc" && direction != "desc" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			// check camera
			camera := ""
			if r.URL.Query().Get("camera") != "" {
				camera = r.URL.Query().Get("camera")
			}

			// check lens
			lens := ""
			if r.URL.Query().Get("lens") != "" {
				lens = r.URL.Query().Get("lens")
			}

			// check type
			mediatype := ""
			if r.URL.Query().Get("type") == "image" {
				mediatype = "image"
			} else if r.URL.Query().Get("type") == "video" {
				mediatype = "video"
			}

			// check term
			term := ""
			if r.URL.Query().Get("term") != "" {
				term = r.URL.Query().Get("term")
			}

			// check folder/folder
			folder := ""
			if r.URL.Query().Get("folder") != "" {
				folder = r.URL.Query().Get("folder")
			}

			// check subject
			subject := ""
			if r.URL.Query().Get("subject") != "" {
				subject = r.URL.Query().Get("subject")
			} else if r.URL.Query().Get("tag") != "" {
				subject = r.URL.Query().Get("tag")
			}

			// check orderby
			orderby := "date"
			if slices.Contains([]string{"date", "modified"}, r.URL.Query().Get("orderby")) {
				orderby = r.URL.Query().Get("orderby")
			}

			// check software
			software := ""
			if r.URL.Query().Get("software") != "" {
				software = r.URL.Query().Get("software")
			}

			// check focallength35
			var focallength35 float64
			if r.URL.Query().Get("focallength35") != "" {
				if s, err := strconv.ParseFloat(r.URL.Query().Get("focallength35"), 64); err == nil {
					focallength35 = s
				}
			}

			params := FilterParams{
				PageSize:      10,
				Json:          json,
				Page:          page,
				Rating:        rating,
				Direction:     direction,
				Camera:        camera,
				Lens:          lens,
				MediaType:     mediatype,
				Term:          term,
				OrderBy:       orderby,
				Folder:        folder,
				Subject:       subject,
				Software:      software,
				FocalLength35: focallength35,
			}

			ctx := context.WithValue(r.Context(), ParamsKey{}, params)

			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
