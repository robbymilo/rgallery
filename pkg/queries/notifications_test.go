package queries_test

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/queries"
	"github.com/robbymilo/rgallery/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestGetNotificationsPaginatesUnreadNotifications(t *testing.T) {
	c := types.Conf{Data: t.TempDir(), Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	database.CreateDB(c)
	db, err := sql.Open("sqlite", database.NewSqlConnectionString(c))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	for id := 1; id <= 5; id++ {
		_, err = db.Exec(`INSERT INTO notifications (id, message, created_at) VALUES (?, ?, ?)`, id, "message", fmt.Sprintf("2025-01-01T00:00:0%dZ", id))
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO notifications_dismissed (notification_id, username, dismissed) VALUES (5, 'alice', 1)`)
	require.NoError(t, err)

	first, err := queries.GetNotifications(c, "alice", 2, 0)
	require.NoError(t, err)
	require.Equal(t, 4, first.Total)
	require.Equal(t, []int64{4, 3}, []int64{first.Notifications[0].ID, first.Notifications[1].ID})

	second, err := queries.GetNotifications(c, "alice", 2, 2)
	require.NoError(t, err)
	require.Equal(t, []int64{2, 1}, []int64{second.Notifications[0].ID, second.Notifications[1].ID})
}
