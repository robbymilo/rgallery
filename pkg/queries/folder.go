package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetFolder returns the media items in a folder.
func GetFolder(group, name string, pageSize, offset int, params FilterParams, c Conf) ([]Media, error) {
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

	query := fmt.Sprintf(`SELECT DISTINCT %s FROM media WHERE %s =? AND DATE != '0001-01-01T00:00:00.000Z' GROUP BY date ORDER BY date %s LIMIT %d OFFSET %d`, columns, group, params.Direction, pageSize, offset)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	stmt.BindText(1, name)

	result, err := parseMediaRows(stmt, c)
	if err != nil {
		return nil, err
	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return result, err
}

// GetTotalOfFolder returns the total of media items in a folder.
func GetTotalOfFolder(group, name string, c Conf) (int, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("pool.Close error", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := fmt.Sprintf(`SELECT count(distinct date) FROM media WHERE %s =? AND DATE != ?`, group)
	stmt, err := conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}

	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

	stmt.BindText(1, name)
	stmt.BindText(2, "0001-01-01T00:00:00.000Z")

	var total int
	hasRow, err := stmt.Step()
	if err != nil {
		return 0, fmt.Errorf("error executing query: %v", err)
	}
	if hasRow {
		total = int(stmt.ColumnInt64(0))
	}

	return total, nil
}
