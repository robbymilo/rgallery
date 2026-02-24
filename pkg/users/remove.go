package users

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/sessions"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// RemoveUser removes a user from the database.
func RemoveUser(creds *UserCredentials, conn *sqlite.Conn) error {
	sessions.DeleteUserSessions(creds.Username)

	stmt, err := conn.Prepare("DELETE FROM users WHERE username = ?")
	if err != nil {
		return fmt.Errorf("error preparing delete statement: %w", err)
	}

	stmt.BindText(1, creds.Username)

	_, err = stmt.Step()
	if err != nil {
		return fmt.Errorf("error executing delete statement: %w", err)
	}

	if finalizeErr := stmt.Finalize(); finalizeErr != nil {
		return fmt.Errorf("error finalizing insert statement: %v", finalizeErr)
	}

	return nil
}

func RemoveUserConnect(creds *UserCredentials, c Conf) error {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadWrite,
		PoolSize: 1,
	})
	if err != nil {
		return fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool: %v", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	err = RemoveUser(creds, conn)
	if err != nil {
		return fmt.Errorf("error removing user: %v", err)
	}

	return nil
}
