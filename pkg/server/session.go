package server

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/robbymilo/rgallery/pkg/render"
	"github.com/robbymilo/rgallery/pkg/sessions"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/robbymilo/rgallery/pkg/users"
	"golang.org/x/crypto/bcrypt"
)

type ResponsAuth = types.ResponsAuth
type UserCredentials = types.UserCredentials
type ApiCredentials = types.ApiCredentials

// SignIn handles a post request to create a session for an existing user.
func SignIn(w http.ResponseWriter, r *http.Request, c Conf) error {

	// get sign in data as JSON
	creds := &UserCredentials{}
	err := r.ParseForm()
	if err != nil {
		return fmt.Errorf("error parsing form: %v", err)
	}

	creds.Username = r.Form["username"][0]
	creds.Password = r.Form["password"][0]

	// create error URL
	errorUrl, _ := url.Parse("/signin")
	errorParams := url.Values{}
	errorParams.Add("error", "login")
	errorUrl.RawQuery = errorParams.Encode()

	storedCreds, err := users.GetUser(creds, c)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("error checking for user: %v", err)
	}

	// compare password with hash
	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if _, err := w.Write([]byte(`Invalid credentials`)); err != nil {
			c.Logger.Error("failed to write response: %v", "err", err)
		}
		return fmt.Errorf("error comparing supplied password with hash: %v", err)
	}

	token := uuid.NewString()
	expires := time.Now().Add(time.Hour * time.Duration(c.SessionLength) * 24)

	err = sessions.CreateSession(storedCreds.Username, storedCreds.Role, token, expires, c)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`Success`)); err != nil {
			c.Logger.Error("failed to write response: %v", "err", err)
		}
		return fmt.Errorf("error generating session: %v", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)

	return nil

}

// ServeLogOut handles a get request to remove a user's session.
func ServeLogOut(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	token := cookie.Value

	c := r.Context().Value(ConfigKey{}).(Conf)
	sessions.DeleteSession(token, c)

	// set empty cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`Logged out`)); err != nil {
		c.Logger.Error("failed to write response: %v", "err", err)
	}

}

// ServeSignIn serves the sign in page.
func ServeSignIn(w http.ResponseWriter, r *http.Request) {
	response := ResponsAuth{
		Section: "auth",
	}
	_ = render.Render(w, r, response, "signin")
}
