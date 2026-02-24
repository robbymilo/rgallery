package server

import (
	"net/http"
	"strconv"

	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type ResponseFolders = types.ResponseFolders
type Folder = types.Folder
type Directory = types.Directory

func ServeFolders(w http.ResponseWriter, r *http.Request) {

	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	var pageSize = -1
	if r.URL.Query().Get("pageSize") != "" {
		q, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if err != nil {
			c.Logger.Error("error parsing pageSize param", "error", err)
		}
		pageSize = q
	}

	folders, err := queries.GetFolders(params, "folder", pageSize, 0, c)
	if err != nil {
		c.Logger.Error("error getting folders", "error", err)
	}

	total, err := queries.GetTotalFolders(c)
	if err != nil {
		c.Logger.Error("error getting total of folders", "error", err)
	}

	response := ResponseFolders{
		Folders:    folders,
		Title:      "Folders",
		OrderBy:    "date",
		Page:       params.Page,
		PageSize:   pageSize,
		Total:      total,
		Direction:  params.Direction,
		Collection: "folder",
		Section:    "folders",
		Meta:       c.Meta,
	}

	err = render.Render(w, r, response, "folders")
	if err != nil {
		c.Logger.Error("error rendering folders response", "error", err)
	}
}
