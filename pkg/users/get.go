package users

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func GetUser(creds *UserCredentials, c Conf) (UserCredentials, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return UserCredentials{}, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return UserCredentials{}, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := "SELECT username, password, role FROM users WHERE username=?"
	stmt, err := conn.Prepare(query)
	if err != nil {
		return UserCredentials{}, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, creds.Username)

	storedCreds := &UserCredentials{}
	hasRow, err := stmt.Step()
	if err != nil {
		return UserCredentials{}, fmt.Errorf("error executing query: %v", err)
	}
	if hasRow {
		storedCreds = &UserCredentials{
			Username: stmt.ColumnText(0),
			Password: stmt.ColumnText(1),
			Role:     stmt.ColumnText(2),
		}
	}

	return *storedCreds, nil
}
