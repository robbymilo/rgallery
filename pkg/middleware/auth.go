package middleware

import (
	"context"
	"net/http"

	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/sessions"
	"github.com/robbymilo/rgallery/pkg/types"
	"golang.org/x/crypto/bcrypt"
)

type UserKey = types.UserKey

// Auth determines whether a user is logged in or not for a request.
func Auth(c Conf) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if !c.DisableAuth {
				api_key := r.Header.Get("api-key")

				if api_key != "" {

					// test if api key is correct
					keys, err := queries.GetAllKeys(c)
					if err != nil {
						c.Logger.Error("error getting all keys", "error", err)
					}

					// compare api key with hashed keys
					for _, storedCreds := range keys {
						err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Key), []byte(api_key))
						if err == nil {
							// api key exists
							ctx := context.WithValue(r.Context(), UserKey{}, UserKey{
								UserRole: "admin",
							})

							// session created, continue request
							next.ServeHTTP(w, r.WithContext(ctx))
						}
					}

					c.Logger.Info("incorrect api_key supplied")

					// no existing api key
					w.WriteHeader(http.StatusUnauthorized)
					return

				} else {
					// cookie + session flow
					cookie, err := r.Cookie("session")
					if err != nil {
						if err == http.ErrNoCookie {
							w.WriteHeader(http.StatusUnauthorized)
							return
						}
						c.Logger.Error("cookie error", "error", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if cookie.Value != "" {
						token := cookie.Value

						session, exists := sessions.GetSession(token)
						if !exists {
							w.WriteHeader(http.StatusUnauthorized)
							return
						}

						if session.IsExpired() {
							sessions.DeleteSession(token, c)
							w.WriteHeader(http.StatusUnauthorized)
							return
						}

						ctx := context.WithValue(r.Context(), UserKey{}, UserKey{
							UserRole: session.Role,
							UserName: session.UserName,
						})

						// session created, continue request
						next.ServeHTTP(w, r.WithContext(ctx))
					}
				}

			} else {

				// auth disabled
				next.ServeHTTP(w, r)

			}

		})
	}
}
