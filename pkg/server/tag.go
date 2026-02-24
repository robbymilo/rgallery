package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
)

func ServeTag(w http.ResponseWriter, r *http.Request) {

	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	var pageSize = 100
	offset := (params.Page - 1) * pageSize

	tag, err := DecodeURL(chi.URLParam(r, "slug"))
	if err != nil {
		c.Logger.Error("error decoding slug", "error", err)
	}
	media, err := queries.GetTag(offset, params.Direction, pageSize, "subject", tag, c)
	if err != nil {
		c.Logger.Error("error getting tag", "error", err)
	}
	title, err := queries.GetTagTitle(tag, c)
	if err != nil {
		c.Logger.Error("error getting tag title", "error", err)
	}
	total, err := queries.GetTotalOfTag(tag, c)
	if err != nil {
		c.Logger.Error("error getting total of tag", "error", err)
	}

	if len(media) == 0 {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("404\n"))
		if err != nil {
			c.Logger.Error("error writing 404 response:", "err", err)
		}
		return
	}

	response := ResponseMediaItems{
		MediaItems: media,
		Title:      title,
		Slug:       tag,
		OrderBy:    "date",
		Page:       params.Page,
		PageSize:   pageSize,
		Total:      total,
		Direction:  params.Direction,
		Collection: "tag",
		Section:    "tags",
	}

	err = render.Render(w, r, response, "images")
	if err != nil {
		c.Logger.Error("error rendering tag response", "error", err)
	}
}
