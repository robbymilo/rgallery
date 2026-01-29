package server

import (
	"net/http"

	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/robbymilo/rgallery/pkg/users"
)

type UserKey = types.UserKey

// ServeAdmin serves the admin page.
func ServeAdmin(w http.ResponseWriter, r *http.Request, c Conf) {
	var keys []users.ApiCredentials
	var err error
	keys, err = users.GetKeyNames(c)
	if err != nil {
		c.Logger.Error("error listing keys", "error", err)
		return
	}

	users, err := users.ListUsers(c)
	if err != nil {
		c.Logger.Error("error listing users", "error", err)
	}

	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	response := ResponseAdmin{
		HideNavFooter: false,
		HideAuth:      c.DisableAuth,
		Keys:          keys,
		Users:         users,
		UserName:      user.UserName,
		UserRole:      user.UserRole,
		Meta:          c.Meta,
	}

	w.Header().Set("Cache-Control", "private, max-age=0, must-revalidate")

	err = render.RenderJson(w, r, response)
	if err != nil {
		c.Logger.Error("error rendering admin response", "error", err)
	}
}
