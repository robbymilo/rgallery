package queries

import (
	"context"
	"fmt"

	"github.com/robbymilo/rgallery/pkg/database"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// GetTags returns all tags.
func GetTags(group, direction string, c Conf) (Subjects, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
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

	query := fmt.Sprintf(`SELECT key, value FROM tags ORDER BY key %s`, direction)

	stmt, err := conn.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing SELECT statement: %v", err)
	}

	var result Subjects
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, fmt.Errorf("error stepping through result set: %v", err)
		}
		if !hasRow {
			break
		}

		result = append(
			result,
			Subject{
				Key:   stmt.ColumnText(0),
				Value: stmt.ColumnText(1),
			})

	}

	err = stmt.Finalize()
	if err != nil {
		return nil, fmt.Errorf("error finalizing statement: %v", err)
	}

	return result, err

}

// GetTotalTags returns the number of tags.
func GetTotalTags(c Conf) (int, error) {
	pool, err := sqlitex.NewPool(database.NewSqlConnectionString(c), sqlitex.PoolOptions{
		Flags:    sqlite.OpenReadOnly,
		PoolSize: 1,
	})
	if err != nil {
		return 0, fmt.Errorf("error opening sqlite db pool: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			c.Logger.Error("error closing pool", "err", err)
		}
	}()

	conn, err := pool.Take(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to take connection from pool: %w", err)
	}
	defer pool.Put(conn)

	query := `SELECT COUNT(*) FROM tags`
	stmt, err := conn.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf("error preparing query: %v", err)
	}
	defer func() {
		if finalizeErr := stmt.Finalize(); finalizeErr != nil {
			err = fmt.Errorf("error finalizing statement: %v", finalizeErr)
		}
	}()

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
