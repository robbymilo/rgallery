package server

import (
	"net/http"

	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type ResponseMap = types.ResponseMap

func ServeMap(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)

	mapItems, err := queries.GetMapItems(c)
	if err != nil {
		c.Logger.Error("error getting map items", "error", err)
	}

	response := ResponseMap{
		Section:    "map",
		MapItems:   mapItems,
		TileServer: c.TileServer,
		Meta:       c.Meta,
	}

	err = render.RenderJson(w, r, response)
	if err != nil {
		c.Logger.Error("error rendering map response", "error", err)
	}
}
