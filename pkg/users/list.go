package users

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// ListUsers lists all users.
func ListUsers(c Conf) ([]User, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	stmt, err := conn.Prepare("SELECT username, role FROM users")
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	users := make([]User, 0)
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		users = append(users, User{
			Username: stmt.ColumnText(0),
			Role:     stmt.ColumnText(1),
		})
	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return users, nil
}
