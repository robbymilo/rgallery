package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"github.com/robbymilo/rgallery/pkg/types"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type ApiCredentials = types.ApiCredentials

// GetAllKeys returns a list of all hashed keys.
func GetAllKeys(c Conf) ([]ApiCredentials, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool: %v", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := "SELECT key FROM keys"
	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing query: %v", err)
	}
	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	var keys []ApiCredentials
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through query results: %v", err)
		}
		if !hasRow {
			break
		}

		key := stmt.ColumnText(0)
		keys = append(keys, ApiCredentials{
			Key: key,
		})
	}

	return keys, nil
}
