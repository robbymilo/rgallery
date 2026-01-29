package server

import (
	"net/http"

	"github.com/robbymilo/rgallery/pkg/render"
)

func ServeProfile(w http.ResponseWriter, r *http.Request, c Conf) {
	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	response := ResponseProfile(user)

	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")

	err := render.RenderJson(w, r, response)
	if err != nil {
		c.Logger.Error("error rendering admin response", "error", err)
	}
}
