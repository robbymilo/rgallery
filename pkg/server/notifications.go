package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/robbymilo/rgallery/pkg/notify"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/types"
)

const (
	defaultNotificationsPageSize = 20
	maxNotificationsPageSize     = 100
)

type notificationsResponse struct {
	Notifications []notify.Notification `json:"notifications"`
	Meta          notificationsMeta     `json:"meta"`
}

type notificationsMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

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

	page, pageSize, err := notificationPagination(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := queries.GetNotifications(c, username, pageSize, (page-1)*pageSize)
	if err != nil {
		c.Logger.Error("error querying notifications", "error", err)
		http.Error(w, "Error getting notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := notificationsResponse{
		Notifications: result.Notifications,
		Meta: notificationsMeta{
			Page:     page,
			PageSize: pageSize,
			Total:    result.Total,
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		c.Logger.Error("failed to encode notifications: %v", "err", err)
	}
}

func notificationPagination(r *http.Request) (page, pageSize int, err error) {
	page = 1
	pageSize = defaultNotificationsPageSize

	if value := r.URL.Query().Get("page"); value != "" {
		page, err = strconv.Atoi(value)
		if err != nil || page < 1 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
	}
	if value := r.URL.Query().Get("pageSize"); value != "" {
		pageSize, err = strconv.Atoi(value)
		if err != nil || pageSize < 1 || pageSize > maxNotificationsPageSize {
			return 0, 0, fmt.Errorf("pageSize must be between 1 and %d", maxNotificationsPageSize)
		}
	}
	return page, pageSize, nil
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
