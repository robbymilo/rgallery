package server

import (
	"net/http"
	"strings"

	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type ResponseMediaItems = types.ResponseMediaItems
type ConfigKey = types.ConfigKey
type ParamsKey = types.ParamsKey

func ServeFolder(w http.ResponseWriter, r *http.Request) {

	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	page := params.Page
	var pageSize = 100
	offset := (page - 1) * pageSize

	folder, err := DecodeURL(strings.TrimPrefix(r.URL.Path, "/folder/"))
	if err != nil {
		c.Logger.Error("error decoding slug", "error", err)
	}
	if folder == "root" {
		folder = "."
	}
	media, err := queries.GetFolder("folder", folder, pageSize, offset, params, c)
	if err != nil {
		c.Logger.Error("error getting folder", "error", err)
	}

	total, err := queries.GetTotalOfFolder("folder", folder, c)
	if err != nil {
		c.Logger.Error("error getting total of folder", "error", err)
	}

	if len(media) == 0 {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("404\n"))
		if err != nil {
			c.Logger.Error("error writing folder 404 response", "error", err)
		}
		return
	}

	response := ResponseMediaItems{
		Title:      folder,
		Slug:       folder,
		MediaItems: media,
		OrderBy:    "date",
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		Direction:  params.Direction,
		Collection: "folder",
		Section:    "folders",
		Meta:       c.Meta,
	}

	err = render.Render(w, r, response, "images")
	if err != nil {
		c.Logger.Error("error rendering folder response", "error", err)
	}
}
