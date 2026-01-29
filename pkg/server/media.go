package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type ResponseGear = types.ResponseGear
type Conf = types.Conf
type PrevNext = types.PrevNext

func ServeMedia(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	params := r.Context().Value(ParamsKey{}).(FilterParams)

	h, err := DecodeURL(chi.URLParam(r, "hash"))
	if err != nil {
		c.Logger.Error("error decoding hash", "error", err)
	}
	hash := GetHash(h)

	media, err := queries.GetSingleMediaItem(hash, c)
	if err != nil {
		c.Logger.Error("error getting single media item", "error", err)
	}

	if media.Path == "" {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("404\n"))
		if err != nil {
			c.Logger.Error("error writing media 404 response", "error", err)

		}
		return
	}

	previous, err := queries.GetPrevious(media.Date, hash, params, c)
	if err != nil {
		c.Logger.Error("error getting previous media items", "error", err)
	}
	total_next := 6 - len(previous)
	next, err := queries.GetNext(media.Date, hash, total_next, params, previous, c)
	if err != nil {
		c.Logger.Error("error getting next media items", "error", err)
	}

	response := ResponseMedia{
		Media:    media,
		Previous: previous,
		Next:     next,
	}

	err = render.Render(w, r, response, "media")
	if err != nil {
		c.Logger.Error("error rendering media response", "error", err)
	}

}
