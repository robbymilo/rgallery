package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/robbymilo/rgallery/pkg/users"
)

// CreateKey handles a post request to create a new api key.
func CreateKey(w http.ResponseWriter, r *http.Request) {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	creds := &ApiCredentials{}
	if params.Json {
		if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
			c.Logger.Error("error decoding json", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			c.Logger.Error("error parsing form", "error", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		if r.Form.Get("name") == "" {
			http.Error(w, "Missing required field: name", http.StatusBadRequest)
			return
		}
		creds.Name = r.Form.Get("name")
	}

	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	if user.UserRole == "viewer" {
		c.Logger.Error("error adding key", "error", fmt.Errorf("api key cannot be added by viewers"))
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	key, err := users.AddKey(creds, c)
	if err != nil {
		c.Logger.Error("error adding key", "error", err)
		http.Error(w, "Error adding key", http.StatusInternalServerError)
		return
	}

	c.Logger.Info("api key added: " + creds.Name)

	resp := struct {
		Key string `json:"key"`
	}{
		Key: key,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		c.Logger.Error("error writing success response", "error", err)
	}
}

// RemoveKey handles a post request to remove an API key.
func RemoveKey(w http.ResponseWriter, r *http.Request) {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	creds := &ApiCredentials{}
	if params.Json {
		if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
			c.Logger.Error("error decoding json", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			c.Logger.Error("error parsing form", "error", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		if r.Form.Get("name") == "" {
			http.Error(w, "Missing required field: name", http.StatusBadRequest)
			return
		}
		creds.Name = r.Form.Get("name")
	}

	if err := users.RemoveKey(creds, c); err != nil {
		c.Logger.Error("error removing key", "error", err)
		http.Error(w, "Error removing key", http.StatusInternalServerError)
		return
	}

	c.Logger.Info("api key removed: " + creds.Name)
	if _, err := w.Write([]byte("Success\n")); err != nil {
		c.Logger.Error("error writing success response", "error", err)
	}
}
