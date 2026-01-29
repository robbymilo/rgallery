package render

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/robbymilo/rgallery/pkg/types"
)

type FilterParams = types.FilterParams
type Conf = types.Conf
type ConfigKey = types.ConfigKey
type ParamsKey = types.ParamsKey
type UserKey = types.UserKey

// Render coordinates the sending of raw JSON, an embedded template, or a local template to the response.
func Render(w http.ResponseWriter, r *http.Request, response interface{}, layout string) error {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	e := setEtag(r.URL, response, user, params)
	w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))

	err := RenderJson(w, r, response)
	if err != nil {
		return err
	}

	return nil

}

// RenderJson sends raw JSON to the request.
func RenderJson(w http.ResponseWriter, r *http.Request, response interface{}) error {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	e := setEtag(r.URL, response, user, params)
	w.Header().Set("Etag", fmt.Sprintf("\"%s\"", e))
	w.Header().Set("Content-Type", "application/json")

	json, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error rendering json response: %v", err)
	}
	_, err = w.Write(json)
	if err != nil {
		return fmt.Errorf("error writing json response: %v", err)

	}

	return nil
}
