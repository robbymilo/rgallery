package server

import (
	"encoding/json"
	"net/http"

	"github.com/robbymilo/rgallery/pkg/users"
)

// SignUp handles a post request to create a new user.
func SignUp(w http.ResponseWriter, r *http.Request) {
	params := r.Context().Value(ParamsKey{}).(FilterParams)
	c := r.Context().Value(ConfigKey{}).(Conf)

	creds := &UserCredentials{}

	if params.Json {
		// Keep password input separate from credentials, whose password must
		// never be serialized into responses.
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			c.Logger.Error("error decoding json", "error", err)
			return
		}
		creds.Username, creds.Password, creds.Role = input.Username, input.Password, input.Role
	} else {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			c.Logger.Error("error parsing form", "error", err)
			return
		}
		// Defensive checks for form fields
		if r.Form.Get("username") == "" || r.Form.Get("password") == "" || r.Form.Get("role") == "" {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}
		creds.Username = r.Form.Get("username")
		creds.Password = r.Form.Get("password")
		creds.Role = r.Form.Get("role")
	}

	if err := users.ValidateCredentials(*creds); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := users.AddUser(*creds, c); err != nil {
		c.Logger.Error("error adding user", "error", err)
		http.Error(w, "Error adding user", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte("Success\n")); err != nil {
		c.Logger.Error("error writing success response", "error", err)
	}
}
