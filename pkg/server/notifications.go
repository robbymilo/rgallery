package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/notify"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/types"
)

// ServeNotifications returns a list of notifications
func ServeNotifications(w http.ResponseWriter, r *http.Request) {
	c := r.Context().Value(ConfigKey{}).(Conf)
	userKey := r.Context().Value(types.UserKey{})
	var username string
	if user, ok := userKey.(types.UserKey); ok {
		username = user.UserName
	}

	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notifications, err := queries.GetNotifications(c, username)
	if err != nil {
		c.Logger.Error("error querying notifications", "error", err)
		http.Error(w, "Error getting notifications", http.StatusInternalServerError)
		return
	}

	if notifications == nil {
		notifications = []notify.Notification{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(notifications); err != nil {
		c.Logger.Error("failed to encode notifications: %v", "err", err)
	}
}

// MarkNotificationRead marks a notification as read
func MarkNotificationRead(w http.ResponseWriter, r *http.Request, c Conf) {
	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	if c.DisableAuth || (!c.DisableAuth && user.UserRole != "admin") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	userKey := r.Context().Value(types.UserKey{})
	var username string
	if user, ok := userKey.(types.UserKey); ok {
		username = user.UserName
	}

	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = queries.MarkNotificationRead(c, id, username)
	if err != nil {
		c.Logger.Error("error updating notification", "error", err)
		http.Error(w, "Error marking notification read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ClearAllNotifications marks all notifications as read
func ClearAllNotifications(w http.ResponseWriter, r *http.Request, c Conf) {
	var user UserKey
	if r.Context().Value(UserKey{}) != nil {
		user = r.Context().Value(UserKey{}).(UserKey)
	}

	if c.DisableAuth || (!c.DisableAuth && user.UserRole != "admin") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err := queries.ClearAllNotifications(c, user.UserName)
	if err != nil {
		c.Logger.Error("error clearing notifications", "error", err)
		http.Error(w, "Error clearing notifications", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
