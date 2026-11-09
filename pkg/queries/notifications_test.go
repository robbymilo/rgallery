package queries_test

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/stretchr/testify/require"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func TestGetNotificationsPaginatesUnreadNotifications(t *testing.T) {
	c := types.Conf{Data: t.TempDir(), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	database.CreateDB(c)
	db, err := sqlite.OpenConn(database.NewSqlConnectionString(c), sqlite.OpenReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for id := 1; id <= 5; id++ {
		err = sqlitex.Execute(db, `INSERT INTO notifications (id, message, created_at) VALUES (?, ?, ?)`, &sqlitex.ExecOptions{
			Args: []any{id, "message", fmt.Sprintf("2025-01-01T00:00:0%dZ", id)},
		})
		require.NoError(t, err)
	}
	err = sqlitex.Execute(db, `INSERT INTO notifications_dismissed (notification_id, username, dismissed) VALUES (5, 'alice', 1)`, nil)
	require.NoError(t, err)

	first, err := queries.GetNotifications(c, "alice", 2, 0)
	require.NoError(t, err)
	require.Equal(t, 4, first.Total)
	require.Equal(t, []int64{4, 3}, []int64{first.Notifications[0].ID, first.Notifications[1].ID})

	second, err := queries.GetNotifications(c, "alice", 2, 2)
	require.NoError(t, err)
	require.Equal(t, []int64{2, 1}, []int64{second.Notifications[0].ID, second.Notifications[1].ID})
}
