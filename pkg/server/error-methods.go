package server

import (
	"net/http"

	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
)

type ResponseNotFound = types.ResponseNotFound
type Meta = types.Meta

func NotFound(w http.ResponseWriter, r *http.Request) {

	response := ResponseNotFound{
		Message: "Not found.",
		Title:   "404",
		Section: "errors",
	}

	w.WriteHeader(http.StatusNotFound)
	_ = render.Render(w, r, response, "errors")

}

func NotAllowed(w http.ResponseWriter, r *http.Request) {

	response := ResponseNotFound{
		Message: "Not allowed.",
		Title:   "500",
		Section: "errors",
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = render.Render(w, r, response, "errors")

}

// Send404 explicitly sends a 404 Not Found response
func Send404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	NotFound(w, r)
}
