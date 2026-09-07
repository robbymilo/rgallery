package rgallery_test

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/middleware"
	"github.com/robbymilo/rgallery/pkg/rgallery"
	"github.com/robbymilo/rgallery/pkg/sessions"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/robbymilo/rgallery/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// These regression tests exercise authentication through the real router.
// Do not parallelize: production sessions are stored in a process-global map.
type authFixture struct {
	t      *testing.T
	c      types.Conf
	db     *sql.DB
	router http.Handler
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	root := t.TempDir()
	c := types.Conf{Dev: true, Data: filepath.Join(root, "data"), Media: filepath.Join(root, "media"), Cache: filepath.Join(root, "cache"), SessionLength: 1, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	database.CreateDB(c)
	require.NoError(t, users.InitUser(c))
	db, err := sql.Open("sqlite", database.NewSqlConnectionString(c))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return &authFixture{t: t, c: c, db: db, router: rgallery.SetupRouter(c, cache.New(-1, -1), "test", "test")}
}

func (f *authFixture) request(method, path, body, contentType string, cookie *http.Cookie) *httptest.ResponseRecorder {
	f.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	// Also clean up sessions produced by unexpected successful logins.
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.Value != "" {
			token := c.Value
			f.t.Cleanup(func() { sessions.DeleteSession(token, f.c) })
		}
	}
	return w
}

func (f *authFixture) signIn(username, password string) *httptest.ResponseRecorder {
	return f.request("POST", "/api/signin", url.Values{"username": {username}, "password": {password}}.Encode(), "application/x-www-form-urlencoded", nil)
}

func (f *authFixture) login(username, password string) *http.Cookie {
	f.t.Helper()
	w := f.signIn(username, password)
	require.Equal(f.t, http.StatusFound, w.Code, w.Body.String())
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	f.t.Fatal("successful login did not issue a session cookie")
	return nil
}

func (f *authFixture) seedViewer() {
	f.t.Helper()
	// Seed directly to keep fixture setup independent of user creation.
	hash, err := bcrypt.GenerateFromPassword([]byte("viewer-password"), bcrypt.MinCost)
	require.NoError(f.t, err)
	_, err = f.db.Exec("INSERT INTO users (username, password, role) VALUES (?, ?, ?)", "reader", string(hash), "viewer")
	require.NoError(f.t, err)
}

func (f *authFixture) count(query string, args ...any) int {
	f.t.Helper()
	var n int
	require.NoError(f.t, f.db.QueryRow(query, args...).Scan(&n))
	return n
}

func signupBody(encoding, username, password, role string) (string, string) {
	if encoding == "json" {
		b, _ := json.Marshal(map[string]string{"username": username, "password": password, "role": role})
		return string(b), "application/json"
	}
	return url.Values{"username": {username}, "password": {password}, "role": {role}}.Encode(), "application/x-www-form-urlencoded"
}

// Both anonymous clients and viewers must be unable to create users.
func TestAuthUserCreationRequiresAdministrator(t *testing.T) {
	for _, actor := range []string{"anonymous", "viewer"} {
		for _, encoding := range []string{"form", "json"} {
			for _, role := range []string{"admin", "viewer"} {
				t.Run(actor+"/"+encoding+"/"+role, func(t *testing.T) {
					f := newAuthFixture(t)
					f.seedViewer()
					var cookie *http.Cookie
					want := http.StatusUnauthorized
					if actor == "viewer" {
						cookie = f.login("reader", "viewer-password")
						want = http.StatusForbidden
					}
					body, ct := signupBody(encoding, "intruder", "secret", role)
					w := f.request("POST", "/api/user/add", body, ct, cookie)
					assert.Equal(t, want, w.Code, w.Body.String())
					assert.Equal(t, 0, f.count("SELECT count(*) FROM users WHERE username = ?", "intruder"), "unauthorized request inserted a user")
					assert.Equal(t, 1, f.count("SELECT count(*) FROM users WHERE username = 'admin'"), "unauthorized request removed default admin")
				})
			}
		}
	}
}

// Admin data and API-key mutations require administrator privileges.
func TestAuthAdminEndpoints(t *testing.T) {
	for _, actor := range []string{"anonymous", "viewer", "admin"} {
		for _, endpoint := range []struct{ name, method, path, body string }{
			{"read_admin", "GET", "/api/admin?format=json", ""},
			{"create_key", "POST", "/api/keys/create", "name=new-key"},
			{"delete_key", "POST", "/api/keys/delete", "name=existing-key"},
		} {
			t.Run(actor+"/"+endpoint.name, func(t *testing.T) {
				f := newAuthFixture(t)
				f.seedViewer()
				_, err := users.AddKey(&types.ApiCredentials{Name: "existing-key"}, f.c)
				require.NoError(t, err)
				var cookie *http.Cookie
				want := http.StatusUnauthorized
				if actor == "viewer" {
					cookie = f.login("reader", "viewer-password")
					want = http.StatusForbidden
				}
				if actor == "admin" {
					cookie = f.login("admin", "admin")
					want = http.StatusOK
				}
				w := f.request(endpoint.method, endpoint.path, endpoint.body, "application/x-www-form-urlencoded", cookie)
				assert.Equal(t, want, w.Code, w.Body.String())
				if actor != "admin" {
					assert.Equal(t, 1, f.count("SELECT count(*) FROM keys WHERE name = 'existing-key'"))
					assert.Equal(t, 0, f.count("SELECT count(*) FROM keys WHERE name = 'new-key'"))
					if endpoint.name == "read_admin" {
						assert.NotContains(t, w.Body.String(), "existing-key")
						assert.NotContains(t, w.Body.String(), `"Users"`)
					}
				} else if endpoint.name == "create_key" {
					assert.Equal(t, 1, f.count("SELECT count(*) FROM keys WHERE name = 'new-key'"))
				} else if endpoint.name == "delete_key" {
					assert.Equal(t, 0, f.count("SELECT count(*) FROM keys WHERE name = 'existing-key'"))
				}
			})
		}
	}
}

// Only a replacement administrator should retire admin/admin.
func TestAuthDefaultAdminReplacement(t *testing.T) {
	for _, role := range []string{"viewer", "admin"} {
		t.Run(role, func(t *testing.T) {
			f := newAuthFixture(t)
			first, second := f.login("admin", "admin"), f.login("admin", "admin")
			body, ct := signupBody("form", "replacement", "replacement-password", role)
			require.Equal(t, http.StatusOK, f.request("POST", "/api/user/add", body, ct, first).Code)
			wantCount, wantSession, wantLogin := 1, http.StatusOK, http.StatusFound
			if role == "admin" {
				wantCount, wantSession, wantLogin = 0, http.StatusUnauthorized, http.StatusUnauthorized
			}
			assert.Equal(t, wantCount, f.count("SELECT count(*) FROM users WHERE username = 'admin'"))
			assert.Positive(t, f.count("SELECT count(*) FROM users WHERE role = 'admin'"), "must retain an administrator")
			for _, cookie := range []*http.Cookie{first, second} {
				assert.Equal(t, wantSession, f.request("GET", "/api/profile", "", "", cookie).Code)
			}
			assert.Equal(t, wantLogin, f.signIn("admin", "admin").Code)
			assert.Equal(t, http.StatusFound, f.signIn("replacement", "replacement-password").Code)
			// Startup must not recreate the default credentials after replacement.
			require.NoError(t, users.InitUser(f.c))
			assert.Equal(t, wantCount, f.count("SELECT count(*) FROM users WHERE username = 'admin'"))
		})
	}
}

// Malformed client input must be a 400, never a panic or 500.
func TestAuthSignInMalformedInput(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"missing_both", ""}, {"missing_username", "password=admin"}, {"missing_password", "username=admin"},
		{"empty_username", "username=&password=admin"}, {"empty_password", "username=admin&password="},
		{"invalid_form_escape", "username=%zz&password=admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newAuthFixture(t)
			var w *httptest.ResponseRecorder
			require.NotPanics(t, func() { w = f.request("POST", "/api/signin", tc.body, "application/x-www-form-urlencoded", nil) })
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Empty(t, w.Header().Values("Set-Cookie"))
		})
	}
}

func TestAuthJSONSignupPreservesPassword(t *testing.T) {
	f := newAuthFixture(t)
	cookie := f.login("admin", "admin")
	body, ct := signupBody("json", "owner", "chosen-password", "admin")
	require.Equal(t, http.StatusOK, f.request("POST", "/api/user/add", body, ct, cookie).Code)
	assert.Equal(t, http.StatusFound, f.signIn("owner", "chosen-password").Code, "the supplied password must work")
	assert.NotEqual(t, http.StatusFound, f.signIn("owner", "").Code, "empty password must not authenticate")
}

func TestAuthSignupRejectsInvalidCredentials(t *testing.T) {
	for _, encoding := range []string{"form", "json"} {
		for _, tc := range []struct{ name, username, password, role string }{
			{"missing_username", "", "secret", "admin"}, {"missing_password", "owner", "", "admin"},
			{"missing_role", "owner", "secret", ""}, {"unknown_role", "owner", "secret", "superuser"},
		} {
			t.Run(encoding+"/"+tc.name, func(t *testing.T) {
				f := newAuthFixture(t)
				cookie := f.login("admin", "admin")
				body, ct := signupBody(encoding, tc.username, tc.password, tc.role)
				assert.Equal(t, http.StatusBadRequest, f.request("POST", "/api/user/add", body, ct, cookie).Code)
				assert.Equal(t, 1, f.count("SELECT count(*) FROM users"))
				assert.Equal(t, 1, f.count("SELECT count(*) FROM users WHERE username = 'admin'"))
				assert.Equal(t, http.StatusOK, f.request("GET", "/api/profile", "", "", cookie).Code)
			})
		}
	}
}

func TestAuthSessionValidation(t *testing.T) {
	for _, state := range []string{"missing", "empty", "unknown", "expired", "admin", "viewer"} {
		t.Run(state, func(t *testing.T) {
			f := newAuthFixture(t)
			var cookie *http.Cookie
			want := http.StatusUnauthorized
			switch state {
			case "empty":
				cookie = &http.Cookie{Name: "session", Value: ""}
			case "unknown":
				cookie = &http.Cookie{Name: "session", Value: uuid.NewString()}
			case "expired":
				token := uuid.NewString()
				require.NoError(t, sessions.CreateSession("admin", "admin", token, time.Now().Add(-time.Hour), f.c))
				t.Cleanup(func() { sessions.DeleteSession(token, f.c) })
				cookie = &http.Cookie{Name: "session", Value: token}
			case "admin":
				cookie = f.login("admin", "admin")
				want = http.StatusOK
			case "viewer":
				f.seedViewer()
				cookie = f.login("reader", "viewer-password")
				want = http.StatusOK
			}
			assert.Equal(t, want, f.request("GET", "/api/profile", "", "", cookie).Code)
		})
	}
}

func TestAuthLogoutRevokesOnlyCurrentSession(t *testing.T) {
	f := newAuthFixture(t)
	first, second := f.login("admin", "admin"), f.login("admin", "admin")
	assert.Equal(t, "/", first.Path, "login cookie must cover all protected routes")
	w := f.request("POST", "/api/logout", "", "", first)
	require.Equal(t, http.StatusOK, w.Code)
	cleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session" {
			cleared = true
			assert.Equal(t, first.Path, cookie.Path, "logout must clear the same cookie path")
			assert.Equal(t, -1, cookie.MaxAge)
		}
	}
	assert.True(t, cleared, "logout must expire the browser cookie")
	assert.Equal(t, http.StatusUnauthorized, f.request("GET", "/api/profile", "", "", first).Code)
	assert.Equal(t, http.StatusOK, f.request("GET", "/api/profile", "", "", second).Code)
}

type authStatusRecorder struct {
	*httptest.ResponseRecorder
	statuses []int
}

func (w *authStatusRecorder) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
	w.ResponseRecorder.WriteHeader(status)
}

func TestAuthAPIKey(t *testing.T) {
	f := newAuthFixture(t)
	key, err := users.AddKey(&types.ApiCredentials{Name: "test-key"}, f.c)
	require.NoError(t, err)
	for _, valid := range []bool{true, false} {
		name := "invalid"
		if valid {
			name = "valid"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			handler := middleware.Auth(f.c)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				user, ok := r.Context().Value(types.UserKey{}).(types.UserKey)
				assert.True(t, ok)
				assert.Equal(t, "admin", user.UserRole)
				w.WriteHeader(http.StatusNoContent)
			}))
			r := httptest.NewRequest("GET", "/api/profile", nil)
			r.Header.Set("api-key", "invalid-key")
			if valid {
				r.Header.Set("api-key", key)
			}
			w := &authStatusRecorder{ResponseRecorder: httptest.NewRecorder()}
			handler.ServeHTTP(w, r)
			if valid {
				assert.Equal(t, 1, calls)
				assert.Equal(t, []int{http.StatusNoContent}, w.statuses, "successful authentication must not write a later rejection")
			} else {
				assert.Zero(t, calls)
				assert.Equal(t, []int{http.StatusUnauthorized}, w.statuses)
			}
		})
	}
}

func TestAuthInvalidCredentialsDoNotCreateSession(t *testing.T) {
	for _, username := range []string{"admin", "nonexistent"} {
		t.Run(username, func(t *testing.T) {
			f := newAuthFixture(t)
			w := f.signIn(username, "wrong-password")
			assert.Equal(t, http.StatusUnauthorized, w.Code)
			assert.Empty(t, w.Header().Values("Set-Cookie"))
		})
	}
}
