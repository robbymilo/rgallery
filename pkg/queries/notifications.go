package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/notify"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// Notify sends a notification to all users and subscribers
func Notify(c Conf, message string, status string) error {
	// Always persist and broadcast every notification
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()
	return notify.SendToAllUsers(conn, message)
}

// GetNotifications retrievesall  notifications
func GetNotifications(c Conf, username string) ([]notify.Notification, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 5,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	var notifications []notify.Notification
	query := `SELECT n.id, ?, n.message, nd.dismissed, n.created_at
			  FROM notifications n
			  LEFT JOIN notifications_dismissed nd ON nd.notification_id = n.id AND nd.username = ?
			  WHERE IFNULL(nd.dismissed, 0) = 0
			  ORDER BY n.created_at DESC`
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{username, username},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			n := notify.Notification{
				ID:        stmt.ColumnInt64(0),
				Username:  stmt.ColumnText(1),
				Message:   stmt.ColumnText(2),
				IsRead:    stmt.ColumnBool(3),
				CreatedAt: stmt.ColumnText(4),
			}
			notifications = append(notifications, n)
			return nil
		},
	})

	return notifications, err

}

// MarkNotificationRead marks a notification as read for a user
func MarkNotificationRead(c Conf, id int64, username string) error {
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	err = sqlitex.Execute(conn, "INSERT OR REPLACE INTO notifications_dismissed (notification_id, username, dismissed, dismissed_at) VALUES (?, ?, 1, CURRENT_TIMESTAMP)", &sqlitex.ExecOptions{
		Args: []interface{}{id, username},
	})

	return err
}

// ClearAllNotifications marks all notifications as read for a user
func ClearAllNotifications(c Conf, username string) error {
	conn, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	if err != nil {
		c.Logger.Error("error opening db", "err", err)
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Logger.Error("conn.Close error", "err", err)
		}
	}()

	err = sqlitex.Execute(conn, "INSERT OR REPLACE INTO notifications_dismissed (notification_id, username, dismissed, dismissed_at) SELECT id, ?, 1, CURRENT_TIMESTAMP FROM notifications", &sqlitex.ExecOptions{
		Args: []interface{}{username},
	})

	return err
}
