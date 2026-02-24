package server

import (
	"net/http"

	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/render"
)

func ServeGear(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)

	cameras, err := queries.GetGear("camera", c)
	if err != nil {
		c.Logger.Error("error getting cameras", "error", err)
	}
	lenses, err := queries.GetGear("lens", c)
	if err != nil {
		c.Logger.Error("error getting lenses", "error", err)
	}
	fl, err := queries.GetGear("focallength35", c)
	if err != nil {
		c.Logger.Error("error getting focal lengths", "error", err)
	}
	software, err := queries.GetGear("software", c)
	if err != nil {
		c.Logger.Error("error getting softwares", "error", err)
	}

	response := ResponseGear{
		Cameras:       cameras,
		Lenses:        lenses,
		FocalLength35: fl,
		Software:      software,
		Section:       "gear",
		Title:         "Gear stats",
		Meta:          c.Meta,
	}

	err = render.Render(w, r, response, "gear")
	if err != nil {
		c.Logger.Error("error rendering gear response", "error", err)
	}
}
